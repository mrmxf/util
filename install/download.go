//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// downloadArtifact fetches the tool artifact according to dl.Strategy.
// Returns the local path and a cleanup func that removes the temp file.
func downloadArtifact(dl *DownloadSpec, version, osVal, archVal string) (path string, cleanup func(), err error) {
	switch dl.Strategy {
	case "url-template":
		return downloadURLTemplate(dl, version, osVal, archVal)
	case "github-release-asset":
		return downloadGitHubReleaseAsset(dl, version)
	default:
		return "", noop, fmt.Errorf("download strategy %q not implemented", dl.Strategy)
	}
}

func downloadURLTemplate(dl *DownloadSpec, version, osVal, archVal string) (string, func(), error) {
	rawURL := substituteTokens(dl.URL, version, osVal, archVal)
	slog.Info("downloading", "url", rawURL)
	return downloadToTemp(rawURL, extFromURL(rawURL))
}

func downloadGitHubReleaseAsset(dl *DownloadSpec, version string) (string, func(), error) {
	name, url, err := resolveGitHubReleaseAsset(dl, version)
	if err != nil {
		return "", noop, err
	}
	slog.Info("matched release asset", "name", name)
	return downloadToTemp(url, extFromURL(name))
}

// resolveGitHubReleaseAsset finds the release asset matching dl.Filter and
// returns its name and download URL without fetching it. When version is known
// the release tagged v{version} (or {version}) is used, so a version pinned by
// go.mod / module.yaml is honoured; otherwise the latest release is used.
func resolveGitHubReleaseAsset(dl *DownloadSpec, version string) (name, url string, err error) {
	if dl.Repo == "" {
		return "", "", fmt.Errorf("github-release-asset: repo field is required")
	}
	base := fmt.Sprintf("%s/repos/%s/releases/", githubAPI, dl.Repo)
	candidates := []string{base + "latest"}
	if version != unknownVersion {
		candidates = []string{base + "tags/v" + version, base + "tags/" + version}
	}

	for _, apiURL := range candidates {
		vlog("  looking up release", "api", apiURL)
		resp, err := githubGet(apiURL)
		if err != nil {
			return "", "", fmt.Errorf("github-release-asset: GET %s: %w", apiURL, err)
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return "", "", fmt.Errorf("github-release-asset: %s returned HTTP %d", apiURL, resp.StatusCode)
		}

		var release struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}
		err = json.NewDecoder(resp.Body).Decode(&release)
		resp.Body.Close()
		if err != nil {
			return "", "", fmt.Errorf("github-release-asset: decode %s: %w", apiURL, err)
		}

		for _, asset := range release.Assets {
			if matchesAllFilters(asset.Name, dl.Filter, version) {
				vlog("  release asset matched", "tag", release.TagName, "filter", dl.Filter)
				return asset.Name, asset.BrowserDownloadURL, nil
			}
		}
		names := make([]string, len(release.Assets))
		for i, a := range release.Assets {
			names[i] = a.Name
		}
		return "", "", fmt.Errorf("github-release-asset: no asset in %s@%s matched filters %v\navailable: %v", dl.Repo, release.TagName, dl.Filter, names)
	}
	return "", "", fmt.Errorf("github-release-asset: no release in %s for version %q", dl.Repo, version)
}

// matchesAllFilters returns true when assetName contains every filter string
// (with {version} substituted in each filter).
func matchesAllFilters(assetName string, filters []string, version string) bool {
	for _, f := range filters {
		if !strings.Contains(assetName, strings.ReplaceAll(f, "{version}", version)) {
			return false
		}
	}
	return true
}

// downloadToTemp fetches url into a temporary file with the given extension.
// The caller must call cleanup() when the file is no longer needed.
func downloadToTemp(url, ext string) (path string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "clog-install-*"+ext)
	if err != nil {
		return "", noop, fmt.Errorf("create temp file: %w", err)
	}
	cleanup = func() { os.Remove(tmp.Name()) }

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		cleanup()
		return "", noop, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		cleanup()
		return "", noop, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("write download to temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("close temp file: %w", err)
	}

	if fi, statErr := os.Stat(tmp.Name()); statErr == nil {
		slog.Info("downloaded", "path", tmp.Name(), "bytes", fi.Size())
	}
	return tmp.Name(), cleanup, nil
}

func extFromURL(u string) string {
	if strings.HasSuffix(u, ".tar.gz") {
		return ".tar.gz"
	}
	return filepath.Ext(u)
}

func noop() {}
