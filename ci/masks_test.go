package ci

import (
	"bytes"
	"strings"
	"testing"
)

// A pretty-printed JSON secret once masked "[" and "]": every bracket in the
// job log became *** and GitHub dropped the scan_files job output, a JSON list.
func TestGitHubMasksSkipStructuralLinesOfMultiLineSecrets(t *testing.T) {
	logs := captureLogs(t)
	var out bytes.Buffer
	writeGitHubMasks(&out, []Secret{
		{Key: "ADMIN_USERS", Value: "[\n  {\"email\": \"admin@example.com\"\n  },\n  -\n]\n"},
		{Key: "PEM", Value: "-----BEGIN KEY-----\r\nMIIBOgIBAAJBAKj34\r\n-----END KEY-----\r\n"},
	})
	masks := map[string]bool{}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		masks[strings.TrimPrefix(l, "::add-mask::")] = true
	}

	for _, want := range []string{
		`  {"email": "admin@example.com"`, // as written
		`{"email": "admin@example.com"`,   // and trimmed
		"-----BEGIN KEY-----", "MIIBOgIBAAJBAKj34", "-----END KEY-----",
	} {
		if !masks[want] {
			t.Errorf("not masked: %q\n%s", want, out.String())
		}
	}
	for _, never := range []string{"[", "]", "  },", "},", "  -", "-"} {
		if masks[never] {
			t.Errorf("structural line %q masked: it would redact ordinary log text", never)
		}
	}
	for m := range masks {
		if strings.ContainsAny(m, "\r") {
			t.Errorf("mask keeps a carriage return: %q", m)
		}
	}
	if !strings.Contains(logs.String(), "name=ADMIN_USERS") || !strings.Contains(logs.String(), "skipped-lines=4") {
		t.Errorf("want a warning naming ADMIN_USERS with 4 skipped lines:\n%s", logs.String())
	}
	if strings.Contains(logs.String(), "admin@example.com") {
		t.Errorf("warning leaks the secret:\n%s", logs.String())
	}
	if strings.Contains(logs.String(), "name=PEM") {
		t.Errorf("PEM has no skipped lines and should not warn:\n%s", logs.String())
	}
}

// A single-line secret is masked whole, however short: dropping it would be
// a leak, not a log-hygiene choice.
func TestGitHubMasksSingleLineAlwaysMasked(t *testing.T) {
	captureLogs(t)
	var out bytes.Buffer
	writeGitHubMasks(&out, []Secret{{Key: "PIN", Value: "42"}, {Key: "EMPTY", Value: ""}, {Key: "NL", Value: "token-abc\n"}})
	got := out.String()
	if !strings.Contains(got, "::add-mask::42\n") || !strings.Contains(got, "::add-mask::token-abc\n") {
		t.Errorf("single-line values must be masked:\n%s", got)
	}
	if strings.Count(got, "::add-mask::") != 2 {
		t.Errorf("want exactly 2 masks (empty value skipped):\n%s", got)
	}
}
