//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

// `clog Install clog` and its alias `clog Upgrade`.
//
// This is get-clog.sh in Go, step for step, and it has to stay that way: the
// script bootstraps every CI job (setup-clog wraps it, GitLab sources it), so
// if this file and the script resolve a different version, asset or URL, a
// laptop runs a different clog from CI and nobody notices. mrmxf/clog's
// getclog_parity_test.go runs both and fails on any difference - change them
// together.
//
// It follows the script in everything but one place: with no CLOG_INSTALL_DIR
// it replaces the running binary rather than writing $HOME/.local/bin/clog.
// In CI those are the same file (the running clog IS $RUNNER_TEMP/clog/bin/clog);
// on a laptop a second clog in another PATH directory is itself drift.

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/mrmxf/util/buildinfo"
)

const (
	selfStrategy    = "self-replace"
	defaultClogRepo = "mrmxf/clog" // the same default get-clog.sh carries
	defaultPinFile  = ".clog-version"
)

// pinFlag is --pin: write the installed version into the pin file.
var pinFlag bool

// selfPlan is everything an install of clog resolves before it downloads.
// Its lines are what --dry-run prints and what get-clog.sh's CLOG_PLAN=1 prints,
// so the parity test can compare the two.
type selfPlan struct {
	Version       string // the release tag, e.g. v1.3.5
	VersionSource string
	Repo          string // owner/repo holding the release
	RepoSource    string
	Asset         string // clog-<cpu>-<os>
	Base          string // https://github.com/<repo>/releases/download/<tag>
	Dir           string // install directory
	Dest          string // file that is replaced
	PinFile       string // the pin file this run reads and --pin writes
}

func (p selfPlan) URL() string { return p.Base + "/" + p.Asset }

// lines is the plan in the key=value form get-clog.sh's CLOG_PLAN=1 prints.
func (p selfPlan) lines() string {
	return fmt.Sprintf("version=%s\nrepo=%s\nasset=%s\nurl=%s\ndir=%s\n",
		p.Version, p.Repo, p.Asset, p.URL(), p.Dir)
}

// Replaceable for tests.
var (
	builtRepo      = func() string { return buildinfo.Info().Repo }
	builtTag       = func() string { return buildinfo.Info().Tag }
	executablePath = os.Executable
	selfGOOS       = runtime.GOOS
	selfGOARCH     = runtime.GOARCH
	githubWeb      = "https://github.com"
)

func isSelfReplace(e ToolEntry) bool {
	return e.Install != nil && e.Install.Strategy == selfStrategy
}

// runSelf installs the clog release the plan resolves, replacing the target
// atomically once the new binary is verified and has run.
func runSelf(entry ToolEntry, dryRun bool) error {
	p, err := planSelf(entry)
	if err != nil {
		return err
	}
	slog.Info("clog install plan", "version", p.Version, "from", p.VersionSource,
		"repo", p.Repo, "repo-from", p.RepoSource, "asset", p.Asset, "dest", p.Dest)

	if dryRun {
		_, _ = fmt.Fprint(os.Stdout, p.lines())
		warnPinDrift(p)
		slog.Info("dry-run: stopping before download")
		return nil
	}

	if !sameFile(p.Dest, runningExe()) || builtTag() != p.Version {
		if err := installSelf(p); err != nil {
			return err
		}
	} else {
		slog.Info("clog is already at this version; nothing to replace", "version", p.Version, "path", p.Dest)
	}

	if pinFlag {
		if err := writePin(p.PinFile, p.Version); err != nil {
			return err
		}
	} else {
		warnPinDrift(p)
	}
	warnShadowed(p.Dest)
	return nil
}

