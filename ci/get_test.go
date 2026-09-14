//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import "testing"

func TestGet(t *testing.T) {
	withConfig(t, Config{}, map[string]any{
		"ci.artifact": "clog",
		"ci.list":     []any{"a", "b"},
		"ci.empty":    "",
	})
	for _, tc := range []struct {
		key, want string
		required  bool
		wantErr   bool
	}{
		{key: "ci.artifact", want: "clog"},
		{key: "ci.list", want: "a\nb"},
		{key: "ci.missing", want: ""},
		{key: "ci.empty", want: ""},
		{key: "ci.missing", required: true, wantErr: true},
		{key: "ci.artifact", required: true, want: "clog"},
	} {
		got, err := Get(tc.key, tc.required)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("Get(%q, required=%v) = %q, %v", tc.key, tc.required, got, err)
		}
	}
}
