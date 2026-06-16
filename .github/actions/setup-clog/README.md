# setup-clog

Install a **pinned, checksum-verified** `clog` binary and add it to `PATH`.

This is the Phase 9½ replacement for the legacy `eval "$(secrets.get_clog)"`
bootstrap (audit finding **F1** — executing secret *contents* is an
RCE-in-CI risk that bypasses code review). Here the clog version is an explicit
input, the binary comes from a GitHub Release, and its `sha256` is verified
before it is ever executed.

## Usage

```yaml
- uses: actions/checkout@<sha>          # needed so .clog-version is on disk
- uses: mrmxf/util/.github/actions/setup-clog@workflows-v1
  with:
    clog-ref: ""                         # default: read ./.clog-version
    clog-repo: mrmxf/clog-sample         # public binary; use mrmxf/clog-app for private
    token: ${{ secrets.clog_download_token }}   # only for a private clog-repo
- run: clog --version
```

The same logic runs on a laptop and under GitLab via the underlying
[`get-clog.sh`](./get-clog.sh) (`source ./get-clog.sh`), so dev and every CI
install **byte-identical clog** — the core dev↔CI drift lever.

## Version pinning

The installed version is, in order of precedence:

1. the `clog-ref` input, else
2. the first non-comment line of `.clog-version` in the consumer repo.

`.clog-version` is the single source of truth for the clog orchestrator
version and is consumed by both the dev install and this action. Pinning here —
not a floating tag, not a secret — is what keeps dev and CI aligned.

## Publishing requirement (coupled deliverable)

This action downloads from a GitHub Release and **refuses to install an
unverified binary**. For it to succeed, the clog release pipeline
(`clog deploy`) must publish, per release tag `vX.Y.Z`, to
`github.com/<clog-repo>/releases/download/vX.Y.Z/`:

| asset | required | purpose |
| --- | --- | --- |
| `clog-amd-lnx`, `clog-arm-lnx`, `clog-amd-mac`, `clog-arm-mac` | yes | the platform binaries (existing naming convention) |
| `checksums.txt` | yes | `sha256sum` lines; verified before the binary runs |
| `clog-<cpu>-<os>.intoto.jsonl` | optional | SLSA provenance; verified if `slsa-verifier` is on `PATH` |

Today the binaries are served unversioned and unchecksummed from the
`mrmxf.com/get/...` CDN. Cutting over to checksummed, tag-versioned GitHub
Releases is the publishing-side half of Workstream B and a prerequisite for
this action to run green. Until then, the bootstrap path is in place and
reviewable but will report a clear `download failed` / `could not fetch
checksums.txt` error.
