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
	if dl.Repo == "" {
		return "", noop, fmt.Errorf("github-release-asset: repo field is required")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", dl.Repo)
	resp, err := http.Get(apiURL) //nolint:gosec
	if err != nil {
		return "", noop, fmt.Errorf("github-release-asset: GET %s: %w", apiURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", noop, fmt.Errorf("github-release-asset: %s returned HTTP %d", apiURL, resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", noop, fmt.Errorf("github-release-asset: decode %s: %w", apiURL, err)
	}

	for _, asset := range release.Assets {
		if matchesAllFilters(asset.Name, dl.Filter, version) {
			slog.Info("matched release asset", "name", asset.Name)
			return downloadToTemp(asset.BrowserDownloadURL, extFromURL(asset.Name))
		}
	}

	names := make([]string, len(release.Assets))
	for i, a := range release.Assets {
		names[i] = a.Name
	}
	return "", noop, fmt.Errorf("github-release-asset: no asset in %s matched filters %v\navailable: %v", dl.Repo, dl.Filter, names)
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
