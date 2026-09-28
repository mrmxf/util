# mrmxf/util reusable workflows

Generic, callable (`workflow_call`) CI workflows shared across mrmxf Go/Hugo
repos. Phase 9½ moved them here from the (now deprecated) `mrmxf/clog` repo and
rewrote each as a thin shim over `clog` — **thin YAML, fat clog**.

| workflow | purpose |
| --- | --- |
| [`../actions/clog-prepare`](../actions/clog-prepare/action.yaml) | shared first step: checkout, Go, clog, ci.policy |
| [`build-golang.yaml`](./build-golang.yaml) | multiplatform Go build via `clog build` |
| [`deploy-s3.yaml`](./deploy-s3.yaml) | deploy an artifact to S3 via `clog deploy` |
| [`dump-context.yaml`](./dump-context.yaml) | debug: dump GitHub/env/job contexts (no secrets) |
| [`test-setup-clog.yaml`](./test-setup-clog.yaml) | smoke-test the clog bootstrap |

## Shared shape (v1: secrets from Infisical)

Every build/deploy job is the same few moves; the decisions live in the caller's
`.clog.yaml`, the secrets in Infisical, the logic in `clog`:

1. **prepare** — [`clog-prepare`](../actions/clog-prepare/action.yaml): checkout,
   Go from `go.mod`, clog (built from the repo with `self-build: true`, else a
   pinned, checksum-verified release via [`setup-clog`](../actions/setup-clog)),
   then `clog ci resolve` + `clog ci policy` + `ci.*` values into `$GITHUB_ENV`.
2. **gate** — steps run only when `env.do_build` / `env.do_deploy` is `true`
   (`ci.policy`: events, branch/tag globs, `releases-yaml`, `actors`). A run is
   `dev` or `prod` — there is no staging — and the mode picks the Infisical
   environment, identity and per-target data.
3. **work** — `clog ci run -- bash -c 'clog ci require <verb> && clog <verb>'`:
   OIDC login to Infisical, secrets in the command's environment only, masked.
   Deploy runs once per `ci.targets` entry, each with `$CLOG_TARGET` set.
4. **report** — `clog SlackStash` under `clog ci run` (`HOOK_SLACK` from Infisical).

No workflow takes secrets; each carries least-privilege `permissions:`
(`contents: read`, `id-token: write`), a `timeout-minutes:`, a `concurrency:`
group and SHA-pinned third-party actions.

## Consuming

```yaml
permissions: {contents: read, id-token: write}
jobs:
  build:
    uses: mrmxf/util/.github/workflows/build-golang.yaml@workflows-v1
    permissions: {contents: read, id-token: write}
    with: {self-build: true}          # only for repos that ARE clog
  deploy:
    needs: [build]
    uses: mrmxf/util/.github/workflows/deploy-s3.yaml@workflows-v1
    permissions: {contents: read, id-token: write}
    with: {self-build: true}
```

The caller's `.clog.yaml` supplies everything else:

| key | used for |
| --- | --- |
| `ci.policy` | build / deploy decision (`clog ci policy --help`) |
| `ci.artifact` | artifact uploaded by build, downloaded by deploy |
| `ci.title` | Slack title |
| `ci.modes.<dev\|prod>` | per-mode build settings (`clog ci mode get base-url`) |
| `ci.targets.<name>` | deploy destinations: `kind`, `require`, per-mode data |
| `ci.require.<verb>` | secrets/config that must exist before the verb runs |
| `ci.infisical.*` | project, env map, identity UUIDs — `clog ci --config-help` |

GitLab: [`gitlab/clog.gitlab-ci.yml`](../../gitlab/clog.gitlab-ci.yml) provides the
same `clog-build` / `clog-deploy` jobs for `include: remote`.

Pin to the moving major tag `@workflows-v1`, or to an immutable
`@workflows-v1.0.0` (or a commit SHA) for reproducibility.

## Coupled dependencies

1. **clog version** — the clog that runs must contain `clog ci run/policy/get`
   (util/ci v0.11.3+). `self-build: true` gets it from the repo's `go.mod`;
   releases need a clog-sample release built with it.
2. **release publishing** — `setup-clog` downloads a GitHub Release and verifies
   `checksums.txt`. `mrmxf/clog-sample` has no releases yet, so today only
   `self-build: true` callers work.
3. **Infisical** — project, identities with OIDC auth, and the `.clog.yaml`
   `ci.infisical` block (`clog ci --config-help`).
