//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const sha = "34103be0123456789abcdef0123456789abcdef0"

func TestReleaseTags(t *testing.T) {
	Convey("IsReleaseTag accepts vX.Y.Z / X.Y.Z only", t, func() {
		for _, ok := range []string{"v0.11.12", "3.5.1", "v10.0.100"} {
			So(IsReleaseTag(ok), ShouldBeTrue)
		}
		for _, no := range []string{"v3.5.0-dev", "v1.0.0-rc1", "v1.2", "release-1.2.3", "v1.2.3+meta", "ci/v0.12.1", ""} {
			So(IsReleaseTag(no), ShouldBeFalse)
		}
	})
}

func TestVersion(t *testing.T) {
	Convey("GitState.Version", t, func() {
		So(GitState{Hash: sha, ExactTag: "v1.2.3", Nearest: "v1.2.3"}.Version(), ShouldEqual, "v1.2.3")
		So(GitState{Hash: sha, ExactTag: "v1.2.3", Nearest: "v1.2.3", Dirty: true}.Version(), ShouldEqual, "v1.2.3+dirty")
		So(GitState{Hash: sha, Nearest: "v1.2.3", Ahead: 3}.Version(), ShouldEqual, "v1.2.3+dev.3.g34103be")
		So(GitState{Hash: sha, Nearest: "v1.2.3", Ahead: 3, Dirty: true}.Version(), ShouldEqual, "v1.2.3+dev.3.g34103be.dirty")
		So(GitState{Hash: sha, Ahead: 41}.Version(), ShouldEqual, "v0.0.0+dev.41.g34103be")
		So(DockerTag("v1.2.3+dev.3.g34103be"), ShouldEqual, "v1.2.3-dev.3.g34103be")
	})
}

func TestGenerate(t *testing.T) {
	onTag := GitState{Hash: sha, CommitDate: "2026-09-19", ExactTag: "v1.2.3", Nearest: "v1.2.3"}
	after := GitState{Hash: sha, CommitDate: "2026-09-19", Nearest: "v1.2.3", Ahead: 3}

	Convey("Generate", t, func() {
		Convey("prod on a clean release tag", func() {
			d, err := Generate(onTag, BuildOptions{Mode: "prod", Name: "clog", Title: "Command Line Of Go"})
			So(err, ShouldBeNil)
			So(d.Build, ShouldEqual, "prod")
			So(d.Tag, ShouldEqual, "v1.2.3")
			So(d.AppTitle, ShouldEqual, "Command_Line_Of_Go")
			So(d.Date, ShouldEqual, "2026-09-19")
		})
		Convey("prod refuses a commit that is not a release tag, and says where it is", func() {
			_, err := Generate(after, BuildOptions{Mode: "prod"})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "3 commits after v1.2.3")
		})
		Convey("prod refuses a dirty tree", func() {
			dirty := onTag
			dirty.Dirty = true
			_, err := Generate(dirty, BuildOptions{Mode: "prod"})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "clean working tree")
		})
		Convey("dev: the version carries the dev-ness, the branch is only a suffix on a tag", func() {
			d, _ := Generate(after, BuildOptions{Branch: "main"})
			So(d.Build, ShouldEqual, "dev")
			So(d.Suffix, ShouldBeEmpty)
			d, _ = Generate(onTag, BuildOptions{Mode: "dev", Branch: "main"})
			So(d.Suffix, ShouldEqual, "main")
		})
		Convey("an unknown mode is an error", func() {
			_, err := Generate(onTag, BuildOptions{Mode: "stage"})
			So(err, ShouldNotBeNil)
		})
		Convey("the stamp shows as the version, with no -dev added to a +dev version", func() {
			d, _ := Generate(after, BuildOptions{Branch: "main", Name: "clog", Title: "clog"})
			info, isProd, err := ParseLinkerJSON(d.JSON())
			So(err, ShouldBeNil)
			So(isProd, ShouldBeFalse)
			So(info.Short, ShouldEqual, "v1.2.3+dev.3.g34103be")
			d, _ = Generate(onTag, BuildOptions{Mode: "prod", Name: "clog", Title: "clog"})
			info, isProd, _ = ParseLinkerJSON(d.JSON())
			So(isProd, ShouldBeTrue)
			So(info.Short, ShouldEqual, "v1.2.3")
		})
		Convey("Ldflags targets the variable this package reads", func() {
			So(LinkerPath(), ShouldEqual, "github.com/mrmxf/util/buildinfo.SemVerJSON")
			d, _ := Generate(onTag, BuildOptions{Mode: "prod"})
			So(d.Ldflags(), ShouldStartWith, "-X github.com/mrmxf/util/buildinfo.SemVerJSON='{")
		})
	})
}

