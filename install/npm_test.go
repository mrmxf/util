//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeUpstream serves the npm registry and the nodejs.org index.
func fakeUpstream(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/npm/wrangler/latest":
			_, _ = w.Write([]byte(`{"name":"wrangler","version":"4.45.0"}`))
		case "/node/index.json":
			_, _ = w.Write([]byte(`[{"version":"v25.1.0","lts":false},{"version":"v24.11.0","lts":"Krypton"},{"version":"v22.21.0","lts":"Jod"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	prevNpm, prevNode := npmRegistry, nodeDist
	npmRegistry, nodeDist = srv.URL+"/npm", srv.URL+"/node"
	t.Cleanup(func() { npmRegistry, nodeDist = prevNpm, prevNode })
}

func TestResolveNpmLatest(t *testing.T) {
	fakeUpstream(t)
	v, err := ResolveVersion(&VersionSpec{Strategy: "npm-latest", Package: "wrangler"})
	if err != nil || v != "4.45.0" {
		t.Fatalf("npm-latest = %q, %v; want 4.45.0", v, err)
	}
	if _, err := ResolveVersion(&VersionSpec{Strategy: "npm-latest", Package: "no-such-pkg"}); err == nil {
		t.Error("want an error for an unknown package")
	}
	if _, err := ResolveVersion(&VersionSpec{Strategy: "npm-latest"}); err == nil {
		t.Error("want an error without a package")
	}
}

func TestResolveNodeLTS(t *testing.T) {
	fakeUpstream(t)
	v, err := ResolveVersion(&VersionSpec{Strategy: "node-lts"})
	if err != nil || v != "24.11.0" {
		t.Fatalf("node-lts = %q, %v; want 24.11.0 (newest LTS, not the newer Current)", v, err)
	}
}

func TestResolveNvmrc(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) string {
		p := filepath.Join(dir, ".nvmrc")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for body, want := range map[string]string{"26.13.0\n": "26.13.0", "v26.13.0": "26.13.0", " v24.11.1 \n# c\n": "24.11.1"} {
		v, err := ResolveVersion(&VersionSpec{Strategy: "nvmrc", File: write(body)})
		if err != nil || v != want {
			t.Errorf("nvmrc %q = %q, %v; want %s", body, v, err, want)
		}
	}
	for _, body := range []string{"lts/*", "node", "26", ""} {
		if _, err := ResolveVersion(&VersionSpec{Strategy: "nvmrc", File: write(body)}); err == nil {
			t.Errorf("nvmrc %q should be refused", body)
		}
	}
	// no file: the fallback decides
	fakeUpstream(t)
	v, err := ResolveVersion(&VersionSpec{Strategy: "nvmrc", File: filepath.Join(dir, "missing"),
		Fallback: &VersionSpec{Strategy: "node-lts"}})
	if err != nil || v != "24.11.0" {
		t.Errorf("missing .nvmrc with node-lts fallback = %q, %v", v, err)
	}
}
