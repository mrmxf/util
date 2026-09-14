# mrmxf/util reusable workflows

Generic, callable (`workflow_call`) CI workflows shared across mrmxf Go/Hugo
repos. Phase 9½ moved them here from the (now deprecated) `mrmxf/clog` repo and
rewrote each as a thin shim over `clog` — **thin YAML, fat clog**.

| workflow | purpose |
| --- | --- |
| [`build-golang.yaml`](./build-golang.yaml) | multiplatform Go build via `clog build` |
| [`build-hugo.yaml`](./build-hugo.yaml) | Hugo site → container via hugo + ko |
| [`deploy-s3.yaml`](./deploy-s3.yaml) | deploy an artifact to S3 via `clog deploy` |
| [`dump-context.yaml`](./dump-context.yaml) | debug: dump GitHub/env/job contexts (no secrets) |
| [`test-setup-clog.yaml`](./test-setup-clog.yaml) | smoke-test the clog bootstrap |

### Legacy `-v0` workflows

Verbatim clones of the old `mrmxf/clog/.github/workflows/*` so existing callers
can leave the deprecated repo by changing only the `uses:` path — inputs and
secrets are unchanged (`get_clog`, `bcMd`, `cf_*`):

```yaml
# before
uses: mrmxf/clog/.github/workflows/build-hugo.yaml@main
# after
uses: mrmxf/util/.github/workflows/build-hugo-v0.yaml@main
```

| workflow | replaces | caveats |
| --- | --- | --- |
| [`build-golang-v0.yaml`](./build-golang-v0.yaml) | `mrmxf/clog` `build-golang.yaml` | `eval`s the `get_clog` secret (audit F1); runs `clog install golang/slsa-verifier/ko` snippets |
| [`build-hugo-v0.yaml`](./build-hugo-v0.yaml) | `mrmxf/clog` `build-hugo.yaml` | as above, plus `clog install hugo`; `trivy-action@master` and a third-party Cloudflare bypass action |
| [`deploy-s3-v0.yaml`](./deploy-s3-v0.yaml) | `mrmxf/clog` `deploy-s3.yaml` | `eval`s the `get_clog` secret |
| [`dump-context-v0.yaml`](./dump-context-v0.yaml) | `mrmxf/clog` `dump-context.yaml` | none — no clog, no secrets |
| [`test-get-clog-v0.yaml`](./test-get-clog-v0.yaml) | `mrmxf/clog` `test-get-clog.yaml` | `eval`s the `get_clog` secret |

The `clog install <tool>` snippets only exist in clog releases before the
built-in `clog Install`; point `GET_CLOG` at such a release, or migrate to the
non-`-v0` workflow. The `-v0` files are frozen: fix forward in the rewrites.

## Shared shape

Every build/deploy job is the same four moves; the logic lives in `clog`, not
the YAML:

1. **bootstrap** — [`setup-clog`](../actions/setup-clog) installs a pinned,
   checksum-verified clog (no `eval`-of-secret, audit F1).
2. **normalize** — `clog ci resolve --format env >> $GITHUB_ENV` turns the
   event into `verb/ref/repo/url/depth/is_production` (testable Go, replaces
   ~140 lines of inline bash).
3. **work** — `clog build` / `clog deploy` (optional `MAKE` override).
4. **report** — `clog SlackStash` posts the BC stash to Slack.

Each job also carries least-privilege `permissions:`, a `timeout-minutes:`, a
`concurrency:` group, and SHA-pinned `uses:` (F4/F5/F8).

## Consuming

```yaml
jobs:
  build:
    uses: mrmxf/util/.github/workflows/build-golang.yaml@workflows-v1
    with:
      artifact-name: clog-bin
    secrets:
      webhook_slack: ${{ secrets.WEBHOOK_SLACK }}
```

Pin to the moving major tag `@workflows-v1` for convenience, or to an immutable
`@workflows-v1.0.0` (or a commit SHA) for reproducibility.

## Coupled dependencies (must land before these run green)

1. **clog version** — `.clog-version` must point at a clog release that
   contains `clog ci resolve` and `clog BC …` (Phase 9½ Workstream A). Releases
   before that (e.g. v0.10.11) will fail at the *resolve* step. Bump the pin to
   the first release that ships `clog ci`.
2. **release publishing** — `setup-clog` downloads from a GitHub Release and
   verifies a checksum. The release pipeline must publish `clog-<cpu>-<os>` +
   `checksums.txt` (+ optional SLSA provenance) per tag. See
   [`../actions/setup-clog/README.md`](../actions/setup-clog/README.md).
3. **tags** — create `workflows-v1` (moving) and `workflows-v1.0.0` (immutable)
   on this repo so callers can reference them. (Release step — not auto-created.)
