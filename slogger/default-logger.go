//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode"
)

// ForceStyleEnv overrides the automatic choice of logger style. The name says
// what it does: whoever sets it is switching off the CI / laptop detection.
//
//	CLOG_LOG_FORCE_STYLE=Pretty clog build     # console only, even in CI
const ForceStyleEnv = "CLOG_LOG_FORCE_STYLE"

// styleNames are the values ParseStyle accepts, keyed by styleKey (lowercase,
// no separators). The short display names from String() are accepted too.
// Nats and Tee are absent: they need a URL / path a style name cannot carry.
var styleNames = map[string]SlogStyle{
	"plain":            StylePlain,
	"pretty":           StylePretty,
	"json":             StyleJSON,
	"job":              StyleJob,
	"prettywithdbgtmp": StylePrettyWithDbgTmp,
	"dbgtmp":           StylePrettyWithDbgTmp,
	"ciwithdbgtmp":     StyleCiWithDbgTmp,
	"cidbgtmp":         StyleCiWithDbgTmp,
}

// styleKey folds a typed style name to its lookup key: case, spaces, dashes,
// underscores and dots are ignored, as is a leading "Style".
func styleKey(name string) string {
	key := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '-', '_', '.':
			return -1
		}
		return unicode.ToLower(r)
	}, name)
	return strings.TrimPrefix(key, "style")
}

// Name is the style's canonical name, the form ParseStyle and
// CLOG_LOG_FORCE_STYLE use (String() is padded for display).
func (s SlogStyle) Name() string {
	switch s {
	case StylePlain:
		return "Plain"
	case StylePretty:
		return "Pretty"
	case StyleJSON:
		return "JSON"
	case StyleJob:
		return "Job"
	case StyleTee:
		return "Tee"
	case StyleNats:
		return "Nats"
	case StylePrettyWithDbgTmp:
		return "PrettyWithDbgTmp"
	case StyleCiWithDbgTmp:
		return "CiWithDbgTmp"
	}
	return "unknown"
}

// ParseStyle turns a style name into a SlogStyle. It is forgiving about how
// the name is typed: "CiWithDbgTmp", "ciwithdbgtmp", "CI_WITH_DBG_TMP",
// "ci-with-dbg-tmp", "cidbgtmp" and "StyleCiWithDbgTmp" are all the same.
func ParseStyle(name string) (SlogStyle, error) {
	if st, ok := styleNames[styleKey(name)]; ok {
		return st, nil
	}
	return StylePrettyWithDbgTmp, fmt.Errorf("unknown log style %q (want %s)", name, styleList())
}

func styleList() string {
	return "Plain, Pretty, JSON, Job, PrettyWithDbgTmp or CiWithDbgTmp - any case, - or _ allowed"
}

// IsCI reports whether this process runs under a CI system: GitLab, GitHub
// Actions, or anything that follows the common CI=true convention.
func IsCI(getenv func(string) string) bool {
	if getenv("GITLAB_CI") == "true" || getenv("GITHUB_ACTIONS") == "true" {
		return true
	}
	ci := strings.ToLower(strings.TrimSpace(getenv("CI")))
	return ci != "" && ci != "false" && ci != "0"
}

// ChooseStyle picks the logger style, strongest first:
//
//  1. $CLOG_LOG_FORCE_STYLE, if it names a style
//  2. CiWithDbgTmp under CI
//  3. PrettyWithDbgTmp
//
// reason says which rule won. warning is non-empty when $CLOG_LOG_FORCE_STYLE
// is set but unusable: the automatic choice is used and the caller should say so.
func ChooseStyle(getenv func(string) string) (style SlogStyle, reason, warning string) {
	if forced := strings.TrimSpace(getenv(ForceStyleEnv)); forced != "" {
		st, err := ParseStyle(forced)
		if err == nil {
			return st, "forced by $" + ForceStyleEnv, ""
		}
		warning = fmt.Sprintf("$%s ignored: %v", ForceStyleEnv, err)
	}
	if IsCI(getenv) {
		return StyleCiWithDbgTmp, "CI detected", warning
	}
	return StylePrettyWithDbgTmp, "default", warning
}

// UseDefaultLogger installs the style ChooseStyle picks from the environment
// at the given console level, replacing (and closing) any current logger. Apps
// call it from init() and again for --debug.
func UseDefaultLogger(level slog.Level) (io.Closer, error) {
	style, reason, warning := ChooseStyle(os.Getenv)
	_ = CloseLogger()
	SetLogger(level, style)
	if warning != "" {
		slog.Warn(warning)
	}
	slog.Debug("logger style", "style", style.Name(), "reason", reason, "console_level", level.String())
	if logCloser == nil {
		return io.NopCloser(nil), nil
	}
	return logCloser, nil
}
