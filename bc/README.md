# util/bc — `clog BC` (Build Control)

Build-control primitives for CI/build pipelines: git/tag/release inspection, a
structured progress **stash**, and a check→build **flow** runner — all as a
`clog BC …` command tree and a small Go API.

## Why this package exists

The guiding principle of clog is **thin YAML, fat clog**: CI pipelines should be
a checkout plus a few `clog …` calls, with the real logic in testable Go rather
than sprawling shell. `bc` is where the build-orchestration logic lives — the
things a pipeline needs to ask and record:

- *What is the production tag / current ref / working-tree state?* → `BC git …`
- *What does `releases.yaml` say we're building?* → `BC releases …`, `BC is …`
- *Record structured progress so a later step can report it* → `BC stash* …`
- *Run the check and build steps for this release* → `BC flow`

Because it's Go, every one of these is unit-tested and behaves identically on a
laptop and in CI.

## Design decisions

These are the non-obvious choices a future reader should understand before
changing anything here.

### 1. Config comes from overridable hooks, not an app struct

`bc` is a **public, app-agnostic** module (`github.com/mrmxf/util/bc`). It does
*not* import any single app's config. Everything it needs from the host app is a
small set of function hooks in [`config.go`](./config.go), each defaulting to
`util/kfg`:

| hook | returns | default |
| --- | --- | --- |
| `bc.Releases` | `[]kfg.AppRelease` (current first) | `kfg.Releases()` |
| `bc.ReleasesPath` | path of `releases.yaml` | `kfg.ReleasesPath()` |
| `bc.StashPath` | stash file path (`""` = default) | `""` |
| `bc.DryRun` | dry-run mode? | `false` |

A host wires them at bootstrap (clog-app and clog-mrmxf both do):

```go
bc.StashPath = func() string { return my.App.StashPath }
bc.DryRun    = my.App.IsDryRun
```

Tests override the same hooks to inject fixtures. This is what let `bc` move out
of the old `clog-mrmxf` monolith (where it read `my.App` directly) without a
rewrite.

### 2. The stash is a `flow → phase → step` tree on disk

A build runs as **many separate `clog` processes** (a workflow step, then
`clog build`, which shells out to more `clog` calls). They can't share memory, so
progress is accumulated in a YAML **stash** file that any process can append to
and a final step (`clog SlackStash`) reads to render one report.

```yaml
flow:
  build:            # a flow groups related work
    compile:        # a phase within the flow
      - step: go-build
        level: 2     # slog level: INFO(2) SUCCESS(3) WARN(4) ERROR(5) …
        message: "ℹ️  INFO: compiled"
        timestamp: 2026-06-16T16:44:58Z
```

### 3. Stash names are case-insensitive identifiers — on purpose

The schema ([`stash-schema.json`](./stash-schema.json)) validates the *structure*
and each step's contents (required `step`/`message`/`timestamp`, `level` 0–7, no
stray keys), but **does not constrain letter-case** of flow/phase/step names.
That's deliberate: the uppercase summary flow `FLOW` coexists with lowercase
flows like `build`/`check`, and `SlackStash` treats a validation failure as fatal
— so the schema must accept any stash the running system actually writes.
Enforcing a case convention would break the Slack path. Names must, however, be
valid identifiers (start with a letter), which the schema *does* enforce.

### 4. Concurrent stash writes are advisory-locked

`AppendStash` is a load→modify→write cycle. Because stashLog runs as separate
processes, [`stash-lock.go`](./stash-lock.go) guards the cycle with an in-process
mutex **and** a cross-process advisory `<stash>.lock` (O_EXCL). A stale lock from
a crashed process is stolen after 30 s, and after a 10 s wait it proceeds
**unlocked rather than failing the build** — a slightly racy log beats a red
pipeline. (The lock dir is created first; otherwise the `O_EXCL` create would
`ENOENT` and stall every write.)

### 5. The release model is data-driven, not hard-coded

