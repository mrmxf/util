//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// exposedPaths are the things a static host should never serve. Each one is a
// real incident shape rather than a theoretical one: a published .git lets
// anyone reconstruct the whole source history including deleted secrets, and a
// served .env is the credentials themselves.
var exposedPaths = []string{"/.git/config", "/.env", "/.wrangler/config.json"}

// probeWebsite asks a served site whether it is up, and whether it is serving
// anything it should not. It covers all three Pages kinds: what is being
// checked is the HTTP surface, which does not vary by who hosts it.
func probeWebsite(d Deployment) []ProbeFinding {
	base, err := websiteURL(d)
	if err != nil {
		return []ProbeFinding{warn(d, "url known", err.Error())}
	}
	var out []ProbeFinding

	status, _, err := httpGet(base)
	switch {
	case err != nil:
		out = append(out, fail(d, "site responds", fmt.Sprintf("GET %s: %v", base, err)))
		// If the site is unreachable there is nothing to ask about its
		// contents, and every follow-up would report the same outage twice.
		return out
	case status != http.StatusOK:
		out = append(out, fail(d, "site responds", fmt.Sprintf("GET %s returned %d, want 200", base, status)))
	default:
		out = append(out, pass(d, "site responds", fmt.Sprintf("GET %s returned 200", base)))
	}
	if status == http.StatusOK {
		out = append(out, probeServedBuild(d, base))
	}

	for _, p := range exposedPaths {
		u := base + p
		status, _, err := httpGet(u)
		switch {
		case err != nil:
			out = append(out, warn(d, "not serving "+p, fmt.Sprintf("GET %s: %v", u, err)))
		case status == http.StatusOK:
			out = append(out, fail(d, "not serving "+p,
				fmt.Sprintf("GET %s returned 200 - this path is public", u)))
		default:
			out = append(out, pass(d, "not serving "+p, fmt.Sprintf("returns %d", status)))
		}
	}
	return out
}

// websiteURL works out what to ask. `domain` is the honest answer when config
// gives one; otherwise the Pages kinds have predictable addresses.
func websiteURL(d Deployment) (string, error) {
	if v, err := d.Get("url", false); err == nil && v != "" {
		return trimURL(v), nil
	}
	if v, err := d.Get("domain", false); err == nil && v != "" {
		if strings.HasPrefix(v, "http") {
			return trimURL(v), nil
		}
		return "https://" + trimURL(v), nil
	}
	if v, err := ModeGet(d.Mode, "base-url", false); err == nil && v != "" {
		return trimURL(v), nil
	}
	return "", fmt.Errorf("no domain, url or ci.modes.%s.base-url to probe", d.Mode)
}

// probeBucket asks whether an object store is handing out its contents. The
// failure this exists for is a bucket made readable "just to test something"
// and never closed again.
func probeBucket(d Deployment) []ProbeFinding {
	var out []ProbeFinding

	endpoint, err := d.Get("url", false)
	if err != nil || endpoint == "" {
		bucket, _ := d.Get("bucket", false)
		if bucket == "" {
			return []ProbeFinding{warn(d, "url known",
				"no bucket or url in the target data, so public access could not be checked")}
		}
		// The S3 website form is the one that serves a listing to the world.
		endpoint = "https://" + bucket + ".s3.amazonaws.com"
	}
	endpoint = trimURL(endpoint)

	status, body, err := httpGet(endpoint)
	switch {
	case err != nil:
		out = append(out, warn(d, "listing not public", fmt.Sprintf("GET %s: %v", endpoint, err)))
	case status == http.StatusOK && strings.Contains(body, "<ListBucketResult"):
		out = append(out, fail(d, "listing not public",
			fmt.Sprintf("GET %s returned a bucket listing - the contents are enumerable by anyone", endpoint)))
	case status == http.StatusOK:
		out = append(out, warn(d, "listing not public",
			fmt.Sprintf("GET %s returned 200; it serves something, but not a listing", endpoint)))
	default:
		out = append(out, pass(d, "listing not public", fmt.Sprintf("returns %d", status)))
	}

	for _, p := range []string{"/.env", "/.git/config"} {
		u := endpoint + p
		status, _, err := httpGet(u)
		switch {
		case err != nil:
			out = append(out, warn(d, "no public "+p, err.Error()))
		case status == http.StatusOK:
			out = append(out, fail(d, "no public "+p, fmt.Sprintf("GET %s returned 200", u)))
		default:
			out = append(out, pass(d, "no public "+p, fmt.Sprintf("returns %d", status)))
		}
	}
	return out
}

