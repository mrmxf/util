//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"log/slog"
	"os/exec"
)

// VerifySLSA checks the SLSA provenance of a downloaded artifact using
// the slsa-verifier CLI. If slsa-verifier is not installed the check is
// skipped with a warning so that users without it are not blocked.
// Returns an error when verification fails.
func VerifySLSA(spec *SLSASpec, artifactPath, version, osVal, archVal string) error {
	if spec == nil {
		return nil
	}

	if _, err := exec.LookPath("slsa-verifier"); err != nil {
		slog.Warn("slsa-verifier not found; skipping SLSA check",
			"hint", "clog install slsa-verifier",
			"artifact", artifactPath)
		return nil
	}

	provenanceURL := substituteTokens(spec.URLTemplate, version, osVal, archVal)
	slog.Info("downloading SLSA provenance", "url", provenanceURL)

	provenancePath, cleanup, err := downloadToTemp(provenanceURL, ".intoto.jsonl")
	if err != nil {
		return err
	}
	defer cleanup()

	slog.Info("verifying SLSA provenance", "artifact", artifactPath, "source-uri", spec.SourceURI)
	return streamCmd("slsa-verifier", "verify-artifact", artifactPath,
		"--provenance-path", provenancePath,
		"--source-uri", spec.SourceURI,
	)
}
