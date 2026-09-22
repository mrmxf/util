# Proposal — scan and lint inside the build flow

2026-09-22. **Proposal only, nothing implemented.** Supersedes the 2026-09-21
draft, which proposed a per-target `scan` flag driving a `trivy-action` step.
That draft diagnosed the fault correctly and then fixed it in the wrong place.
This one is written against the four repo shapes the flow actually serves:

- Hugo sites published to Cloudflare Pages, GitHub Pages, or a container
- Go apps **and libraries**
- container management and infrastructure monitoring apps
- embedded firmware built with TinyGo for esp32

Eleven questions are now settled and written in below; the **Settled** table near
the end lists them together. The load-bearing one is that `pre-build` stays in the
default check list, because **`clog build` means ready to push** and `clog build
--fast` makes no claim at all. That promise turns out to be the most demanding
part of this document: six things in the flow today quietly break it, and they
get their own section.

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

Both still hold. What follows changes the fix, not the diagnosis.

## Why the first fix was the wrong shape

The first draft derived the sweep from the deploy target: an image gets an image
scan, a directory gets a filesystem scan. That binds scanning to the
**deployable**, which assumes the deployable carries the risk. Across the four
shapes above, it carries the risk exactly once.

- **A Go library has no target at all.** No image, no publish directory, often
  no `ci.targets` block. Under the first draft it is never scanned, and the
  omission is silent because no config is missing. A library's module graph is
  the one thing worth scanning.
- **A Hugo publish directory is not scannable.** Trivy finds vulnerabilities
  through lockfiles and compiled binaries. A rendered site has neither — it is
  HTML, CSS, images and bundled JS. `security: fs` on `kodata` is a permanently
  green step that reads like coverage. The real surface is the *source*: the npm
  asset pipeline, Hugo modules, theme submodules.
- **Firmware is the same story with one real win.** A vulnerability scan of a
  `.bin` or `.uf2` finds nothing. Trivy's **secret** scanner does work on a
  firmware blob, and baked-in WiFi credentials or API tokens are a genuine esp32
  failure mode. That is a different axis, not a severity level.
- **Only the container work** has deployable and risk in the same place.

The first draft already contained the argument against itself, in the lint
section: *the flag describes the subject, and source has one subject while
deployables have several.* Dependency scanning is a source subject. It belongs
repo-wide, running whether or not a target exists.

## The shape: four verbs

The config does the thinking so a developer types four things and never thinks
about scanners, linters, modes or targets:

```
clog watch                # interactive loop: hugo server, go run, tinygo monitor
clog build dev --fast     # skip every gate — a long way still to go
clog build dev|prod       # every gate passed — ready to push
clog deploy dev|prod      # publish what was built
```

Each verb takes an optional stack name, so a repo with more than one still reads
the same way:

```
clog build                # every stack — the rigorous default
clog build bonfire        # just the stack named bonfire
clog watch bonfire        # the inner loop is always one stack
```

**The machinery already exists.** `clog BC flow` ([bc/sub-flow.go](../bc/sub-flow.go))
runs a `CHK` phase list, then a `MAKE` phase list, accumulating errors. `clog Check`
([check/command.go](../check/command.go)) already runs a named group of
`try`/`ok`/`catch` blocks, and `check.pre-build` and `check.golang` already exist
in [embedfs/konfig.yaml](../embedfs/konfig.yaml).

So scanning and linting are **check phases**, not bespoke workflow steps:

- `clog build <mode>` resolves the stack's `chk` and `make` lists and calls flow
- `--fast` runs with an empty `chk` list — nothing else changes
- `clog watch` runs no checks by definition; it is the inner loop
- CI calls the same verbs, so CI and a laptop cannot drift

