//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// DetectPlatform determines the current platform by examining runtime.GOOS,
// runtime.GOARCH, and (on Linux) the distribution family.
func DetectPlatform() (Platform, error) {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return PlatformDarwinArm64, nil
		}
		return PlatformDarwinAmd64, nil
	case "linux":
		family := "rpm"
		if isDebianFamily() {
			family = "deb"
		}
		switch runtime.GOARCH {
		case "arm64":
			return Platform(fmt.Sprintf("linux-%s/arm64", family)), nil
		case "amd64":
			return Platform(fmt.Sprintf("linux-%s/amd64", family)), nil
		default:
			return "", fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// isDebianFamily returns true for Debian/Ubuntu and derivatives.
func isDebianFamily() bool {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return true
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") || strings.HasPrefix(line, "ID_LIKE=") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "debian") || strings.Contains(lower, "ubuntu") {
				return true
			}
		}
	}
	return false
}

// Family returns the Linux package family ("deb" or "rpm") for a linux-*
// platform, or "" for non-Linux platforms. Used by the lnx-native strategy to
// choose the apt vs dnf sequence without re-detecting the distro.
func (p Platform) Family() string {
	s := string(p)
	switch {
	case strings.HasPrefix(s, "linux-deb"):
		return "deb"
	case strings.HasPrefix(s, "linux-rpm"):
		return "rpm"
	default:
		return ""
	}
}

// substituteTokens replaces {version}, {os}, and {arch} in s.
func substituteTokens(s, version, osVal, archVal string) string {
	r := strings.NewReplacer(
		"{version}", version,
		"{os}", osVal,
		"{arch}", archVal,
	)
	return r.Replace(s)
}
