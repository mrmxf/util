//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// selfEnv clears every input get-clog.sh reads and resets the shared flags.
func selfEnv(t *testing.T) string {
	t.Helper()
	for _, k := range []string{"CLOG_VERSION", "CLOG_VERSION_FILE", "CLOG_REPO", "CLOG_INSTALL_DIR", "CLOG_TOKEN", "CLOG_SKIP_VERIFY"} {
		t.Setenv(k, "")
	}
	useFlag, atFlag, pinFlag = "", "", false
	t.Cleanup(func() { useFlag, atFlag, pinFlag = "", "", false })
	dir := t.TempDir()
	t.Chdir(dir)
	oldRepo, oldTag, oldExe := builtRepo, builtTag, executablePath
	t.Cleanup(func() { builtRepo, builtTag, executablePath = oldRepo, oldTag, oldExe })
	builtRepo = func() string { return "" }
	builtTag = func() string { return "" }
	return dir
}

func clogRecipe(t *testing.T) ToolEntry {
	t.Helper()
	m, err := LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	_, e, err := LoadTool(m, "clog", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !isSelfReplace(e) {
		t.Fatalf("clog recipe install strategy = %+v, want self-replace", e.Install)
	}
	return e
}

func TestSelfVersionPrecedence(t *testing.T) {
	cases := []struct {
		name, use, env, file, fileBody, want, wantErr string
	}{
		{name: "pin file", fileBody: "v1.3.5\n", want: "v1.3.5"},
		{name: "pin file with comments", fileBody: "# header\n\n  # indented\n v1.3.4 \nv9.9.9\n", want: "v1.3.4"},
		{name: "env beats file", env: "v1.2.0", fileBody: "v1.3.5\n", want: "v1.2.0"},
		{name: "--use beats env", use: "1.1.0", env: "v1.2.0", fileBody: "v1.3.5\n", want: "v1.1.0"},
		{name: "--use keeps a v", use: "v1.1.0", want: "v1.1.0"},
		{name: "CLOG_VERSION_FILE", file: "pins/clog.txt", fileBody: "v1.0.9\n", want: "v1.0.9"},
		{name: "no pin is an error", wantErr: "no CLOG_VERSION and no pin file at '.clog-version'"},
		{name: "empty pin is an error", fileBody: "# nothing\n\n", wantErr: "is empty"},
		{name: "bad --use", use: "stable", wantErr: "expected a version number"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := selfEnv(t)
			useFlag = c.use
			t.Setenv("CLOG_VERSION", c.env)
			pin := ".clog-version"
			if c.file != "" {
				pin = c.file
				t.Setenv("CLOG_VERSION_FILE", c.file)
			}
			if c.fileBody != "" {
				p := filepath.Join(dir, pin)
				_ = os.MkdirAll(filepath.Dir(p), 0o755)
				if err := os.WriteFile(p, []byte(c.fileBody), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			p, err := planSelf(clogRecipe(t))
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Version != c.want {
				t.Errorf("version = %q, want %q", p.Version, c.want)
			}
		})
	}
}

func TestSelfRepoPrecedenceAndEditionGuard(t *testing.T) {
	selfEnv(t)
	t.Setenv("CLOG_VERSION", "v1.0.0")
	e := clogRecipe(t)

	p, err := planSelf(e)
	if err != nil || p.Repo != defaultClogRepo {
		t.Fatalf("default repo = %q, %v; want %s", p.Repo, err, defaultClogRepo)
	}

	builtRepo = func() string { return "mrmxf/clog-mrmxf" }
	if p, _ = planSelf(e); p.Repo != "mrmxf/clog-mrmxf" {
		t.Errorf("a stamped repo must be the default, got %q", p.Repo)
	}

	override := e
	inst := *e.Install
	inst.Repo = "mrmxf/clog"
	override.Install = &inst
	if _, err = planSelf(override); err == nil || !strings.Contains(err.Error(), "swap the edition") {
		t.Errorf("a config repo that swaps the edition must be refused, got %v", err)
	}

	t.Setenv("CLOG_REPO", "mrmxf/clog")
	if p, err = planSelf(override); err != nil || p.Repo != "mrmxf/clog" {
		t.Errorf("CLOG_REPO is explicit and must win: %q, %v", p.Repo, err)
	}
}

func TestSelfAsset(t *testing.T) {
	ok := map[[2]string]string{
		{"linux", "amd64"}:  "clog-amd-lnx",
		{"linux", "arm64"}:  "clog-arm-lnx",
		{"darwin", "amd64"}: "clog-amd-mac",
		{"darwin", "arm64"}: "clog-arm-mac",
	}
	for in, want := range ok {
		if got, err := selfAsset(in[0], in[1]); err != nil || got != want {
			t.Errorf("selfAsset%v = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range [][2]string{{"windows", "amd64"}, {"linux", "386"}} {
		if _, err := selfAsset(bad[0], bad[1]); err == nil {
			t.Errorf("selfAsset%v: want an error", bad)
		}
	}
}

func TestSelfDestDefaultsToRunningBinary(t *testing.T) {
	dir := selfEnv(t)
	t.Setenv("CLOG_VERSION", "v1.0.0")
	exe := filepath.Join(dir, "bin", "clog-local")
	_ = os.MkdirAll(filepath.Dir(exe), 0o755)
	_ = os.WriteFile(exe, []byte("x"), 0o755)
	executablePath = func() (string, error) { return exe, nil }

	p, err := planSelf(clogRecipe(t))
	if err != nil {
		t.Fatal(err)
	}
	if p.Dest != exe || p.Dir != filepath.Dir(exe) {
		t.Errorf("dest = %q dir = %q, want the running binary %q", p.Dest, p.Dir, exe)
	}

	t.Setenv("CLOG_INSTALL_DIR", filepath.Join(dir, "elsewhere"))
	if p, _ = planSelf(clogRecipe(t)); p.Dest != filepath.Join(dir, "elsewhere", "clog") {
		t.Errorf("CLOG_INSTALL_DIR dest = %q", p.Dest)
	}
}

func TestWritePinKeepsComments(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, ".clog-version")
	_ = os.WriteFile(f, []byte("# why we pin\nv1.0.0\n# trailer\n"), 0o644)
	if err := writePin(f, "v1.3.5"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(f)
	if string(got) != "# why we pin\nv1.3.5\n# trailer\n" {
		t.Errorf("pin file = %q", got)
	}
	fresh := filepath.Join(dir, "new")
	if err := writePin(fresh, "v1.3.5"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(fresh); string(got) != "v1.3.5\n" {
		t.Errorf("new pin file = %q", got)
	}
}

// fakeRelease serves one release of a shell-script "clog" plus checksums.txt.
func fakeRelease(t *testing.T, binary string, checksum string) *httptest.Server {
	t.Helper()
	if checksum == "" {
		sum := sha256.Sum256([]byte(binary))
		checksum = hex.EncodeToString(sum[:])
	}
	asset, _ := selfAsset(selfGOOS, selfGOARCH)
	mux := http.NewServeMux()
	mux.HandleFunc("/mrmxf/clog/releases/download/v1.3.5/"+asset, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, binary)
	})
	mux.HandleFunc("/mrmxf/clog/releases/download/v1.3.5/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", checksum, asset)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	old := githubWeb
	githubWeb = srv.URL
	t.Cleanup(func() { githubWeb = old })
	return srv
}

func TestInstallSelfReplacesOnlyAWorkingVerifiedBinary(t *testing.T) {
	if selfGOOS == "windows" {
		t.Skip("shell-script stand-in binary")
	}
	const good = "#!/bin/sh\necho clog v1.3.5\n"
	const broken = "#!/bin/sh\nexit 3\n"

	run := func(t *testing.T, binary, checksum string) (string, error) {
		dir := selfEnv(t)
		t.Setenv("CLOG_VERSION", "v1.3.5")
		exe := filepath.Join(dir, "clog")
		_ = os.WriteFile(exe, []byte("old"), 0o755)
		executablePath = func() (string, error) { return exe, nil }
		fakeRelease(t, binary, checksum)
		err := runSelf(clogRecipe(t), false)
		got, _ := os.ReadFile(exe)
		return string(got), err
	}

	t.Run("good", func(t *testing.T) {
		got, err := run(t, good, "")
		if err != nil || got != good {
			t.Fatalf("err = %v, binary = %q", err, got)
		}
	})
	t.Run("checksum mismatch", func(t *testing.T) {
		got, err := run(t, good, strings.Repeat("0", 64))
		if err == nil || !strings.Contains(err.Error(), "checksum mismatch") || got != "old" {
			t.Fatalf("err = %v, binary = %q; want mismatch and the old binary kept", err, got)
		}
	})
	t.Run("does not run", func(t *testing.T) {
		got, err := run(t, broken, "")
		if err == nil || !strings.Contains(err.Error(), "does not run") || got != "old" {
			t.Fatalf("err = %v, binary = %q; want refusal and the old binary kept", err, got)
		}
	})
	t.Run("pin", func(t *testing.T) {
		dir := selfEnv(t)
		useFlag, pinFlag = "1.3.5", true
		_ = os.WriteFile(filepath.Join(dir, ".clog-version"), []byte("# pin\nv1.0.0\n"), 0o644)
		exe := filepath.Join(dir, "clog")
		_ = os.WriteFile(exe, []byte("old"), 0o755)
		executablePath = func() (string, error) { return exe, nil }
		fakeRelease(t, good, "")
		if err := runSelf(clogRecipe(t), false); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, ".clog-version")); string(got) != "# pin\nv1.3.5\n" {
			t.Errorf("--pin wrote %q", got)
		}
	})
}
