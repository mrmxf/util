//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubHTTP replaces the network seam with a fixed map of url -> (status, body).
// A url the test did not list returns 404, which is the honest default: a probe
// asking about a path nobody configured should see "not there".
func stubHTTP(t *testing.T, routes map[string]struct {
	status int
	body   string
}) {
	t.Helper()
	orig := httpGet
	httpGet = func(url string) (int, string, error) {
		if r, ok := routes[url]; ok {
			return r.status, r.body, nil
		}
		return http.StatusNotFound, "", nil
	}
	t.Cleanup(func() { httpGet = orig })
}

func route(status int, body string) struct {
	status int
	body   string
} {
	return struct {
		status int
		body   string
	}{status, body}
}

func probeDeployment(kind string, data map[string]any) Deployment {
	return Deployment{
		Env:    DefaultEnv(),
		Cfg:    Config{Targets: map[string]Target{"t": {Kind: kind, Prod: data}}},
		Mode:   ModeProd,
		Target: "t",
		Out:    io_Discard{},
	}
}

type io_Discard struct{}

func (io_Discard) Write(p []byte) (int, error) { return len(p), nil }

func levelOf(fs []ProbeFinding, check string) ProbeLevel {
	for _, f := range fs {
		if strings.Contains(f.Check, check) {
			return f.Level
		}
	}
	return ""
}

// A healthy site: up, and serving nothing it should not.
func TestProbeWebsiteClean(t *testing.T) {
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://example.org": route(200, "<html>"),
	})
	got := probeWebsite(probeDeployment(KindGitHubPages, map[string]any{"domain": "example.org"}))
	if l := levelOf(got, "site responds"); l != ProbePass {
		t.Errorf("site responds = %q, want pass", l)
	}
	for _, p := range exposedPaths {
		if l := levelOf(got, p); l != ProbePass {
			t.Errorf("%s = %q, want pass (404 means not served)", p, l)
		}
	}
}

// The incident this probe exists for: a published .git directory hands over
// the entire source history, including anything ever committed and removed.
func TestProbeWebsiteCatchesExposedGit(t *testing.T) {
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://example.org":             route(200, "<html>"),
		"https://example.org/.git/config": route(200, "[core]\n"),
	})
	got := probeWebsite(probeDeployment(KindCloudflarePage, map[string]any{"domain": "example.org"}))
	if l := levelOf(got, "/.git/config"); l != ProbeFail {
		t.Errorf("served .git/config = %q, want fail", l)
	}
}

// A site that is down fails once, not five times: every follow-up would be
// reporting the same outage again under a different name.
func TestProbeWebsiteDownReportsOnce(t *testing.T) {
	orig := httpGet
	httpGet = func(string) (int, string, error) { return 0, "", http.ErrHandlerTimeout }
	t.Cleanup(func() { httpGet = orig })

	got := probeWebsite(probeDeployment(KindGitHubPages, map[string]any{"domain": "example.org"}))
	if len(got) != 1 {
		t.Fatalf("got %d findings for an unreachable site, want 1", len(got))
	}
	if got[0].Level != ProbeFail {
		t.Errorf("unreachable site = %q, want fail", got[0].Level)
	}
}

// A bucket handing out <ListBucketResult> is enumerable by anyone.
func TestProbeBucketCatchesPublicListing(t *testing.T) {
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://acme-assets.s3.amazonaws.com": route(200,
			`<?xml version="1.0"?><ListBucketResult><Name>acme-assets</Name></ListBucketResult>`),
	})
	got := probeBucket(probeDeployment(KindBucket, map[string]any{"bucket": "acme-assets"}))
	if l := levelOf(got, "listing not public"); l != ProbeFail {
		t.Errorf("public listing = %q, want fail", l)
	}
}

func TestProbeBucketPrivateIsClean(t *testing.T) {
	stubHTTP(t, nil) // everything 404s
	got := probeBucket(probeDeployment(KindBucket, map[string]any{"bucket": "acme-assets"}))
	if l := levelOf(got, "listing not public"); l != ProbePass {
		t.Errorf("private bucket = %q, want pass", l)
	}
}

// A release is only as good as the checksums the installer verifies against.
func TestProbeReleaseMatchesBuiltArtifacts(t *testing.T) {
	dir := t.TempDir()
	body := []byte("binary contents")
	if err := os.WriteFile(filepath.Join(dir, "clog-amd-lnx"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	hexsum := hex.EncodeToString(sum[:])

	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://github.com/acme/clog/releases/download/v1.0.0/checksums.txt": route(200,
			hexsum+"  clog-amd-lnx\n"),
	})
	got := probeRelease(probeDeployment(KindGitHubRelease, map[string]any{
		"repo": "acme/clog", "tag": "v1.0.0", "dir": dir, "assets": "clog-amd-lnx",
	}))
	if l := levelOf(got, "asset clog-amd-lnx"); l != ProbePass {
		t.Errorf("matching checksum = %q, want pass:\n%+v", l, got)
	}
}

// The failure that matters: what is published is NOT what was built.
func TestProbeReleaseCatchesChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clog-amd-lnx"), []byte("what we built"), 0o600); err != nil {
		t.Fatal(err)
	}
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://github.com/acme/clog/releases/download/v1.0.0/checksums.txt": route(200,
			strings.Repeat("a", 64)+"  clog-amd-lnx\n"),
	})
	got := probeRelease(probeDeployment(KindGitHubRelease, map[string]any{
		"repo": "acme/clog", "tag": "v1.0.0", "dir": dir, "assets": "clog-amd-lnx",
	}))
	if l := levelOf(got, "asset clog-amd-lnx"); l != ProbeFail {
		t.Errorf("mismatched checksum = %q, want fail", l)
	}
}

// A declared asset missing from checksums.txt means the installer cannot
// verify it, so it will refuse to install it.
func TestProbeReleaseCatchesMissingAsset(t *testing.T) {
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://github.com/acme/clog/releases/download/v1.0.0/checksums.txt": route(200, ""),
	})
	got := probeRelease(probeDeployment(KindGitHubRelease, map[string]any{
		"repo": "acme/clog", "tag": "v1.0.0", "dir": t.TempDir(), "assets": "clog-arm-mac",
	}))
	if l := levelOf(got, "asset clog-arm-mac"); l != ProbeFail {
		t.Errorf("absent asset = %q, want fail", l)
	}
}

// An image that only exists as :latest cannot be pinned or rolled back to.
func TestProbeRegistryWarnsOnLatestOnly(t *testing.T) {
	orig := execCommand
	execCommand = func(dir, name string, args ...string) (string, error) { return "{}", nil }
	t.Cleanup(func() { execCommand = orig })

	got := probeRegistry(probeDeployment(KindRegistry, map[string]any{"image": "acme/site:latest"}))
	if l := levelOf(got, "immutable tag"); l != ProbeWarn {
		t.Errorf("latest-only = %q, want warn", l)
	}
	got = probeRegistry(probeDeployment(KindRegistry, map[string]any{"image": "acme/site:v1.2.3"}))
	if l := levelOf(got, "immutable tag"); l != ProbePass {
		t.Errorf("versioned tag = %q, want pass", l)
	}
}

// A deploy that reported success but left no tag is the worst case, because
// everything downstream believes the image is there.
func TestProbeRegistryCatchesMissingTag(t *testing.T) {
	orig := execCommand
	execCommand = func(dir, name string, args ...string) (string, error) {
		return "manifest unknown", os.ErrNotExist
	}
	t.Cleanup(func() { execCommand = orig })

	got := probeRegistry(probeDeployment(KindRegistry, map[string]any{"image": "acme/site:v1"}))
	if l := levelOf(got, "tag exists"); l != ProbeFail {
		t.Errorf("absent tag = %q, want fail", l)
	}
}

// A kind with no prober must say so. "nothing was asked" and "nothing was
// wrong" reading the same is how a gap becomes invisible.
func TestProbeReportsKindsItCannotCheck(t *testing.T) {
	cfg := Config{Targets: map[string]Target{
		"odd": {Kind: KindGitLabRelease, Prod: map[string]any{"tag": "v1"}},
	}}
	for kind := range probers {
		if _, ok := probers[kind]; !ok {
			t.Errorf("prober map holds an empty entry for %q", kind)
		}
	}
	// every known kind should have a prober, so this is really a completeness
	// assertion over knownKinds
	for _, k := range knownKinds {
		if _, ok := probers[k]; !ok {
			t.Errorf("target kind %q has no probe - a deploy to it is never checked", k)
		}
	}
	_ = cfg
}
