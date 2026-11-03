//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serveChecksums serves body as a checksums file and returns its URL.
func serveChecksums(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/checksums.txt"
}

func artifact(t *testing.T, data string) (path, sum string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte(data))
	return path, hex.EncodeToString(h[:])
}

func TestVerifyChecksum(t *testing.T) {
	path, sum := artifact(t, "nabu binary")
	other := strings.Repeat("0", 64)
	cases := []struct {
		name, body string
		status     int
		wantErr    string
	}{
		{"match", other + "  nabu-amd-mac\n" + sum + "  nabu-amd-lnx\n", 200, ""},
		{"binary-mode star", sum + " *nabu-amd-lnx\n", 200, ""},
		{"upper-case hex", strings.ToUpper(sum) + "  nabu-amd-lnx\n", 200, ""},
		{"mismatch", other + "  nabu-amd-lnx\n", 200, "checksum mismatch"},
		{"no entry", sum + "  nabu-arm-lnx\n", 200, "no entry for nabu-amd-lnx"},
		{"http error", "", 404, "HTTP 404"},
	}
	for _, c := range cases {
		url := serveChecksums(t, c.status, c.body)
		err := VerifyChecksum(url, "nabu-amd-lnx", path)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error: %v", c.name, err)
		case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
			t.Errorf("%s: err = %v, want containing %q", c.name, err, c.wantErr)
		}
	}
}

func TestURLBase(t *testing.T) {
	for in, want := range map[string]string{
		"https://x/releases/download/v1.2.3/nabu-amd-lnx": "nabu-amd-lnx",
		"https://x/a/node-v24.1.0-linux-x64.tar.gz?raw=1": "node-v24.1.0-linux-x64.tar.gz",
		"https://x/gitleaks_8.28.0_linux_x64.tar.gz#frag": "gitleaks_8.28.0_linux_x64.tar.gz",
	} {
		if got := urlBase(in); got != want {
			t.Errorf("urlBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckCleanPath(t *testing.T) {
	for p, ok := range map[string]bool{
		"/usr/local/lib/node_modules/npm":      true,
		"/usr/local/lib/node_modules/corepack": true,
		"/usr/local/lib":                       true,
		"/usr/local":                           false,
		"/opt":                                 false,
		"/":                                    false,
		"usr/local/lib/x":                      false,
		"/usr/local/lib/../../etc":             false,
		"/usr/local/lib/x/":                    false,
	} {
		if err := checkCleanPath(p); (err == nil) != ok {
			t.Errorf("checkCleanPath(%q) = %v, want ok=%v", p, err, ok)
		}
	}
}