// planSelf resolves version, repo, asset and destination, in get-clog.sh's
// order. It reads the network only for --use latest|lts or a `latest` pin.
func planSelf(entry ToolEntry) (selfPlan, error) {
	p := selfPlan{PinFile: os.Getenv("CLOG_VERSION_FILE")}
	if p.PinFile == "" {
		p.PinFile = defaultPinFile
		if entry.Version != nil && entry.Version.Strategy == "pin-file" && entry.Version.File != "" {
			p.PinFile = entry.Version.File
		}
	}

	if err := resolveSelfRepo(&p, entry); err != nil {
		return p, err
	}
	if err := resolveSelfVersion(&p, entry); err != nil {
		return p, err
	}

	asset, err := selfAsset(selfGOOS, selfGOARCH)
	if err != nil {
		return p, err
	}
	p.Asset = asset
	p.Base = fmt.Sprintf("%s/%s/releases/download/%s", githubWeb, p.Repo, p.Version)

	if d := os.Getenv("CLOG_INSTALL_DIR"); d != "" {
		p.Dir, p.Dest = d, filepath.Join(d, "clog")
	} else {
		exe := runningExe()
		if exe == "" {
			return p, fmt.Errorf("cannot find the running clog to replace; set CLOG_INSTALL_DIR to say where to install")
		}
		p.Dir, p.Dest = filepath.Dir(exe), exe
	}
	return p, nil
}

// resolveSelfRepo: CLOG_REPO, then the recipe's repo (only a .clog.yaml
// install.clog override sets one), then the repo this binary was released
// from, then mrmxf/clog. Anything other than CLOG_REPO that would install from
// a different repo than this binary's is refused: that silently swaps the
// edition (a clog-mrmxf binary turning into the public one).
func resolveSelfRepo(p *selfPlan, entry ToolEntry) error {
	built := builtRepo()
	switch {
	case os.Getenv("CLOG_REPO") != "":
		p.Repo, p.RepoSource = os.Getenv("CLOG_REPO"), "CLOG_REPO"
		if built != "" && p.Repo != built {
			slog.Warn("installing from a different repo than this clog was released from",
				"this-clog", built, "installing", p.Repo, "source", "CLOG_REPO")
		}
		return nil
	case entry.Install != nil && entry.Install.Repo != "":
		p.Repo, p.RepoSource = entry.Install.Repo, "install.clog.install.repo"
	case built != "":
		p.Repo, p.RepoSource = built, "this clog's build info"
		return nil
	default:
		p.Repo, p.RepoSource = defaultClogRepo, "default"
		return nil
	}
	if built != "" && p.Repo != built {
		return fmt.Errorf("this clog was released from %s but %s says install from %s, which would swap the edition; "+
			"say so explicitly with CLOG_REPO=%s", built, p.RepoSource, p.Repo, p.Repo)
	}
	return nil
}

// resolveSelfVersion: --use/--at, CLOG_VERSION, the pin file, else an error.
// A missing pin never falls back to latest: a repo that lost its .clog-version
// must fail rather than float.
func resolveSelfVersion(p *selfPlan, entry ToolEntry) error {
	req, err := requestedVersion()
	if err != nil {
		return err
	}
	switch {
	case req != "":
		p.VersionSource = "command line --use"
		return applySelfUse(p, req)
	case os.Getenv("CLOG_VERSION") != "":
		p.Version, p.VersionSource = os.Getenv("CLOG_VERSION"), "CLOG_VERSION"
	case entry.Version != nil && entry.Version.Strategy != "pin-file":
		// a .clog.yaml override chose another strategy; the recipe engine resolves it
		v, err := ResolveVersion(entry.Version)
		if err != nil {
			return err
		}
		p.Version, p.VersionSource = "v"+strings.TrimPrefix(v, "v"), "recipe version ("+entry.Version.Strategy+")"
	default:
		v, err := readPinFile(p.PinFile)
		if err != nil {
			return fmt.Errorf("%w; pass --use <version> or --use latest to choose one", err)
		}
		p.Version, p.VersionSource = v, "pin file "+p.PinFile
	}
	if p.Version == useLatest {
		v, err := latestRelease(p.Repo)
		if err != nil {
			return err
		}
		p.Version = v
	}
	return nil
}

// applySelfUse applies the normal --use rules: a version number, latest or lts.
func applySelfUse(p *selfPlan, req string) error {
	req = strings.TrimSpace(req)
	slog.Info("version requested", "use", req, "source", "command line")
	switch strings.ToLower(req) {
	case useLatest:
		v, err := latestRelease(p.Repo)
		p.Version = v
		return err
	case useLTS:
		v, err := resolveLTS(&VersionSpec{Strategy: "github-tags", Repo: p.Repo, TagPrefix: "v", TagFilter: `^v[0-9]+\.[0-9]+\.[0-9]+$`})
		p.Version = "v" + strings.TrimPrefix(v, "v")
		return err
	default:
		v, err := normaliseVersion(req, "")
		p.Version = "v" + v // clog release tags always carry the v
		return err
	}
}

