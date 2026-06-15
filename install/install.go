//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mrmxf/util/check"
)

// Run orchestrates the full install pipeline for a tool on a given platform.
// dryRun=true resolves version + URL and logs the plan without executing anything.
func Run(entry ToolEntry, toolName string, platform Platform, dryRun bool) error {
	pe, ok := entry.Platforms[platform]
	if !ok {
		return fmt.Errorf("tool %q has no recipe for platform %s", toolName, platform)
	}
	if pe.Unsupported {
		res, err := resolveUnsupported(entry, toolName, platform)
		if err != nil {
			return err
		}
		if res.goInstall {
			return runGoInstallFallback(entry, toolName, dryRun)
		}
		// install the amd64 build instead (runs under emulation on arm64).
		platform = res.platform
		pe = entry.Platforms[platform]
	}

	// Resolve effective specs (platform overrides take precedence).
	verSpec := entry.Version
	if pe.Version != nil {
		verSpec = pe.Version
	}
	dlSpec := entry.Download
	if pe.Download != nil {
		dlSpec = pe.Download
	}
	instSpec := entry.Install
	if pe.Install != nil {
		instSpec = pe.Install
	}
	postSteps := entry.PostInstall
	if len(pe.PostInstall) > 0 {
		postSteps = pe.PostInstall
	}

	// Resolve version.
	version := "[unknown]"
	if verSpec != nil {
		v, err := ResolveVersion(verSpec)
		if err != nil {
			slog.Warn("version resolution failed", "tool", toolName, "err", err)
		} else {
			version = v
		}
	}

	osVal := pe.OS
	archVal := pe.Arch

	if dryRun {
		return runDryRun(toolName, platform, version, osVal, archVal, dlSpec, instSpec, postSteps)
	}

	if err := runInstallStrategy(toolName, platform, version, osVal, archVal, dlSpec, instSpec); err != nil {
		return err
	}

	if len(postSteps) > 0 {
		if err := RunPostInstall(postSteps); err != nil {
			slog.Warn("post-install had errors", "tool", toolName, "err", err)
		}
	}
	return nil
}

func runDryRun(toolName string, platform Platform, version, osVal, archVal string, dl *DownloadSpec, inst *InstallSpec, post []PostStep) error {
	slog.Info("dry-run plan", "tool", toolName, "platform", platform, "version", version)
	if dl != nil {
		url := substituteTokens(dl.URL, version, osVal, archVal)
		slog.Info("  download", "strategy", dl.Strategy, "url", url)
	}
	if inst != nil {
		detail := inst.Package
		if detail == "" {
			detail = inst.Import
		}
		if detail == "" {
			detail = substituteTokens(inst.Dest, version, osVal, archVal)
		}
		slog.Info("  install", "strategy", inst.Strategy, "detail", detail)
	}
	for i, ps := range post {
		slog.Info(fmt.Sprintf("  post-install[%d]", i+1), "strategy", ps.Strategy, "line", ps.Line)
	}
	return nil
}

// runInstallStrategy dispatches to the correct installer.
// Strategies that need a download step fetch first, then call installArtifact.
func runInstallStrategy(toolName string, platform Platform, version, osVal, archVal string, dl *DownloadSpec, inst *InstallSpec) error {
	if inst == nil {
		return fmt.Errorf("tool %q: no install spec for platform %s", toolName, platform)
	}

	switch inst.Strategy {
	case "brew-install":
		return runBrew(inst.Package)
	case "apt-install":
		return runApt(inst.Package, inst.Sudo)
	case "dnf-install":
		return runDnf(inst.Package, inst.Sudo)
	case "go-install":
		return runGoInstall(inst.Import)
	case "run-script":
		return runScript(inst.Script)
	case "lnx-native":
		return runLnxNative(inst, platform)
	default:
		// All remaining strategies require a downloaded artifact.
		if dl == nil {
			return fmt.Errorf("tool %q: strategy %q requires a download spec", toolName, inst.Strategy)
		}
		artifactPath, cleanup, err := downloadArtifact(dl, version, osVal, archVal)
		if err != nil {
			return err
		}
		defer cleanup()

		if dl.SLSA != nil {
			if err := VerifySLSA(dl.SLSA, artifactPath, version, osVal, archVal); err != nil {
				return fmt.Errorf("SLSA verification failed: %w", err)
			}
		}

		return installArtifact(inst, artifactPath, version, osVal, archVal)
	}
}