"Production" is simply the first `{flow: main, build: prod}` row in
`releases.yaml`; "current" is row 0. `bc` hard-codes no servers, hosts, or tag
names, which is why it's general enough to be public. The `v` prefix on Go tags
is the responsibility of whoever maintains `releases.yaml`.

### 6. Security / resilience baked in

- Git operations that hit the network (`ls-remote`, `push`) are **timeout-bounded**
  (90 s) so a hung remote can't wedge a build — see [`git-net.go`](./git-net.go).
- Tag/ref arguments are checked by `safeRef` and passed after `--`, so a value
  like `-x` can't be smuggled in as a git option.
- The stash file is written `0600` (it can capture log text containing secrets).

### 7. Two git backends, by necessity

Reads and local operations use **go-git** (`PlainOpen`); the **push** path shells
out to the `git` binary because go-git's push is unreliable across auth setups.
Expect both in the tag code.

## Command tree

```
clog BC
├── git
│   ├── branch                 print the current branch
│   ├── suffix                 git-state suffix for version strings
│   ├── tree                   working-tree status …
│   │   ├── ahead | behind | clean | unstaged
│   ├── checkout [ref]         checkout a ref
│   │   ├── production         checkout the production tag
│   │   └── list               list candidate checkout refs
│   ├── hash                   commit hashes …
│   │   ├── head | origin | ref [ref] | prod
│   └── tag                    tags …
│       ├── head | origin | ref | prod
│       └── tidy               create/replace + push the release tag (--force, --dryrun)
├── releases                   read releases.yaml …
│   ├── version | date | flow | build | note     print a field of the current release
│   └── yaml                   print the releases.yaml path
├── is <field> <value>         compare a current-release field; exit 0 match / 1 no-match
├── stash                      inspect the stash …
│   ├── get path               print the stash file path
│   ├── get error              print the accumulated error text
│   └── has error              exit non-zero if the stash holds any errors
├── stashLog … -1 <flow> -2 <phase> -3 <step> -I "msg"   log AND record to the stash
├── flow --check "…" --build "…"   run check then build steps, accumulating status
├── semver                     semantic-version helpers
└── linkerpath                 ldflags path for the SemVerJSON build variable
```

Run any node with `--help` for flags and examples.

### `flow` — the orchestrator

`clog BC flow` runs check steps then build steps, recording each into the stash
and aborting a **production** build on the first error (non-production continues
and reports). Steps come from `--check`/`--build` or the `$CHK`/`$MAKE`
environment variables:

```bash
clog BC flow --check "pre-build tools" --build "golang deploy"
# or
export CHK="lint test"; export MAKE="build deploy"; clog BC flow
```

## Go API (for embedders)

`SlackStash` and other commands consume `bc` directly. The stash surface:

```go
func LoadStash() Stash                                  // read (empty on missing/corrupt)
func UpdateStash(Stash) error                           // overwrite
func AppendStash(flow, phase, step string,
                 level StashLevels, msg string) error   // locked read-modify-write
func ResetStash() error                                 // truncate to empty
func LoadValidateStash() (Stash, error)                 // read + schema-validate
func ValidateStashYAML([]byte) error                    // validate raw YAML
```

Core types (`defs.go`): `Stash{ Flow map[FlowName]FlowPhases }`,
`FlowPhases map[PhaseName]PhaseSteps`, `PhaseSteps []FlowPhaseStep`,
`FlowPhaseStep{ Step, Level, Message, Timestamp }`, and the `StashLevels`
constants `LvlDebug … LvlEmergency`.

Git/release helpers: `Checkout`/`CheckoutRef`/`CheckoutBranch`/`CheckoutTag`,
`GitTagRef()`, `GitTagProduction()`.

## Tests

```bash
go test ./...        # from util/bc
```

Tests are table-driven and override the `config.go` hooks for fixtures; the
git-dependent tests skip gracefully outside a repo.
