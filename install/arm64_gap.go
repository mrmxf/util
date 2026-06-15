//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// errAborted is returned when the user aborts an arm64-gap prompt.
var errAborted = fmt.Errorf("install aborted by user")

// gapResolution is the outcome of resolving an unsupported platform.
type gapResolution struct {
	platform  Platform // effective platform to install for ("[i]nstall amd")
	goInstall bool     // run the go-install fallback instead ("[g]o-install")
}

// resolveUnsupported decides what to do when the requested platform is marked
// unsupported. Per decision D15, for a Linux arm64 target whose amd64 sibling
// IS supported, an interactive shell is prompted to (i) install the amd64
// build, (g) fall back to go-install, or (A)bort. A non-interactive shell, or
// any unsupported platform without a viable amd64 sibling, fails-unsupported.
func resolveUnsupported(entry ToolEntry, toolName string, platform Platform) (gapResolution, error) {
	sibling := amd64Sibling(platform)
	siblingOK := false
	if sibling != "" {
		if pe, ok := entry.Platforms[sibling]; ok && !pe.Unsupported {
			siblingOK = true
		}
	}

	if !stdinIsInteractive() || !siblingOK {
		return gapResolution{}, fmt.Errorf("tool %q is not supported on %s", toolName, platform)
	}

	switch promptArm64Gap(toolName, platform, sibling) {
	case 'i':
		slog.Warn("installing amd64 build on arm64 host (runs under emulation)", "tool", toolName, "platform", sibling)
		return gapResolution{platform: sibling}, nil
	case 'g':
		return gapResolution{goInstall: true}, nil
	default:
		return gapResolution{}, errAborted
	}
}

// amd64Sibling returns the amd64 platform key for a linux arm64 platform, or ""
// for any other platform. ("install the amd64 build" only makes sense there.)
func amd64Sibling(p Platform) Platform {
	switch p {
	case PlatformLinuxDebArm64:
		return PlatformLinuxDebAmd64
	case PlatformLinuxRpmArm64:
		return PlatformLinuxRpmAmd64
	default:
		return ""
	}
}

// stdinIsInteractive reports whether stdin is a terminal. It is a var so tests
// can force non-interactive behaviour.
var stdinIsInteractive = isInteractive

// isInteractive reports whether stdin is a terminal (character device).
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptArm64Gap asks the user how to handle the arm64 gap and returns the
// lower-cased first character of their answer ('i', 'g', or 'a'). An empty
// answer defaults to abort.
func promptArm64Gap(toolName string, platform, sibling Platform) byte {
	fmt.Fprintf(os.Stderr,
		"%s has no %s build.\n  [i]nstall the %s build (runs under emulation)\n  [g]o-install from source\n  [A]bort\nchoice [i/g/A]: ",
		toolName, platform, sibling)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return 'a'
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer == "" {
		return 'a'
	}
	return answer[0]
}

// goInstallImport finds a go-install import path declared anywhere in the entry
// (top-level install or any platform install with strategy go-install), so the
// [g]o-install fallback can build the tool from source.
func goInstallImport(entry ToolEntry) string {
	if entry.Install != nil && entry.Install.Strategy == "go-install" && entry.Install.Import != "" {
		return entry.Install.Import
	}
	for _, pe := range entry.Platforms {
		if pe.Install != nil && pe.Install.Strategy == "go-install" && pe.Install.Import != "" {
			return pe.Install.Import
		}
	}
	return ""
}
