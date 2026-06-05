//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package latest

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// tinygoCmd gets the latest TinyGo release information for the current platform
var tinygoCmd = &cobra.Command{
	Use:   "tinygo",
	Short: "Get the latest TinyGo release for current platform",
	Long: `Get the latest TinyGo release information for the current platform.

This command fetches the latest TinyGo release from GitHub and returns
a platform-specific version string like "0.39.0_amd64.deb" or "0.39.0_armhf.deb".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine architecture
		arch := "armhf"
		if runtime.GOARCH == "amd64" {
			arch = "amd64"
		}

		// Fetch latest release from GitHub API
		resp, err := http.Get("https://api.github.com/repos/tinygo-org/tinygo/releases/latest")
		if err != nil {
			slog.Error("failed to fetch TinyGo releases", "error", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("GitHub API request failed", "status_code", resp.StatusCode)
			return fmt.Errorf("GitHub API request failed with status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("failed to read response body", "error", err)
			return err
		}

		var release GitHubRelease
		if err := json.Unmarshal(body, &release); err != nil {
			slog.Error("failed to parse JSON response", "error", err)
			return err
		}

		// Extract version number (remove 'v' prefix if present)
		version := strings.TrimPrefix(release.TagName, "v")

		// Find the matching .deb file for the current architecture
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, "linux") &&
				strings.Contains(asset.Name, arch) &&
				strings.HasSuffix(asset.Name, ".deb") {
				// Format as version_arch.deb
				fmt.Printf("%s_%s.deb\n", version, arch)
				return nil
			}
		}

		// If no matching asset found, return just the version with arch
		fmt.Printf("%s_%s.deb\n", version, arch)
		return nil
	},
}

func init() {
	// Add tinygo subcommand to the main Latest command
	Command.AddCommand(tinygoCmd)
}
