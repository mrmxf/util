//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// npmRegistry and nodeDist are the upstream endpoints; tests point them at a
// local server.
var (
	npmRegistry = "https://registry.npmjs.org"
	nodeDist    = "https://nodejs.org/dist"
)

// resolveNpmLatest returns the `latest` dist-tag of spec.Package from the npm
// registry. Used for npm-distributed CLIs (e.g. wrangler) whose GitHub repo is a
// monorepo with too many tags for the github-tags strategy to page through.
func resolveNpmLatest(spec *VersionSpec) (string, error) {
	if spec.Package == "" {
		return "", fmt.Errorf("npm-latest: package field is required")
	}
	var doc struct {
		Version string `json:"version"`
	}
	url := npmRegistry + "/" + spec.Package + "/latest"
	if err := getJSON(url, &doc); err != nil {
		return "", fmt.Errorf("npm-latest: %w", err)
	}
	if doc.Version == "" {
		return "", fmt.Errorf("npm-latest: %s has no version", url)
	}
	logVersionMatch(doc.Version, "npm registry dist-tag latest", "package", spec.Package)
	return doc.Version, nil
}

// resolveNodeLTS returns the newest Node.js LTS release (without the "v"), from
// nodejs.org's release index, which lists releases newest first.
func resolveNodeLTS(_ *VersionSpec) (string, error) {
	var releases []struct {
		Version string `json:"version"`
		LTS     any    `json:"lts"` // false, or the LTS codename
	}
	url := nodeDist + "/index.json"
	if err := getJSON(url, &releases); err != nil {
		return "", fmt.Errorf("node-lts: %w", err)
	}
	for _, r := range releases {
		if name, ok := r.LTS.(string); ok && name != "" {
			v := strings.TrimPrefix(r.Version, "v")
			logVersionMatch(v, "nodejs.org release index (newest LTS)", "codename", name)
			return v, nil
		}
	}
	return "", fmt.Errorf("node-lts: no LTS release in %s", url)
}

// getJSON GETs url and decodes the JSON body into v.
func getJSON(url string, v any) error {
	resp, err := http.Get(url) //nolint:gosec // fixed upstream endpoints
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

// runNpmInstall installs an npm package globally at version (latest when the
// version is unknown). Node and npm must already be installed.
func runNpmInstall(inst *InstallSpec, version string) error {
	if inst.Package == "" {
		return fmt.Errorf("npm-install: package field is required")
	}
	spec := inst.Package
	if version != "" && version != unknownVersion {
		spec += "@" + version
	}
	return privileged(inst.Sudo, "npm", "install", "--global", spec)
}
