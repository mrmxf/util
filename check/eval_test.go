//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import "testing"

func TestMatchState(t *testing.T) {
	cases := []struct {
		name  string
		state string
		value string
		ok    bool
		want  bool
	}{
		{"truthy when ok", "truthy", "", true, true},
		{"truthy when not ok", "truthy", "", false, false},
		{"empty state defaults to truthy", "", "", true, true},
		{"falsy when not ok", "falsy", "", false, true},
		{"falsy when ok", "falsy", "", true, false},
		{"exact match", "v1.2.3", "v1.2.3", true, true},
		{"exact mismatch", "v1.2.3", "v9", true, false},
		{"exact ignores ok signal", "hello", "hello", false, true},
		{"regex match", "~v[0-9]+", "v123", false, true},
		{"regex no match", "~^v[0-9]+$", "xv1", false, false},
		{"regex on trimmed value", "~^abc$", "  abc  ", false, true},
		{"numeric greater pass", ">5", "9", false, true},
		{"numeric greater fail", ">5", "2", false, false},
		{"numeric less pass", "<5", "2", false, true},
		{"numeric less fail", "<5", "9", false, false},
		{"numeric non-int value fails", ">5", "abc", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchState(c.state, c.value, c.ok); got != c.want {
				t.Fatalf("matchState(%q,%q,%v)=%v want %v", c.state, c.value, c.ok, got, c.want)
			}
		})
	}
}

func TestEvalEnv(t *testing.T) {
	t.Setenv("CHK_SET", "value")
	t.Setenv("CHK_EMPTY", "")

	cases := []struct {
		name string
		cond Condition
		want bool
	}{
		{"set var truthy", Condition{"CHK_SET", "truthy"}, true},
		{"empty var truthy fails", Condition{"CHK_EMPTY", "truthy"}, false},
		{"unset var truthy fails", Condition{"CHK_MISSING", "truthy"}, false},
		{"unset var falsy passes", Condition{"CHK_MISSING", "falsy"}, true},
		{"empty var falsy passes", Condition{"CHK_EMPTY", "falsy"}, true},
		{"exact value match", Condition{"CHK_SET", "value"}, true},
		{"exact value mismatch", Condition{"CHK_SET", "nope"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := evalEnv(c.cond)
			if got != c.want {
				t.Fatalf("evalEnv(%+v)=%v want %v", c.cond, got, c.want)
			}
		})
	}
}
