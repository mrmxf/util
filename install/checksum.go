//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// VerifyChecksum checks the downloaded artifact against a sha256sum-format
// checksums file (`<hex>  <name>` or `<hex> *<name>` per line), as published by
// goreleaser, `clog CI deploy` (github-release) and most GitHub releases.
//
// checksumsURL is fetched; the line whose name equals artifactName must exist
// and its digest must equal the artifact's SHA-256. A missing line is an error,
// not a skip: a recipe that declares checksums promises one for every asset.
func VerifyChecksum(checksumsURL, artifactName, artifactPath string) error {
	want, err := lookupChecksum(checksumsURL, artifactName)
	if err != nil {
		return err
	}
	got, err := fileSHA256(artifactPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s: got sha256 %s, %s lists %s", artifactName, got, checksumsURL, want)
	}
	slog.Info("checksum verified", "asset", artifactName, "sha256", got)
	return nil
}

// lookupChecksum fetches checksumsURL and returns the hex digest listed for name.
func lookupChecksum(checksumsURL, name string) (string, error) {
	resp, err := http.Get(checksumsURL) //nolint:gosec // recipe-provided URL
	if err != nil {
		return "", fmt.Errorf("checksums: GET %s: %w", checksumsURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums: GET %s: HTTP %d", checksumsURL, resp.StatusCode)
	}
	want, err := findChecksum(resp.Body, name)
	if err != nil {
		return "", fmt.Errorf("checksums: %s: %w", checksumsURL, err)
	}
	return want, nil
}

// findChecksum returns the hex digest a sha256sum-format listing gives name.
func findChecksum(r io.Reader, name string) (string, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return "", fmt.Errorf("lists no entry for %s", name)
}

// fileSHA256 returns the hex SHA-256 of the file at path.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("checksum: open %s: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("checksum: read %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
