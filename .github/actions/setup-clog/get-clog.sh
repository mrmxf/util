#!/usr/bin/env bash
#  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
#  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
#
#   get-clog.sh — pinned, checksum-verified clog installer
#
#   Replaces the legacy `eval "$(secrets.get_clog)"` bootstrap (audit F1): the
#   clog version is an explicit input (a checked-in .clog-version file), the
#   binary is downloaded from a GitHub Release, and its sha256 is verified
#   before it is ever run. Nothing executes secret *contents*.
#
#   The same script is the single bootstrap brain for every environment:
#     - a developer laptop          ./get-clog.sh
#     - the GitHub setup-clog action (wraps this script)
#     - a GitLab CI job             source ./get-clog.sh
#
#   Configuration (all via environment, with sensible defaults):
#     CLOG_VERSION        clog tag to install (e.g. v0.10.11). If unset, read
#                         from CLOG_VERSION_FILE.
#     CLOG_VERSION_FILE   path to the pin file (default: ./.clog-version)
#     CLOG_REPO           GitHub owner/repo holding the release
#                         (default: mrmxf/clog-sample — the public binary)
#     CLOG_INSTALL_DIR    where to install the `clog` binary
#                         (default: $RUNNER_TEMP/clog/bin in CI, else
#                          $HOME/.local/bin)
#     CLOG_TOKEN          token for private-repo downloads (optional)
#     CLOG_SKIP_VERIFY    set to 1 to skip checksum verification (NOT advised)
#
#   On success it prints the install dir on stdout (for $GITHUB_PATH) and a log
#   line to stderr.

set -euo pipefail

log() { printf '%s [get-clog] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

# --- resolve version -------------------------------------------------------
CLOG_VERSION_FILE="${CLOG_VERSION_FILE:-.clog-version}"
if [ -z "${CLOG_VERSION:-}" ]; then
  [ -f "$CLOG_VERSION_FILE" ] || die "no CLOG_VERSION and no pin file at '$CLOG_VERSION_FILE'"
  # first non-comment, non-blank line
  CLOG_VERSION="$(grep -vE '^\s*(#|$)' "$CLOG_VERSION_FILE" | head -n1 | tr -d '[:space:]')"
  [ -n "$CLOG_VERSION" ] || die "pin file '$CLOG_VERSION_FILE' is empty"
fi
CLOG_REPO="${CLOG_REPO:-mrmxf/clog-sample}"

# --- detect platform → asset name (matches clog release convention) --------
case "$(uname -s)" in
  Linux*|CYGWIN*|MINGW*|MSYS_NT*) os="lnx" ;;
  Darwin*)                        os="mac" ;;
  *) die "unsupported OS $(uname -s)" ;;
esac
case "$(uname -m)" in
  x86_64*|amd64*) cpu="amd" ;;
  arm64|aarch64*) cpu="arm" ;;
  *) die "unsupported ARCH $(uname -m)" ;;
esac
asset="clog-${cpu}-${os}"

# --- install dir -----------------------------------------------------------
if [ -z "${CLOG_INSTALL_DIR:-}" ]; then
  if [ -n "${RUNNER_TEMP:-}" ]; then
    CLOG_INSTALL_DIR="$RUNNER_TEMP/clog/bin"
  else
    CLOG_INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$CLOG_INSTALL_DIR"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

base="https://github.com/${CLOG_REPO}/releases/download/${CLOG_VERSION}"
log "installing clog ${CLOG_VERSION} (${asset}) from ${CLOG_REPO} → ${CLOG_INSTALL_DIR}"

# curl with optional auth for private repos. --fail makes HTTP errors fatal.
fetch() { # fetch <url> <dest>
  local auth=()
  [ -n "${CLOG_TOKEN:-}" ] && auth=(--header "Authorization: Bearer ${CLOG_TOKEN}")
  curl --fail --location --silent --show-error "${auth[@]}" "$1" --output "$2"
}

fetch "${base}/${asset}" "${tmp}/clog" \
  || die "download failed: ${base}/${asset} (is ${CLOG_VERSION} released for ${CLOG_REPO}?)"

# --- verify checksum -------------------------------------------------------
if [ "${CLOG_SKIP_VERIFY:-0}" = "1" ]; then
  log "WARNING: CLOG_SKIP_VERIFY=1 — skipping checksum verification"
else
  if fetch "${base}/checksums.txt" "${tmp}/checksums.txt"; then
    want="$(grep -E "[[:space:]]\*?${asset}\$" "${tmp}/checksums.txt" | awk '{print $1}' | head -n1)"
    [ -n "$want" ] || die "checksums.txt has no entry for ${asset}"
    got="$(sha256sum "${tmp}/clog" | awk '{print $1}')"
    [ "$want" = "$got" ] || die "checksum mismatch for ${asset}: want ${want}, got ${got}"
    log "checksum verified (${got})"
  else
    die "could not fetch checksums.txt — refusing to install unverified binary (set CLOG_SKIP_VERIFY=1 to override)"
  fi
fi

# --- optional SLSA provenance verification ---------------------------------
# If slsa-verifier is on PATH and the release ships provenance, verify it.
if command -v slsa-verifier >/dev/null 2>&1; then
  if fetch "${base}/${asset}.intoto.jsonl" "${tmp}/${asset}.intoto.jsonl"; then
    log "verifying SLSA provenance"
    slsa-verifier verify-artifact "${tmp}/clog" \
      --provenance-path "${tmp}/${asset}.intoto.jsonl" \
      --source-uri "github.com/${CLOG_REPO}" \
      --source-tag "${CLOG_VERSION}" \
      || die "SLSA provenance verification failed"
    log "SLSA provenance verified"
  else
    log "no provenance asset published for ${asset} — skipping SLSA check"
  fi
fi

# --- install ---------------------------------------------------------------
install -m 0755 "${tmp}/clog" "${CLOG_INSTALL_DIR}/clog"
log "installed: $("${CLOG_INSTALL_DIR}/clog" --version 2>/dev/null || echo "${CLOG_INSTALL_DIR}/clog")"

# emit the install dir so callers can add it to PATH / $GITHUB_PATH
printf '%s\n' "$CLOG_INSTALL_DIR"