// repo builds a throwaway repository. at runs git with a fixed commit/tagger
// date so "newest tag" is deterministic.
type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	r := &repo{t: t, dir: t.TempDir()}
	r.git("", "init", "-q", "-b", "main")
	return r
}

func (r *repo) git(date string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if date != "" {
		cmd.Env = append(cmd.Env, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *repo) commit(date, msg string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, "f"), []byte(msg), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.git(date, "add", "f")
	r.git(date, "commit", "-q", "-m", msg)
}

func TestReadGitState(t *testing.T) {
	Convey("ReadGitState and LatestReleaseTag on a real repository", t, func() {
		r := newRepo(t)
		r.commit("2026-01-01T10:00:00", "one")

		Convey("no release tag: v0.0.0 plus the commit count", func() {
			g, err := ReadGitState(r.dir)
			So(err, ShouldBeNil)
			So(g.Version(), ShouldStartWith, "v0.0.0+dev.1.g")
			_, err = LatestReleaseTag(r.dir, "")
			So(err, ShouldNotBeNil)
		})

		r.git("2026-01-02T10:00:00", "tag", "-a", "v1.0.0", "-m", "v1.0.0")
		r.commit("2026-01-03T10:00:00", "two")
		r.git("2026-01-04T10:00:00", "tag", "-a", "v1.1.0", "-m", "v1.1.0")
		r.git("2026-01-05T10:00:00", "tag", "-a", "v1.2.0-rc1", "-m", "not a release")
		r.git("2026-01-05T10:00:00", "tag", "v3.5.0-dev")

		Convey("on a release tag: exact, clean, commit date", func() {
			g, err := ReadGitState(r.dir)
			So(err, ShouldBeNil)
			So(g.ExactTag, ShouldEqual, "v1.1.0")
			So(g.Version(), ShouldEqual, "v1.1.0")
			So(g.Dirty, ShouldBeFalse)
			So(g.CommitDate, ShouldEqual, "2026-01-03")
		})

		r.commit("2026-01-06T10:00:00", "three")
		r.commit("2026-01-07T10:00:00", "four")

		Convey("after a release tag: nearest release and distance, pre-releases ignored", func() {
			g, err := ReadGitState(r.dir)
			So(err, ShouldBeNil)
			So(g.ExactTag, ShouldBeEmpty)
			So(g.Version(), ShouldEqual, "v1.1.0+dev.2.g"+g.Short())
		})
		Convey("an untracked file makes it dirty", func() {
			So(os.WriteFile(filepath.Join(r.dir, "new"), []byte("x"), 0o644), ShouldBeNil)
			g, _ := ReadGitState(r.dir)
			So(g.Dirty, ShouldBeTrue)
			So(g.Version(), ShouldEndWith, ".dirty")
		})

		// a newer release on another branch, not merged into main
		r.git("", "checkout", "-q", "-b", "side")
		r.commit("2026-02-01T10:00:00", "side")
		r.git("2026-02-02T10:00:00", "tag", "-a", "v9.0.0", "-m", "v9.0.0")
		r.git("", "checkout", "-q", "main")

		Convey("LatestReleaseTag: newest by date, optionally only tags on a branch", func() {
			all, err := LatestReleaseTag(r.dir, "")
			So(err, ShouldBeNil)
			So(all, ShouldEqual, "v9.0.0")
			onMain, err := LatestReleaseTag(r.dir, "main")
			So(err, ShouldBeNil)
			So(onMain, ShouldEqual, "v1.1.0")
		})
		Convey("LatestReleaseTag refuses an option-looking branch", func() {
			_, err := LatestReleaseTag(r.dir, "--all")
			So(err, ShouldNotBeNil)
		})
	})
}