var releaseTagRE = regexp.MustCompile(`^v[0-9]`)

// latestRelease resolves `latest` as get-clog.sh does: follow the
// /releases/latest redirect and take the tag it lands on. No API call, so no
// rate limit.
func latestRelease(repo string) (string, error) {
	url := fmt.Sprintf("%s/%s/releases/latest", githubWeb, repo)
	resp, err := httpClient.Get(url) //nolint:gosec // repo is configuration
	if err != nil {
		return "", fmt.Errorf("cannot resolve the latest release of %s: %w", repo, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cannot resolve the latest release of %s: HTTP %d", repo, resp.StatusCode)
	}
	final := resp.Request.URL.String()
	tag := final[strings.LastIndex(final, "/")+1:]
	if !releaseTagRE.MatchString(tag) {
		return "", fmt.Errorf("latest release of %s is not a vX.Y.Z tag: %s", repo, final)
	}
	logVersionMatch(tag, "GitHub latest release redirect", "repo", repo)
	return tag, nil
}

// selfAsset is the release asset name for a platform, mapped the way
// get-clog.sh maps uname and bc-golang names its outputs. It goes by
// GOOS/GOARCH, not the distro: any Linux runs the static binary.
func selfAsset(goos, goarch string) (string, error) {
	osTok := map[string]string{"linux": "lnx", "darwin": "mac"}[goos]
	cpu := map[string]string{"amd64": "amd", "arm64": "arm"}[goarch]
	if osTok == "" {
		return "", fmt.Errorf("unsupported OS %s: clog is released for linux and darwin", goos)
	}
	if cpu == "" {
		return "", fmt.Errorf("unsupported ARCH %s: clog is released for amd64 and arm64", goarch)
	}
	return "clog-" + cpu + "-" + osTok, nil
}

// readPinFile returns the first line that is not blank or a comment, with all
// whitespace removed, exactly as get-clog.sh reads it:
//
//	grep -vE '^\s*(#|$)' "$CLOG_VERSION_FILE" | head -n1 | tr -d '[:space:]'
func readPinFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no CLOG_VERSION and no pin file at '%s'", path)
		}
		return "", fmt.Errorf("read pin file %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return strings.Join(strings.Fields(t), ""), nil
	}
	return "", fmt.Errorf("pin file '%s' is empty", path)
}

// resolvePinFile is the pin-file version strategy, for recipes other than
// clog's own. It returns the version without its leading v.
func resolvePinFile(spec *VersionSpec) (string, error) {
	file := spec.File
	if file == "" {
		return "", fmt.Errorf("pin-file: file field is required")
	}
	v, err := readPinFile(file)
	if err != nil {
		return "", fmt.Errorf("pin-file: %w", err)
	}
	logVersionMatch(v, "pin file", "path", absPath(file))
	return strings.TrimPrefix(v, "v"), nil
}

// writePin makes path's pin line say version. Comments and blank lines are
// kept; only the first pin line is replaced, or one is appended.
func writePin(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("--pin: read %s: %w", path, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(data) == 0 {
		lines = nil
	}
	done := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if t == version {
			slog.Info("pin already says this version", "file", path, "version", version)
			return nil
		}
		lines[i], done = version, true
		break
	}
	if !done {
		lines = append(lines, version)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("--pin: write %s: %w", path, err)
	}
	slog.Info("pinned: commit this so CI installs the same clog", "file", path, "version", version)
	return nil
}

// warnPinDrift warns when the version chosen differs from the pin CI reads.
func warnPinDrift(p selfPlan) {
	pin, err := readPinFile(p.PinFile)
	if err != nil || pin == p.Version {
		return
	}
	slog.Warn("this clog now differs from the version CI installs",
		"installed", p.Version, "pinned", pin, "pin-file", p.PinFile,
		"fix", "clog upgrade --use "+p.Version+" --pin, then commit "+p.PinFile)
}

// warnShadowed warns when `clog` on PATH is not the file just installed, so a
// fresh install is not hidden behind an older one in an earlier directory.
func warnShadowed(dest string) {
	onPath, err := exec.LookPath("clog")
	if err != nil {
		slog.Warn("clog is not on PATH", "installed", dest, "fix", "add "+filepath.Dir(dest)+" to PATH")
		return
	}
	if !sameFile(onPath, dest) {
		slog.Warn("another clog comes first on PATH; it is not the one just installed",
			"on-path", onPath, "installed", dest)
	}
}

