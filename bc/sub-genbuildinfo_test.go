//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrmxf/util/kfg"
	. "github.com/smartystreets/goconvey/convey"
)

// inRepo runs f with the working directory set to a fresh repo that has one
// commit tagged v1.2.3 and one more commit after it.
func inRepo(t *testing.T, f func(git func(...string))) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "one")
	git("tag", "-a", "v1.2.3", "-m", "v1.2.3")
	saved, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(saved)
	f(git)
}

func runGen(args ...string) (string, error) {
	var out bytes.Buffer
	genFormat, genName, genTitle = "json", "", ""
	// Command is shared with other tests: leave it as found
	defer func() {
		Command.SetArgs(nil)
		Command.SetOut(nil)
		Command.SetErr(nil)
		genFormat = "json"
	}()
	Command.SetOut(&out)
	Command.SetErr(&bytes.Buffer{})
	Command.SetArgs(append([]string{"gen", "buildinfo"}, args...))
	err := Command.Execute()
	return strings.TrimSpace(out.String()), err
}

func TestGenBuildinfo(t *testing.T) {
	Convey("clog BC gen buildinfo", t, func() {
		inRepo(t, func(git func(...string)) {
			Convey("on the release tag, prod is allowed and the version is the tag", func() {
				out, err := runGen("prod", "--format", "version")
				So(err, ShouldBeNil)
				So(out, ShouldEqual, "v1.2.3")
				out, _ = runGen("prod", "--format", "ldflags")
				So(out, ShouldStartWith, "-X github.com/mrmxf/util/buildinfo.SemVerJSON='{\"build\":\"prod\"")
			})
			Convey("after the tag, prod is refused and dev is +dev.1", func() {
				git("commit", "-q", "--allow-empty", "-m", "two")
				_, err := runGen("prod")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "1 commits after v1.2.3")
				out, err := runGen("--format", "version")
				So(err, ShouldBeNil)
				So(out, ShouldStartWith, "v1.2.3+dev.1.g")
				out, _ = runGen("--format", "docker-tag")
				So(out, ShouldStartWith, "v1.2.3-dev.1.g")
				out, _ = runGen("--format", "env")
				So(out, ShouldContainSubstring, "BUILD_MODE=dev")
				So(GitTagRef(), ShouldEqual, "v1.2.3")
			})
			Convey("an unknown format is an error", func() {
				_, err := runGen("--format", "yaml")
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestFlowIdentity(t *testing.T) {
	Convey("the flow banner identity comes from git and the mode, not releases.yaml", t, func() {
		saved, savedRel := IsProduction, Releases
		defer func() { IsProduction, Releases = saved, savedRel }()
		Releases = func() []kfg.AppRelease { return nil } // no releases.yaml at all
		inRepo(t, func(git func(...string)) {
			IsProduction = func() bool { return false }
			v, m := flowIdentity()
			So(v, ShouldEqual, "v1.2.3")
			So(m, ShouldEqual, "dev")
			IsProduction = func() bool { return true }
			_, m = flowIdentity()
			So(m, ShouldEqual, "prod")
		})
	})
}