// installArtifact installs a pre-downloaded artifact using the strategy in inst.
func installArtifact(inst *InstallSpec, artifactPath, version, osVal, archVal string) error {
	switch inst.Strategy {
	case "extract-tar":
		return installExtractTar(inst, artifactPath)
	case "extract-tar-binary":
		return installExtractTarBinary(inst, artifactPath, version, osVal, archVal)
	case "install-deb":
		return installDeb(inst, artifactPath)
	case "install-rpm":
		return installRpm(inst, artifactPath)
	case "copy-binary":
		return installCopyBinary(inst, artifactPath)
	default:
		return fmt.Errorf("install strategy %q not implemented", inst.Strategy)
	}
}

// RunCheck runs entry.Check using the Phase 8 check engine (same engine as
// `clog Check`). It returns nil when the block is truthy — i.e. the tool is
// present — and an error otherwise. The then:/else: scripts in the block emit
// the user-facing messages.
func RunCheck(toolName string, blk *check.Block) error {
	if blk == nil || (len(blk.Try) == 0 && len(blk.Env) == 0) {
		return fmt.Errorf("tool %q has no check block", toolName)
	}
	b := *blk
	if b.Name == "" {
		b.Name = toolName
	}
	return check.RunBlock(b)
}

// ---- direct install strategies (no download step) ----

func runBrew(pkg string) error {
	if pkg == "" {
		return fmt.Errorf("brew-install: package field is required")
	}
	slog.Info("brew install", "package", pkg)
	return streamCmd("brew", "install", pkg)
}

func runApt(pkg string, sudo bool) error {
	if pkg == "" {
		return fmt.Errorf("apt-install: package field is required")
	}
	slog.Info("apt-get install", "package", pkg)
	if sudo {
		return streamCmd("sudo", "apt-get", "install", "-y", pkg)
	}
	return streamCmd("apt-get", "install", "-y", pkg)
}

func runDnf(pkg string, sudo bool) error {
	if pkg == "" {
		return fmt.Errorf("dnf-install: package field is required")
	}
	slog.Info("dnf install", "package", pkg)
	if sudo {
		return streamCmd("sudo", "dnf", "install", "-y", pkg)
	}
	return streamCmd("dnf", "install", "-y", pkg)
}

// runGoInstallFallback implements the [g]o-install choice of the arm64 gap: it
// builds the tool from source via `go install` using an import path declared in
// the recipe. Errors if the recipe declares no go-install import.
func runGoInstallFallback(entry ToolEntry, toolName string, dryRun bool) error {
	imp := goInstallImport(entry)
	if imp == "" {
		return fmt.Errorf("tool %q has no go-install import path to fall back to", toolName)
	}
	if dryRun {
		slog.Info("dry-run plan", "tool", toolName, "strategy", "go-install (arm64 fallback)", "import", imp)
		return nil
	}
	return runGoInstall(imp)
}

func runGoInstall(importPath string) error {
	if importPath == "" {
		return fmt.Errorf("go-install: import field is required")
	}
	slog.Info("go install", "import", importPath)
	return streamCmd("go", "install", importPath)
}

func runScript(script string) error {
	if script == "" {
		return fmt.Errorf("run-script: script field is required")
	}
	slog.Info("running install script")
	cmd := exec.Command("/bin/sh", "-c", script) //nolint:gosec
	cmd.Stdout = newLogWriter(slog.LevelInfo)
	cmd.Stderr = newLogWriter(slog.LevelWarn)
	return cmd.Run()
}

// runLnxNative installs a tool that publishes its own signed apt AND dnf repos.
// It picks the sequence from the platform family and runs it idempotently: the
// existing keyring/repo definition is removed and re-added before installing
// inst.Package, so stale keys or URLs cannot wedge the install (decision D13).
func runLnxNative(inst *InstallSpec, platform Platform) error {
	script, err := lnxNativeScript(inst, platform.Family())
	if err != nil {
		return err
	}
	slog.Info("lnx-native install", "package", inst.Package, "family", platform.Family())
	if inst.Sudo {
		return streamCmd("sudo", "bash", "-c", script)
	}
	return streamCmd("bash", "-c", script)
}

