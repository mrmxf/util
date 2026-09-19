//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mrmxf/util/slogger"
	. "github.com/smartystreets/goconvey/convey"
)

func envOf(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

var datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)

func TestParseStyle(t *testing.T) {
	Convey("ParseStyle", t, func() {
		for in, want := range map[string]slogger.SlogStyle{
			"CiWithDbgTmp":        slogger.StyleCiWithDbgTmp,
			"ciwithdbgtmp":        slogger.StyleCiWithDbgTmp,
			" PrettyWithDbgTmp ":  slogger.StylePrettyWithDbgTmp,
			"StylePretty":         slogger.StylePretty,
			"json":                slogger.StyleJSON,
			"Plain":               slogger.StylePlain,
			"CIWITHDBGTMP":        slogger.StyleCiWithDbgTmp,
			"ci-with-dbg-tmp":     slogger.StyleCiWithDbgTmp,
			"CI_WITH_DBG_TMP":     slogger.StyleCiWithDbgTmp,
			"ci with dbg tmp":     slogger.StyleCiWithDbgTmp,
			"cidbgtmp":            slogger.StyleCiWithDbgTmp,
			"pretty-with-dbg-tmp": slogger.StylePrettyWithDbgTmp,
			"dbgtmp":              slogger.StylePrettyWithDbgTmp,
			"style_json":          slogger.StyleJSON,
			"PRETTY":              slogger.StylePretty,
		} {
			got, err := slogger.ParseStyle(in)
			So(err, ShouldBeNil)
			So(got, ShouldEqual, want)
		}
		Convey("round-trips every selectable style through Name", func() {
			for _, st := range []slogger.SlogStyle{slogger.StylePlain, slogger.StylePretty, slogger.StyleJSON,
				slogger.StyleJob, slogger.StylePrettyWithDbgTmp, slogger.StyleCiWithDbgTmp} {
				got, err := slogger.ParseStyle(st.Name())
				So(err, ShouldBeNil)
				So(got, ShouldEqual, st)
			}
		})
		Convey("rejects unknown names and the ones that need parameters", func() {
			for _, in := range []string{"fancy", "Nats", "Tee", "", "---", "pretty2"} {
				_, err := slogger.ParseStyle(in)
				So(err, ShouldNotBeNil)
			}
		})
	})
}

func TestChooseStyle(t *testing.T) {
	Convey("ChooseStyle", t, func() {
		Convey("a laptop gets PrettyWithDbgTmp", func() {
			st, reason, warn := slogger.ChooseStyle(envOf(nil))
			So(st, ShouldEqual, slogger.StylePrettyWithDbgTmp)
			So(reason, ShouldEqual, "default")
			So(warn, ShouldBeEmpty)
		})
		Convey("GitLab, GitHub and CI=true get CiWithDbgTmp", func() {
			for _, env := range []map[string]string{{"GITLAB_CI": "true"}, {"GITHUB_ACTIONS": "true"}, {"CI": "true"}, {"CI": "1"}} {
				st, reason, _ := slogger.ChooseStyle(envOf(env))
				So(st, ShouldEqual, slogger.StyleCiWithDbgTmp)
				So(reason, ShouldEqual, "CI detected")
			}
		})
		Convey("CI=false is not CI", func() {
			st, _, _ := slogger.ChooseStyle(envOf(map[string]string{"CI": "false"}))
			So(st, ShouldEqual, slogger.StylePrettyWithDbgTmp)
		})
		Convey("CLOG_LOG_FORCE_STYLE beats CI detection", func() {
			st, reason, warn := slogger.ChooseStyle(envOf(map[string]string{"GITLAB_CI": "true", slogger.ForceStyleEnv: "Pretty"}))
			So(st, ShouldEqual, slogger.StylePretty)
			So(reason, ShouldContainSubstring, slogger.ForceStyleEnv)
			So(warn, ShouldBeEmpty)
		})
		Convey("a bad CLOG_LOG_FORCE_STYLE falls back to automatic, with a warning", func() {
			st, reason, warn := slogger.ChooseStyle(envOf(map[string]string{"GITLAB_CI": "true", slogger.ForceStyleEnv: "Fancy"}))
			So(st, ShouldEqual, slogger.StyleCiWithDbgTmp)
			So(reason, ShouldEqual, "CI detected")
			So(warn, ShouldContainSubstring, "Fancy")
		})
	})
}

func TestOmitTime(t *testing.T) {
	Convey("PrettyHandlerOptions.OmitTime", t, func() {
		line := func(omit bool) string {
			var buf bytes.Buffer
			h := slogger.NewPrettyHandler(&buf, &slogger.PrettyHandlerOptions{NoColor: true, OmitTime: omit})
			So(h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)), ShouldBeNil)
			return buf.String()
		}
		So(datePrefix.MatchString(line(false)), ShouldBeTrue)
		omitted := line(true)
		So(datePrefix.MatchString(omitted), ShouldBeFalse)
		So(omitted, ShouldContainSubstring, "hello")
	})
}

// captureStderr swaps os.Stderr for a pipe while build runs, so a handler that
// binds os.Stderr at construction writes into the returned buffer.
func captureStderr(build func()) string {
	saved := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	build()
	w.Close()
	os.Stderr = saved
	out, _ := io.ReadAll(r)
	return string(out)
}

// lastFileRecord returns the last JSON record in today's debug log whose msg is msg.
func lastFileRecord(msg string) map[string]any {
	f, err := os.Open(slogger.DbgTmpLogPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	var found map[string]any
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var rec map[string]any
		if json.Unmarshal(sc.Bytes(), &rec) == nil && rec["msg"] == msg {
			found = rec
		}
	}
	return found
}

func TestCiWithDbgTmpHandler(t *testing.T) {
	Convey("NewCiWithDbgTmpHandler", t, func() {
		msg := "ci-probe-" + time.Now().Format("150405.000000000")
		console := captureStderr(func() {
			h, closer, err := slogger.NewCiWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			defer closer.Close()
			So(h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)), ShouldBeNil)
		})

		Convey("the console line has no timestamp", func() {
			line := strings.TrimSpace(console)
			So(line, ShouldContainSubstring, msg)
			So(datePrefix.MatchString(stripANSI(line)), ShouldBeFalse)
		})
		Convey("the file record keeps its time and carries pid", func() {
			rec := lastFileRecord(msg)
			So(rec, ShouldNotBeNil)
			So(rec["time"], ShouldNotBeEmpty)
			So(rec["pid"], ShouldEqual, float64(os.Getpid()))
		})
	})

	Convey("NewPrettyWithDbgTmpHandler file records carry pid too", t, func() {
		msg := "pretty-probe-" + time.Now().Format("150405.000000000")
		captureStderr(func() {
			h, closer, err := slogger.NewPrettyWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			defer closer.Close()
			So(h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)), ShouldBeNil)
		})
		rec := lastFileRecord(msg)
		So(rec, ShouldNotBeNil)
		So(rec["pid"], ShouldEqual, float64(os.Getpid()))
	})
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }
