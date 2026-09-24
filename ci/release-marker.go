//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// releaseMarker is a one-line file the website deployers publish at the site
// root, holding the version that was deployed. It exists so a probe can answer
// the question a 200 cannot: WHICH build is live.
//
// Not hypothetical. pihuw v0.4.10-v0.4.14 were each pushed to gh-pages while
// Pages served a workflow build from April; every deploy "succeeded" and every
// probe passed, because the old site still returned 200.
const releaseMarker = "clog-release.txt"

// writeReleaseMarker puts the deployed version into the publish dir.
func writeReleaseMarker(dir, version string) error {
	if version == "" {
		return nil
	}
	if err := os.WriteFile(filepath.Join(dir, releaseMarker), []byte(version+"\n"), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", releaseMarker, err)
	}
	return nil
}

// A static host does not serve a push at once: GitHub Pages runs a build, and
// every host sits behind a CDN. So the probe polls rather than asking once, and
// only a site still serving another version after the whole wait is a failure.
var (
	markerAttempts = 20               // × markerPoll = 5 minutes
	markerPoll     = 15 * time.Second //
	probeSleep     = time.Sleep       // replaced in tests
)

// probeServedBuild asks the live site which version it is serving.
func probeServedBuild(d Deployment, base string) ProbeFinding {
	const check = "serving this build"
	want := ReleaseVersion()
	if want == "" {
		return warn(d, check, "this checkout has no version to compare against")
	}
	// The query string only defeats CDN caching; the file is static.
	u := base + "/" + releaseMarker + "?clog=" + url.QueryEscape(want)

	var status int
	var got string
	var err error
	for i := 0; i < markerAttempts; i++ {
		if i > 0 {
			probeSleep(markerPoll)
		}
		var body string
		status, body, err = httpGet(u)
		got = strings.TrimSpace(body)
		if err == nil && status == http.StatusOK && got == want {
			return pass(d, check, fmt.Sprintf("%s serves %s", base, want))
		}
	}
	waited := time.Duration(markerAttempts-1) * markerPoll
	switch {
	case err != nil:
		return warn(d, check, fmt.Sprintf("GET %s: %v", u, err))
	case status == http.StatusOK:
		return fail(d, check, fmt.Sprintf(
			"%s still serves %s after %s, but this run deployed %s - is the host serving a different branch, project or build?",
			base, got, waited, want))
	default:
		return warn(d, check, fmt.Sprintf(
			"/%s returned %d after %s, so which build is live cannot be told (deployed before clog v1.0.3, not by clog, or not being served)",
			releaseMarker, status, waited))
	}
}