// lnxNativeScript builds the idempotent repo-add + key-import + install shell
// sequence for the given Linux family ("deb" or "rpm"). It is separated from
// execution so the generated commands can be unit-tested.
func lnxNativeScript(inst *InstallSpec, family string) (string, error) {
	if inst.Package == "" {
		return "", fmt.Errorf("lnx-native: package field is required")
	}
	repo := inst.RepoName
	if repo == "" {
		repo = inst.Package
	}

	switch family {
	case "deb":
		if inst.AptKeyURL == "" || inst.AptRepo == "" {
			return "", fmt.Errorf("lnx-native: apt-key-url and apt-repo are required for the deb family")
		}
		keyring := "/etc/apt/keyrings/" + repo + ".gpg"
		list := "/etc/apt/sources.list.d/" + repo + ".list"
		repoLine := strings.ReplaceAll(inst.AptRepo, "{keyring}", keyring)
		return strings.Join([]string{
			"set -e",
			"mkdir -p /etc/apt/keyrings",
			// idempotent: drop any previous keyring + source before re-adding
			fmt.Sprintf("rm -f %q %q", keyring, list),
			fmt.Sprintf("curl -fsSL %q | gpg --dearmor -o %q", inst.AptKeyURL, keyring),
			fmt.Sprintf("chmod 0644 %q", keyring),
			fmt.Sprintf("printf '%%s\\n' %q > %q", repoLine, list),
			"apt-get update",
			fmt.Sprintf("apt-get install -y %q", inst.Package),
		}, "\n"), nil
	case "rpm":
		if inst.DnfRepoURL == "" {
			return "", fmt.Errorf("lnx-native: dnf-repo-url is required for the rpm family")
		}
		repoFile := "/etc/yum.repos.d/" + repo + ".repo"
		parts := []string{
			"set -e",
			"dnf install -y dnf-plugins-core",
			// idempotent: remove a prior .repo named for this tool before re-adding
			fmt.Sprintf("rm -f %q", repoFile),
			fmt.Sprintf("dnf config-manager --add-repo %q", inst.DnfRepoURL),
		}
		if inst.DnfKeyURL != "" {
			parts = append(parts, fmt.Sprintf("rpm --import %q", inst.DnfKeyURL))
		}
		parts = append(parts, fmt.Sprintf("dnf install -y %q", inst.Package))
		return strings.Join(parts, "\n"), nil
	default:
		return "", fmt.Errorf("lnx-native is only valid on linux-deb / linux-rpm platforms")
	}
}

// ---- artifact-based install strategies ----

func installExtractTar(inst *InstallSpec, artifactPath string) error {
	slog.Info("extracting tar archive", "dest", inst.Dest)
	if inst.Sudo {
		return streamCmd("sudo", "tar", "-C", inst.Dest, "-xzf", artifactPath)
	}
	return streamCmd("tar", "-C", inst.Dest, "-xzf", artifactPath)
}

func installExtractTarBinary(inst *InstallSpec, artifactPath, version, osVal, archVal string) error {
	binary := substituteTokens(inst.Binary, version, osVal, archVal)
	slog.Info("extracting binary from tar", "binary", binary, "dest", inst.Dest)

	tmpDir, err := os.MkdirTemp("", "clog-extract-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := streamCmd("tar", "-xzf", artifactPath, "-C", tmpDir); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}

	srcPath := filepath.Join(tmpDir, binary)
	destPath := filepath.Join(inst.Dest, filepath.Base(binary))

	if inst.Sudo {
		return streamCmd("sudo", "cp", srcPath, destPath)
	}
	return streamCmd("cp", srcPath, destPath)
}

func installDeb(inst *InstallSpec, artifactPath string) error {
	slog.Info("installing deb package", "path", artifactPath)
	if inst.Sudo {
		return streamCmd("sudo", "dpkg", "-i", artifactPath)
	}
	return streamCmd("dpkg", "-i", artifactPath)
}

func installRpm(inst *InstallSpec, artifactPath string) error {
	slog.Info("installing rpm package", "path", artifactPath)
	if inst.Sudo {
		return streamCmd("sudo", "dnf", "install", "-y", artifactPath)
	}
	return streamCmd("dnf", "install", "-y", artifactPath)
}

func installCopyBinary(inst *InstallSpec, artifactPath string) error {
	if inst.Chmod != "" {
		mode, err := strconv.ParseUint(inst.Chmod, 8, 32)
		if err != nil {
			return fmt.Errorf("copy-binary: invalid chmod %q: %w", inst.Chmod, err)
		}
		if err := os.Chmod(artifactPath, os.FileMode(mode)); err != nil {
			return fmt.Errorf("copy-binary: chmod: %w", err)
		}
	}
	slog.Info("copying binary", "to", inst.Dest)
	if inst.Sudo {
		return streamCmd("sudo", "cp", artifactPath, inst.Dest)
	}
	return streamCmd("cp", artifactPath, inst.Dest)
}

// streamCmd runs a command and streams its output through slog.
func streamCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...) //nolint:gosec
	cmd.Stdout = newLogWriter(slog.LevelInfo)
	cmd.Stderr = newLogWriter(slog.LevelWarn)
	return cmd.Run()
}

// logWriter implements io.Writer by logging each line at the given level.
type logWriter struct{ level slog.Level }

func newLogWriter(level slog.Level) *logWriter { return &logWriter{level: level} }

func (lw *logWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			slog.Log(nil, lw.level, line) //nolint:sloglint
		}
	}
	return len(p), nil
}
