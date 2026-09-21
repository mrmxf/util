# Proposal — per-target scan and lint flags

2026-09-21. **Proposal only, nothing implemented.** Written after `build-hugo`
failed on pihuw: the Trivy step assumes a container, and a Hugo site published
to GitHub Pages has none.

## Two faults in the step as it stands

```yaml
- name: scan image
  if: env.do_deploy == 'true'
  uses: aquasecurity/trivy-action@…
  with:
    image-ref: ${{ env.repo }}
```

**It scans the wrong thing.** `env.repo` comes from `clog ci resolve` and is the
GitHub slug — `mrmxf/pihuw`. Trivy is being asked to pull a container image of
that name. For a Pages site none exists; for a container build it is right only
by coincidence, when the image happens to be named after the repo.

**It scans at the wrong time.** The gate is `do_deploy`, so a build that is not
deploying — every pull request, every branch push — is never scanned at all.
Security scanning belongs on `do_build`: the point is to find the problem before
it is a release, and a PR is exactly when that is cheap. Deployment is a
separate decision.

## The shape of the fix

A target already says where a build goes. It should also say **what kind of
sweep the deployable asset needs**, because that follows from what the asset
*is*: an image gets an image scan, a directory of files gets a filesystem scan.

Three layers, most specific winning, so the common case is silent:

```
kind default            built into clog - most repos write nothing
ci.scan:                repo-wide override
ci.targets.<n>.scan:    per-target override
```

### Kind defaults

| kind | security sweep | ref |
|---|---|---|
| `container-registry` | `image` | the image the target pushes |
| `github-pages` | `fs` | the target's `dir` |
| `gitlab-pages` | `fs` | the target's `dir` |
| `cloudflare-pages` | `fs` | the target's `dir` |
| `bucket` | `fs` | the target's `dir` |
| `package` | `fs` | the target's `dir` |

A repo with one Pages target gets a filesystem scan of its publish directory and
writes not one line of config. pihuw is that repo.

### Config

```yaml
ci:
  scan:                          # repo-wide defaults, all optional
    security: fs                 # image | fs | repo | config | none
    severity: HIGH,CRITICAL
    exit-code: 1
    skip-dirs: node_modules,vendor

  targets:
    pages:
      kind: github-pages
      prod: {dir: kodata, branch: gh-pages}
      # nothing to say: kind gives security=fs, ref=kodata

    registry:
      kind: container-registry
      prod: {image: mrmxf/www, tags: ["{tag}", "latest"]}
      scan: {security: image}    # explicit, though the kind implies it
```

`security: none` is how a target opts out, and it is deliberately a thing you
have to write down rather than a thing that happens when config is missing.

### Say the sweep, not the tool

`security: fs` describes **what is being examined**, not that Trivy does it. The
workflow maps sweep to tool. Swapping Trivy, or adding a second scanner, is then
a workflow change and not a config migration across every repo. The same holds
for lint below: `megalinter` is a tool choice, `lint: true` is the intent.

## Surface

One command, following `ci policy --format env`:

```
$ clog ci scan --format env
scan_security=fs
scan_ref=kodata
scan_severity=HIGH,CRITICAL
scan_exit_code=1
scan_skip_dirs=
```

`clog-prepare` appends it to `$GITHUB_ENV` next to resolve and policy, so the
workflow stays declarative:

```yaml
- name: security scan (${{ env.scan_security }} ${{ env.scan_ref }})
  if: env.do_build == 'true' && env.scan_security != 'none'
  uses: aquasecurity/trivy-action@…
  with:
    scan-type: ${{ env.scan_security }}
    scan-ref:  ${{ env.scan_ref }}
    severity:  ${{ env.scan_severity }}
    exit-code: ${{ env.scan_exit_code }}
    skip-dirs: ${{ env.scan_skip_dirs }}
```

`scan-type` and `scan-ref` are the current trivy-action inputs; `image-ref` is
documented there as "for backward compatibility" and is not needed.

### More than one target

A repo can deploy to several kinds at once — www-mrmxf-com has both a registry
and a Pages target — so the scan is per-target, exactly like deploy. `ci scan`
takes `--target` and defaults to `$CLOG_TARGET`, and the build job loops
`clog ci targets` the way the deploy job already does. One target, one
iteration, no visible difference.

The scan runs **after** `clog build`: the image and the publish directory only
exist once the build has produced them.

## Lint — same shape, later

```yaml
ci:
  lint:
    enabled: true
    tool: megalinter         # the workflow maps this to an action
    config: .mega-linter.yml
    exit-code: 1
```

```
$ clog ci lint --format env
lint_enabled=true
lint_tool=megalinter
lint_config=.mega-linter.yml
lint_exit_code=1
```

Lint is repo-wide rather than per-target: it examines the **source**, which does
not vary by where the build is published. That asymmetry with `scan` is the
point — the flag describes the subject, and source has one subject while
deployables have several.

## What it costs

- `ci/scan.go` — `Scan` struct, kind defaults, resolution, `clog ci scan`
- `ci/targets.go` — a `Scan` field on `Target`, plus `ci.scan` on `Config`
- `clog-prepare` — append `ci scan --format env` to `$GITHUB_ENV`
- `build-hugo.yaml`, `build-golang.yaml` — replace the hardcoded step
- tests — kind defaults, the override ladder, `none`, a missing `dir`

`build-golang.yaml` has no scan step at all today, so this adds coverage there
rather than fixing it.

## Decisions wanted

1. **`exit-code: 1` by default?** A failing scan then blocks a release. That is
   the right default, but it will stop builds that pass today — including, very
   likely, the first pihuw run.
2. **Should `fs` scan the publish directory or the whole worktree?** The
   directory is what ships; the worktree catches a vulnerable dependency that
   never reaches the output. My inclination is the directory for `security`, and
   `repo` as the opt-in for the wider sweep.
3. **Is `package` really `fs`?** It depends whether the target pushes a built
   tarball or a source tree, and no repo here uses it yet.