// installSelf downloads, verifies, smoke-tests and atomically swaps in the new
// binary. Nothing replaces dest until every check has passed.
func installSelf(p selfPlan) error {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", p.Dir, err)
	}
	// the temp file lives beside dest so the final rename cannot cross filesystems
	tmp, err := os.CreateTemp(p.Dir, ".clog-new-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (%w): re-run with sudo, or set CLOG_INSTALL_DIR to a directory you own", p.Dir, err)
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()

	slog.Info("installing clog", "version", p.Version, "asset", p.Asset, "repo", p.Repo, "dest", p.Dest)
	if err := fetchTo(p.URL(), tmp, true); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download failed: %s (is %s released for %s?): %w", p.URL(), p.Version, p.Repo, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if os.Getenv("CLOG_SKIP_VERIFY") == "1" {
		slog.Warn("CLOG_SKIP_VERIFY=1 — skipping checksum verification")
	} else if err := verifySelfChecksum(p, tmpPath); err != nil {
		return err
	}
	if err := verifySelfSLSA(p, tmpPath); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}
	out, err := exec.Command(tmpPath, "--version").CombinedOutput() //nolint:gosec // verified above
	if err != nil {
		return fmt.Errorf("the downloaded clog does not run (%w): %s; the installed clog is unchanged", err, strings.TrimSpace(string(out)))
	}
	if err := os.Rename(tmpPath, p.Dest); err != nil {
		return fmt.Errorf("replace %s: %w", p.Dest, err)
	}
	keep = true
	slog.Info("installed", "version", strings.TrimSpace(string(out)), "path", p.Dest)
	return nil
}

func verifySelfChecksum(p selfPlan, path string) error {
	url := p.Base + "/checksums.txt"
	var sb strings.Builder
	if err := fetchTo(url, &sb, true); err != nil {
		return fmt.Errorf("could not fetch checksums.txt (%w) — refusing to install an unverified binary (set CLOG_SKIP_VERIFY=1 to override)", err)
	}
	want, err := findChecksum(strings.NewReader(sb.String()), p.Asset)
	if err != nil {
		return fmt.Errorf("checksums.txt has no entry for %s", p.Asset)
	}
	got, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("checksum mismatch for %s: want %s, got %s", p.Asset, want, got)
	}
	slog.Info("checksum verified", "sha256", got)
	return nil
}

// verifySelfSLSA checks provenance when slsa-verifier is on PATH and the
// release publishes it; a release without provenance is skipped, as in
// get-clog.sh. A provenance file that fails to verify stops the install.
func verifySelfSLSA(p selfPlan, path string) error {
	if _, err := exec.LookPath("slsa-verifier"); err != nil {
		return nil
	}
	prov, err := os.CreateTemp("", "clog-*.intoto.jsonl")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(prov.Name()) }()
	err = fetchTo(p.URL()+".intoto.jsonl", prov, false)
	_ = prov.Close()
	if err != nil {
		slog.Info("no provenance asset published; skipping SLSA check", "asset", p.Asset)
		return nil
	}
	slog.Info("verifying SLSA provenance")
	if err := streamCmd("slsa-verifier", "verify-artifact", path,
		"--provenance-path", prov.Name(),
		"--source-uri", "github.com/"+p.Repo,
		"--source-tag", p.Version); err != nil {
		return fmt.Errorf("SLSA provenance verification failed: %w", err)
	}
	slog.Info("SLSA provenance verified")
	return nil
}

var httpClient = &http.Client{Timeout: 5 * time.Minute}

// fetchTo GETs url into w. The token (CLOG_TOKEN, then GITHUB_TOKEN, GH_TOKEN,
// GHAT) is sent for private-repo releases; Go drops it on the redirect to the
// asset CDN, which is a different host.
func fetchTo(url string, w io.Writer, required bool) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	token := os.Getenv("CLOG_TOKEN")
	if token == "" {
		token, _ = githubToken()
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if required {
			slog.Error("download failed", "url", url, "status", resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// runningExe is this process's binary with symlinks resolved, or "".
func runningExe() string {
	exe, err := executablePath()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		return r
	}
	return exe
}

func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}