This also disposes of a problem the first draft could not solve. A `uses:` step
cannot iterate targets, but the draft's "more than one target" section required a
loop. The deploy path already solved that with bash, at
[gitlab/clog.gitlab-ci.yml:85](../gitlab/clog.gitlab-ci.yml#L85). Moving the sweep
inside `clog` makes the loop bash rather than YAML, and the workflow stops caring.

### `ci.stack`

One new repo-level key carries the upfront work:

```yaml
ci:
  stack: hugo        # hugo | golang | golang-lib | container | tinygo
```

That is shorthand. The long form is a list of **named** stacks, and the name is
what the verbs take:

```yaml
ci:
  stack:
    - name: bonfire       # clog build bonfire
      type: hugo
    - name: form-parking
      type: golang
```

A bare string is a one-element list whose name is its type, so `stack: hugo` and
`stack: [{name: hugo, type: hugo}]` mean the same thing.

The type sets the default toolchain, watch command, check list and make list, so
the common repo writes one line and nothing else:

| stack | `tools` | `watch` | `chk` | `make` |
|---|---|---|---|---|
| `hugo` | hugo, ko, trivy | `hugo server -D` | `pre-build lint scan` | `hugo ko` |
| `golang` | golang, trivy | `go run .` | `pre-build lint test scan` | `golang` |
| `golang-lib` | golang, trivy | `gotestsum --watch` | `pre-build lint test scan` | `golang` |
| `container` | golang, ko, trivy | `docker compose watch` | `pre-build lint test scan` | `golang ko` |
| `tinygo` | tinygo, trivy | `tinygo flash && tinygo monitor` | `pre-build lint scan` | `tinygo` |

Every column is overridable per repo, and `ci.modes.<mode>` may override again:

```yaml
ci:
  stack: hugo
  modes:
    prod:
      chk: [pre-build, lint, test, scan]   # prod adds the slow one
```

**The callable workflows collapse into one.** `build-hugo.yaml` and
`build-golang.yaml` differ only in which tools they install and which build
snippets they run, and `ci.stack` now knows both. They become a single
`build.yaml` whose body is:

```yaml
- uses: mrmxf/util/.github/actions/clog-prepare@workflows-v1
- run: clog Install $(clog ci stack get tools)     # stack decides
- run: clog ci run -- clog build ${{ env.build_mode }}
```

A repo stops choosing a workflow by name and declares its stack instead. That is
one fewer thing to know, and it removes the class of bug where the workflow and
the config disagree about what kind of repo this is.

### A repo may declare more than one stack

A repo holds more than one stack whenever it ships more than one kind of thing
from one tree. bonfire is the worked example: a main site alongside two small
deployables that are built and released independently of it.

```yaml
ci:
  stack:
    - name: bonfire         # the site — first, so it is what watch follows
      type: hugo
    - name: form-contact
      type: golang
    - name: form-parking
      type: golang
```

Embedded work has the same shape for a different reason: esp32 firmware,
cross-compiled and flashed, beside the host-side companion that flashes it and
reads its telemetry. Different toolchains, different make steps, and linter
settings that must not see each other.

### Which stack a verb acts on

**`clog build` builds every stack.** Narrowing to one costs you a word, exactly
as turning the gates off costs you `--fast`. The rigorous thing is what you get
for free.

| command | acts on |
|---|---|
| `clog build` | **every** stack, in declaration order |
| `clog build all` | the same thing, said out loud |
| `clog build bonfire` | only the stack named `bonfire` |
| `clog deploy` | every stack's targets |
| `clog watch` | the **first** stack — see below |

It echoes what it resolved, by name rather than as the word `all`:

```
$ clog build
building bonfire, form-contact, form-parking (prod)
```

Naming them is more use than echoing `all`, because it is also the cheapest check
that the config says what somebody thought it said. A single-stack repo sees
`building hugo` and is none the wiser that any of this exists.

**The real argument is not symmetry with `--fast`, it is what happens when a repo
gains a stack.** If `clog build` meant the first stack, then adding a second entry
to `ci.stack` would silently halve what CI builds, and nothing would say so. The
config would grow and the coverage would shrink. Defaulting to everything means
adding a stack extends coverage automatically, and the collapsed workflow above
needs no stack argument and stays correct for every repo forever. A default that
loses coverage when config grows is the same family of fault as a scan that
passes because it was never configured, which is what this document exists to
remove.

The cost lands on the multi-stack inner loop, where `clog build` now does twice
the work. That is the right place for it to land: `clog build` is the
ready-to-push gate, run rarely and deliberately, while `clog watch` and
`clog build --fast` own the fast path and are untouched.

**`watch` is the exception and defaults to the first stack, out loud.** A watch
command is an interactive foreground process, so `clog watch all` is refused
rather than defaulted to, and declaration order earns its keep: the first stack
is the one you are usually looking at.

The narrowing is implicit, so it is stated:

```
$ clog watch
watching bonfire, ignoring [form-contact, form-parking]
```

That is the resolution of the tension the rest of this document keeps circling.
Narrowing is fine; **narrowing in silence** is the fault. `clog watch` does not
need to refuse in order to be honest, it needs to say what it left out. Anyone
who expected the parking form to reload now knows within one line why it did not,
instead of debugging a form that was never being watched.

The same line appears wherever a verb acts on fewer stacks than the repo has, so
`clog build bonfire` prints `building bonfire (dev), ignoring [form-contact,
form-parking]` for exactly the same reason. Only the unnarrowed default stays
quiet about omissions, because it omits nothing.

Two rules rather than one, but each falls out of what the verb does — build and
deploy are batch verbs and take everything, watch is a foreground verb and takes
one and says so.

**`all` is a reserved name.** A stack named `all` makes `clog build all`
ambiguous, so the config parser **errors and exits** naming the offending stack.
The same applies to `dev` and `prod`, for the same reason: the verbs disambiguate
their arguments by value rather than by position, which is what lets
`clog build bonfire` and `clog build prod bonfire` both work without a flag.
Three reserved words, rejected at parse time rather than at the moment they
confuse somebody.

### What a multi-stack repo shares

- `tools`, `chk` and `make` are unioned across the selected stacks, in
  declaration order and deduplicated, so `pre-build` and `scan` run once rather
  than twice however many stacks are in play
- `make` phases stay namespaced per stack, so the firmware image and the host
  binary do not collide in `tmp/`
- `ci.lint` resolves **per stack**, because the TinyGo half needs build tags the
  host half must not see
- `ci.scan.source` stays repo-wide, because the worktree is still one worktree

**That lint split walks back part of the first draft's argument.** It justified
repo-wide lint by saying source has one subject. A two-stack repo has two. The
underlying rule survives in a weaker form: lint follows the source, and a repo has
as many source subjects as it has stacks.

### Targets have to name their stack

This is the one place the selector leaks. `ci.targets` is a flat repo-level list,
so `clog deploy bonfire` has no way to know which targets are the firmware's, and
`clog build bonfire` followed by a full deploy would publish an artifact this run
never produced. Targets gain an optional `stack:`, and a target that omits it
belongs to the first stack:

```yaml
targets:
  site:
    kind: github-pages
    stack: bonfire
  parking:
    kind: container-registry
    stack: form-parking
```

`clog deploy <selector>` then publishes only the targets belonging to the
selected stacks, and the artifact scan sweeps only those. A single-stack repo
writes no `stack:` on any target and never notices the field exists.

### Toolchain install refuses, it does not install

Because `clog build` means ready to push, a missing gate tool is a failure, not a
shrug. But `clog build` does **not** install it either. A verb that silently
pulls binaries onto the machine while claiming to check a branch is the wrong
supply-chain posture, and it hides what changed:

```
$ clog build dev
✗ golangci-lint is not installed and check phase `lint` needs it
  clog Install golangci-lint
```

Refuse, name the tool, print the exact line, exit non-zero. Installing stays an
explicit act the developer performs and can see in their shell history. CI does
the same thing explicitly, which is why the collapsed workflow above keeps its
own install step driven by `clog ci stack get tools`.

## Scanning: two axes, not one

Vulnerabilities and secrets are independent questions about the same subject, so
they are two settings rather than two values of one setting:

```yaml
ci:
  scan:
    source:                          # repo-wide, runs with or without targets
      vuln: repo                     # repo | fs | none
      secrets: true
      severity: HIGH,CRITICAL
      ignore-unfixed: true
      ignorefile: .trivyignore
      skip-dirs: node_modules,vendor,public,tmp
    artifact:                        # per-target defaults; the kind fills these in
      severity: CRITICAL
      ignore-unfixed: true
```

**`source`** sweeps the worktree for vulnerable dependencies and committed
secrets. It runs on every `clog build`, in every repo, including the ones with no
targets. This is the axis that covers Go libraries, the Hugo asset pipeline and
the TinyGo module graph — three of the four shapes.

**`artifact`** sweeps the thing a target ships, once per target, after `make`.

### Kind defaults

`package` is removed from `knownKinds`. `github-release` and `gitlab-release` are
added; `gh` and `glab` already have install recipes, so the publishers exist.

| kind | `vuln` | `secrets` | why |
|---|---|---|---|
| `container-registry` | `image` | `true` | layers and embedded Go module data are real findings |
| `github-release` | `fs` | `true` | Trivy reads Go binaries for module data, so release assets are genuinely scannable |
| `gitlab-release` | `fs` | `true` | as above |
| `github-pages` | **required** | `true` | rendered HTML has no lockfiles; there is no honest default |
| `gitlab-pages` | **required** | `true` | as above |
| `cloudflare-pages` | **required** | `true` | as above |
| `bucket` | **required** | `true` | contents are unknowable from the kind alone |

Two things changed from the first draft. Pages and bucket kinds no longer default
to a filesystem sweep, because a filesystem vulnerability scan of a publish
directory finds nothing and reports it in green. And they do not silently default
to `none` either:

```yaml
targets:
  pages:
    kind: github-pages
    prod: {dir: kodata, branch: gh-pages}
    scan: {vuln: none}          # required — a rendered site has no dependencies
```

**A missing `vuln` on these kinds is an error**, naming the target and saying what
to write. One line, written once, that records a decision somebody made. The
first draft's "pihuw writes not one line of config" is given up deliberately:
silence that means *not scanned* is the thing this whole document exists to
remove.

Secret scanning has an honest default everywhere, so it is never required.

### Publishing a release

Both release kinds shell out to the forge's own CLI rather than to a generic
uploader. `gh` and `glab` already have install recipes, both handle auth, tag
creation and multipart upload, and neither needs a new abstraction to justify
itself. A third forge can have a generic path if one ever turns up.

```yaml
targets:
  firmware:
    kind: github-release
    require: [GH_TOKEN]
    prod: {dir: tmp/firmware, notes: releases.yaml}
```

```bash
# deploy-github-release, in outline
gh release view "$tag" >/dev/null 2>&1 \
  || gh release create "$tag" --notes-file "$notes"
gh release upload "$tag" "$dir"/* --clobber
```

`--clobber` matters: a deploy is re-run often enough that failing on an existing
asset would be the common case rather than the exception. The `glab` path is the
same shape with `glab release create` and `glab release upload`.

The expectation is that this is where problems appear first, because release
upload is the one deploy path with real statefulness on the far side: partial
uploads, a tag that exists with no release attached, assets from a previous run.
Start with the CLIs, and fix what breaks.

### Say the sweep, not the tool

`vuln: image` describes **what is examined**, not that Trivy does it. Swapping
Trivy or adding a second scanner is then a change in one check block, not a
config migration across every repo.

### Two corrections the resolver must get right

**Enumerate every target, not the deploying ones.** `clog ci targets` filters by
deploy-mode membership ([ci/targets.go:48](targets.go#L48)); a target with only a
`prod:` block reports no dev membership. Moving the gate to `do_build` and then
enumerating with `ci targets` would leave prod-only targets unscanned on every
pull request — the original bug, through a different door. The scan enumerates
all targets and resolves each against the run's mode.

**`dir` and `image` are per-mode.** They live inside the `dev:`/`prod:` maps
([ci/targets.go:35](targets.go#L35)), not on `Target`. Resolution is mode-aware
like `TargetGet`, and a dev build asking for a prod-only `dir` fails with a
message naming the target and the mode.

### `ignore-unfixed` is not optional

For the container management and monitoring repos, base images accumulate HIGH
and CRITICAL findings with no fix available. A blocking gate without
`ignore-unfixed: true` and an ignore file produces chronic red, and chronic red
gets scanning switched off. Both are defaults above rather than settings someone
has to discover.

## What else breaks the ready-to-push contract

Decision 4 fixes the meaning of the verb: **`clog build` passing means the branch
is fit to push; `clog build --fast` makes no claim at all.** That is a strong
promise, and six things in the flow today quietly break it. They matter more than
the scanning work, because a gate that cannot fail is worse than no gate.

**1. `clog build` pushes to a public registry.** `bc-ko`
([embedfs/konfig.yaml:447](../embedfs/konfig.yaml#L447)) runs `ko build` with
`KO_DOCKER_REPO` set, which publishes. A verb that means "am I ready" must not
mutate the outside world, and a pull request build currently either publishes an
image or fails at a login step it never ran. The rule becomes absolute: **build
never pushes, deploy always does.**

```bash
ko build --base-import-paths --sbom=spdx --push=false \
         --tarball=tmp/image.tar --tags "$tag1" .
```

The artifact sweep reads `tmp/image.tar`, and `clog deploy` pushes the same
image. The `--sbom=none` on that line also throws away the SPDX document that is
the cheapest accurate input for a distroless scan, so it goes.

**2. The staticcheck gate cannot fail.**
[embedfs/konfig.yaml:107](../embedfs/konfig.yaml#L107):

```bash
staticcheck ./... | exit 1
```

The exit status of a pipeline is its last command, so this is always 1 whatever
staticcheck found. The `catch` therefore always fires — and it logs with `-S`,
the success flag, and never exits non-zero. A section fails *only* when a catch
script exits non-zero ([check/run.go:162](../check/run.go#L162)), so the block
reports success unconditionally. Today `clog Check golang` runs staticcheck,
discards the result, and says it passed. It should be `staticcheck ./... ||
exit 1`, with the catch logging `-E` and exiting 1.

**3. Missing linters are skipped silently.** Both Go blocks exit 0 when the tool
is absent ([konfig.yaml:108](../embedfs/konfig.yaml#L108) and
[konfig.yaml:117](../embedfs/konfig.yaml#L117)). On a fresh laptop, `clog build`
lints nothing and declares the branch ready. Per decision 4 the fix is to refuse
rather than to auto-install: the block fails, names the tool and prints
`clog Install golangci-lint`. A gate that cannot run is not a gate that passed.

**4. `clog build dev` currently warns and continues.** Flow aborts on check
failure only in prod. With `--fast` as the explicit escape hatch, that
distinction now costs more than it earns: if `build dev` succeeds with failed
checks, the only difference between `build dev` and `build dev --fast` is how
much scrolling you do. **A check failure fails the build in both modes.** The
mode governs what is checked and how strictly, never whether failure counts.

This lands as one change, with no `ci.strict` flag to soften it — doing it once
and doing it right. Be ready for what that means on the first run after landing.
Three of these six fixes remove a way for a check to pass without running:
strict dev mode, the staticcheck pipeline, and refusing on a missing linter.
Every repo will surface findings that were always there and never reported. That
backlog is the honest measure of what the old gates were worth, and `--fast`
exists so it never blocks someone mid-bug.

**5. `pre-build` is the gate CI never runs.** The tree-clean, tree-ahead and
unstaged checks all exit 0 under `$CI`
([konfig.yaml:78](../embedfs/konfig.yaml#L78)). That is correct — a runner always
has a clean checkout — but it means the local and CI meanings of `clog build`
are not identical, which undercuts the "cannot drift" claim above. Worth stating
plainly in the help text rather than pretending otherwise: `pre-build` is a
laptop gate, everything else runs in both places.

**6. Build must not need deploy secrets.** If `clog build` requires an Infisical
fetch to run, then the verb only works where OIDC works, and the laptop case
degrades to `--fast`. Deploy credentials belong to `clog deploy`. The scan phase
in particular needs no secrets, only a vulnerability database, which should be
cached and whose fetch failure is a check error with a clear message rather than
a hang.

One more, inherent rather than fixable: `clog watch` runs `hugo server`, which
renders drafts, skips minification and uses a different base URL. Watch passing
says nothing about build passing. That is fine as long as nobody reads it as a
gate, so `watch` should say so on startup.

## Surface

The primary path is the check phase — nobody types this. `clog ci scan` stays as
a thin query for the workflow and for debugging:

```
$ clog ci scan --format env                 # source axis
scan_source_vuln=repo
scan_source_secrets=true
scan_source_ref=.
scan_source_severity=HIGH,CRITICAL
scan_source_ignore_unfixed=true

$ clog ci scan --format env --target registry
scan_artifact_vuln=image
scan_artifact_secrets=true
scan_artifact_ref=tmp/image.tar
scan_artifact_severity=CRITICAL
```

Results are written as SARIF to `tmp/scan/` so the workflow can upload them once,
outside any loop:

```yaml
permissions:
  contents: read
  id-token: write
  security-events: write        # ← new, required for the upload below

- name: publish scan results
  if: always() && env.do_build == 'true' && hashFiles('tmp/scan/*.sarif') != ''
  uses: github/codeql-action/upload-sarif@…
  with: {sarif_file: tmp/scan}
```

Trivy needs an install recipe. There is none in
[install/recipes/](../install/recipes/) today, so it joins the stack `tools`
lists, pinned the way `hugo`, `ko` and `slsa-verifier` already are. That also
avoids `trivy-action` pulling its vulnerability database through a rate limited
registry on every run.

## Lint

Lint is a check phase like scan, and for the same reason the first draft gave:
it examines the source, which has one subject.

One `tool:` string is too narrow for this fleet, so it takes a list:

```yaml
ci:
  lint:
    tools: [golangci-lint]
    config: .golangci.yml
```

| stack | default tools | note |
|---|---|---|
| `hugo` | `markdownlint`, `lychee` | MegaLinter is reasonable here if one container is preferred |
| `golang`, `golang-lib`, `container` | `golangci-lint` | already an install recipe, already a `check.golang` block |
| `tinygo` | `golangci-lint` with build tags | see below |

Routing Go repos through MegaLinter would wrap `golangci-lint` in a
multi-gigabyte container to no benefit.

**TinyGo is the sharp edge.** Files guarded by `//go:build tinygo` import
`machine` and friends, which do not compile for the host, so any typecheck-based
linter reports errors that are not errors. The `tinygo` stack default sets the
build tags and target architecture explicitly:

```yaml
ci:
  lint:
    tools: [golangci-lint]
    env: {GOOS: linux, GOARCH: arm}
    build-tags: [tinygo]
```

A generic runner will not infer this, which is the concrete reason `tool:
megalinter` cannot be the single answer.

## What it costs

- `ci/stack.go` — `ci.stack` as a list of named stacks, the string shorthand, the
  defaults table, selector resolution (all by default, by name, explicit `all`,
  first-only for `watch`), the resolved-names echo and the `ignoring [...]` line,
  union-with-dedupe, rejection of the reserved names `all`/`dev`/`prod`,
  `clog ci stack get <tools|watch|chk|make>`
- `ci/scan.go` — `Scan` struct with `source` and `artifact`, kind defaults, the
  required-`vuln` error, mode-aware resolution, `clog ci scan`
- `ci/targets.go` — `Scan` and `Stack` fields on `Target`; drop `KindPackage`;
  add `KindGitHubRelease` and `KindGitLabRelease`; enumeration for scanning that
  ignores deploy-mode membership but respects the stack selector
- `ci/deploy-github-release.go`, `ci/deploy-gitlab-release.go` — new publishers
  wrapping `gh release` and `glab release`, idempotent on re-run
- `bc/sub-flow.go` — `--fast` short-circuits the `chk` list; check failure fails
  the build in dev as well as prod
- `clog build [mode] [stack|all] [--fast]`, `clog deploy [mode] [stack|all]`,
  `clog watch [stack]` — the four verbs, thin wrappers over flow and `ci deploy`,
  sharing one argument parser; a missing tool refuses and prints its install
  line, and `watch all` is refused
- `embedfs/konfig.yaml` — `check.lint` and `check.scan` groups; fix the
  staticcheck pipeline and its catch; refuse rather than skip on a missing
  linter; `bc-ko` never pushes and stops passing `--sbom=none`
- `install/recipes/trivy.yaml` — new
- `.github/workflows/build.yaml` — one workflow replacing `build-hugo.yaml` and
  `build-golang.yaml`, with `security-events: write` and the SARIF upload
- tests — stack defaults, the string shorthand, selector resolution including
  `all` and the default-to-first rule, a parse error for each reserved name,
  multi-stack union and dedupe, target-to-stack binding, the override ladder, the
  required `vuln`, prod-only targets in dev mode, a missing `dir`, the two new
  kinds, and a check that fails in dev mode

## Settled

Every decision so far, in one place, because they are the spine of the design:

| | decision |
|---|---|
| 1 | `ci.stack` is the home; `build-hugo.yaml` and `build-golang.yaml` collapse into one `build.yaml` |
| 2 | `github-release` and `gitlab-release` land now, published with `gh` and `glab` |
| 3 | `package` is removed; `vuln: none` is explicit and required where no honest default exists |
| 4 | `pre-build` stays: `clog build` means ready to push, `--fast` makes no claim |
| 5 | strict dev mode lands as one change, no `ci.strict` escape hatch |
| 6 | a repo may declare several named stacks, each with its own tools, make steps and lint settings |
| 7 | `clog build` refuses on a missing tool and prints its install line |
| 8 | stacks are named; `clog build` and `clog deploy` take **all** stacks by default, `clog watch` takes the first; `all`, `dev` and `prod` are reserved |
| 9 | any verb acting on fewer stacks than the repo has names what it ignored |
| 10 | `--fast` drops the **whole** check list, not just the slow part |
| 11 | rollout order is pihuw, then bonfire, then www-mrmxf-com |

## `--fast` drops everything

`--fast` runs with an empty `chk` list. Not a shorter list, not the cheap checks
only — nothing. *Do the minimum quickly, I will take the risk.*

Narrowing it to skip only the slow phase would be a kindness that costs more than
it gives. A `--fast` that still lints is a `--fast` whose meaning depends on which
phases somebody classified as slow this month, and the developer then has to hold
that classification in their head to know what their green build proved. The
promise has to be trivial to state or it is not a promise: **`clog build` checked
everything, `clog build --fast` checked nothing.** Two sentences, no footnotes.

It follows that `--fast` never runs in CI, and the collapsed workflow has no way
to pass it.

## Status — what is built

Implemented in this repo, tested and green:

| area | state |
|---|---|
| `ci/stack.go` | named stacks, the three config shapes, defaults per type, selector, reserved names, union/dedupe, echo |
| `ci/scan.go` | source + artifact axes, kind defaults, required `vuln`, mode-aware refs, SARIF-ready env lines |
| `ci/targets.go` | `package` dropped, both release kinds added, `stack:` and `scan:` on a target |
| `ci/deploy.go` | deploy narrows to the selected stacks' targets |
| `bc/sub-flow.go` | `--fast` empties the check list; a failed check now aborts in **both** modes |
| `embedfs/konfig.yaml` | `check.lint`, `check.scan`, the four verbs, staticcheck and golangci-lint fixed |
| `install/recipes/trivy.yaml` | new, asset names verified against the live release |
| `.github/workflows/build.yaml` | one workflow, `security-events: write`, one SARIF upload |

Two things deliberately fall short of what this document specifies.

**The old workflows are kept, not deleted.** `build-hugo.yaml` and
`build-golang.yaml` still exist beside the new `build.yaml`. The rollout is
staged, and a repo still pinned to `@workflows-v1` has to keep building until it
moves. They go when www-mrmxf-com lands.

**`clog build` does not yet mean "never pushes" on a deploying CI run.** The rule
holds absolutely on a laptop and on every run that is not deploying: those write
`tmp/image.tar` and the artifact sweep reads it. A deploying CI run still pushes
from inside `bc-ko`, because `ci/deploy.go` registers a deployer for
`github-pages` and nothing else. With no container-registry deployer, moving the
push out of the build would mean the image never reaches the registry at all.

That deployer is the next piece of work, and it is not small: `ko` builds
multi-platform, so a deployer cannot simply `docker load` a tarball and push it.
Until it exists, the developer-facing promise is intact and the CI one is not.

## Rollout

Three repos, in order, each one proving something the last could not.

**The staged rollout is only possible because repos pin clog.** `.clog-version`
in the consumer repo fixes the release that `clog-prepare` installs. Strict dev
mode and the staticcheck fix are changes to clog itself, not to any repo's
config, so without that pin they would land everywhere on the same afternoon. A
repo joins the new behaviour by bumping its pin, which makes the order below real
rather than aspirational.

**1. pihuw** — Hugo to GitHub Pages, the repo that started this. It proves the
spine: `ci.stack` in its one-line form, the collapsed `build.yaml`, the source
sweep against the asset pipeline, the required `vuln: none` on a Pages target,
lint, and strict dev mode. It is the smallest blast radius and it exercises the
config path every other repo depends on.

**2. bonfire** — the first repo with more than one stack, so it proves everything
the selector touches: declaration order, the `ignoring [...]` line, union and
dedupe across stacks, per-stack lint settings, target-to-stack binding, and the
reserved-name parse errors. None of this can be tested on a single-stack repo,
because on one stack every selector resolves to the same answer.

**3. www-mrmxf-com** — a registry target and a Pages target in one repo. It proves
the image sweep, the two-kinds-at-once case and multi-target deploy.

### The gap in that order

**The ko change is not exercised until stage 3.** Making `clog build` stop pushing
is one of the riskier fixes here, because it moves when an image becomes public,
and neither pihuw nor a firmware repo will touch it. It ships untested through two
stages and then lands on the busiest repo of the three.

Worth pulling forward rather than discovering late: run www-mrmxf-com through
`clog build dev` against the new ko path as soon as stage 1 is green, without
promoting it or bumping its pin. That is a laptop command, it costs nothing, and
it turns the one genuinely unproven change into the one that was checked first.