// probeRegistry asks whether the image a deploy claims to have pushed is
// actually there, under the immutable tag rather than only under a moving one.
// `latest` alone is not a deployment anyone can roll back to.
func probeRegistry(d Deployment) []ProbeFinding {
	image, err := d.Get("image", true)
	if err != nil {
		return []ProbeFinding{warn(d, "image known", err.Error())}
	}
	var out []ProbeFinding

	if _, err := execCommand(".", "docker", "manifest", "inspect", image); err != nil {
		out = append(out, fail(d, "tag exists",
			fmt.Sprintf("docker manifest inspect %s failed - the deploy reported success but the tag is not there: %v", image, err)))
		return out
	}
	out = append(out, pass(d, "tag exists", image))

	// A reference with no tag, or only :latest, cannot be pinned by a consumer
	// and cannot be rolled back to.
	ref := image
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		tag := ref[i+1:]
		if tag == "latest" || tag == "latest-dev" {
			out = append(out, warn(d, "immutable tag",
				fmt.Sprintf("%s is only tagged %q, so nothing can pin or roll back to this build", image, tag)))
		} else {
			out = append(out, pass(d, "immutable tag", tag))
		}
	} else {
		out = append(out, warn(d, "immutable tag", image+" has no tag, so it resolves to :latest"))
	}
	return out
}

// probeRelease asks whether what is published matches what was built. Every
// declared asset must be present in the published checksums file, with the
// hash of the local artifact - which is what the installer will verify before
// it runs the binary.
func probeRelease(d Deployment) []ProbeFinding {
	tag, err := d.Get("tag", true)
	if err != nil {
		return []ProbeFinding{warn(d, "tag known", err.Error())}
	}
	dir, err := d.Get("dir", true)
	if err != nil {
		return []ProbeFinding{warn(d, "dir known", err.Error())}
	}
	assetList, err := d.Get("assets", true)
	if err != nil {
		return []ProbeFinding{warn(d, "assets known", err.Error())}
	}
	repo, _ := d.Get("repo", false)
	if repo == "" {
		if repo, err = repoSlug(d.Env); err != nil || repo == "" {
			return []ProbeFinding{warn(d, "repo known", "cannot work out which repo the release is in")}
		}
	}

	base := fmt.Sprintf("https://github.com/%s/releases/download/%s", repo, tag)
	status, body, err := httpGet(base + "/" + releaseChecksums)
	if err != nil {
		return []ProbeFinding{fail(d, "checksums published", fmt.Sprintf("GET %s: %v", base, err))}
	}
	if status != http.StatusOK {
		return []ProbeFinding{fail(d, "checksums published",
			fmt.Sprintf("%s/%s returned %d - the installer refuses a release it cannot verify",
				base, releaseChecksums, status))}
	}

	published := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 {
			published[strings.TrimPrefix(f[1], "*")] = f[0]
		}
	}
	out := []ProbeFinding{pass(d, "checksums published", fmt.Sprintf("%d entries", len(published)))}

	for _, a := range strings.Fields(assetList) {
		want, ok := published[a]
		if !ok {
			out = append(out, fail(d, "asset "+a, "declared but absent from the published checksums.txt"))
			continue
		}
		got, err := sha256File(filepath.Join(dir, a))
		if err != nil {
			if os.IsNotExist(err) {
				out = append(out, warn(d, "asset "+a, "published, but the local copy is gone so it cannot be compared"))
				continue
			}
			out = append(out, warn(d, "asset "+a, err.Error()))
			continue
		}
		if got != want {
			out = append(out, fail(d, "asset "+a,
				fmt.Sprintf("published checksum %s does not match the built artifact %s", want[:12], got[:12])))
			continue
		}
		out = append(out, pass(d, "asset "+a, "published checksum matches what was built"))
	}
	return out
}
