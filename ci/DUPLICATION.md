# Duplication audit — 2026-09-21

Found while adding the `github-pages` deployer. Each item is measured, not
estimated. Ordered by how much a fix removes.

## 1. The embedded konfig is copied, not layered — ~840 lines

| file | lines | identical to util's, non-comment |
|---|---|---|
| `util/embedfs/konfig.yaml` | 752 | — (the base) |
| `clog-mrmxf/embedfilesystem/konfig.yaml` | 841 | 422 / 467 |
| `utbd/embedfs/konfig.yaml` | 804 | 415 / 480 |

Plus two more copies under `*/embedfs/deb/konfig.yaml`.

clog-mrmxf's copy differs from util's by **one snippet key** (`htmltest`) and one
top-level key (`install:`). utbd's is similar. So roughly 420 duplicated lines in
each exist to carry a handful of unique ones.

`util/kfg` already has `MergeKonfig` and `PreventAutoMerge`, and
`config.FindEmbedded` already documents "last one found wins — this is usually the
project's embedded fs". The layering exists; the copies predate it or were never
collapsed onto it.

**Fix:** util's konfig becomes the base layer; clog-mrmxf and utbd ship only their
delta. This is the single biggest reduction available, and it is the mechanism that
stops the next copy drifting.

**Evidence that drift is real, not theoretical:** `todo-hugo-sites-config-only.md`
records the `Zone.Identifier` check shipping broken into 6 site configs from
`embedfs/konfig.yaml`, fixed in all 7 places on 2026-09-20.

## 2. Three ways to shell out to git in Go

| module | how | error handling |
|---|---|---|
| `buildinfo` | `git(dir, args...)` helper, 30s timeout | returns `(string, error)` |
| `ci` | `execGit.git(args...)` behind the `GitResolver` interface | swallows errors, returns `""` |
| `bc` | 8 separate inline `exec.Command("git", …)` | ad hoc per call site |

`bc` is the outlier: no helper, no timeout, and the retry/verify logic in
`sub-git-hash-origin.go` is written out by hand.

**Fix:** export one git runner — `buildinfo.Git(dir, args...)` is the natural home,
since it already has the timeout and both other modules already depend on
buildinfo — and have `bc` and `ci` call it. `ci` keeps `GitResolver` as its
testing seam; only the exec underneath changes.

Note the deployer added in this commit uses its own `execCommand` var, because
`buildinfo`'s helper is unexported. Exporting it should absorb that too.

## 3. `git:` shell snippets that Go already implements — 5 copies

The `snippets.git:` block appears in all five konfig files with
`branch`, `hash`, `suffix`, `tag`, `tree`, `origin`, `repo`, `unstaged`, `parents`.

`clog BC git` already implements `branch`, `hash`, `suffix`, `tag`, `tree` in Go.
One snippet, `vcode`, already delegates: `vcode: clog BC git tag ref`.

**Fix:** the rest follow `vcode` and delegate, or are deleted. Only `repo`,
`origin`, `unstaged` and `parents` have no Go equivalent; `repo`/`origin` are one
small `BC git repo` away, and `ci.slugFromRemote` is the regexp to reuse rather
than re-derive in bash. (A hand-rolled sed version of exactly this parse was
written and thrown away during this work: POSIX sed has no non-greedy quantifier,
so `(\.git)?` silently leaves `owner/name.git` behind. Go's regexp does not have
that trap — another reason to keep the parse in one Go place.)

## 4. Already fixed upstream — no action

- `ci.ReleaseVersion` delegates to `buildinfo.ReadGitState`, so version derivation
  is **not** duplicated between `ci` and `buildinfo`. Good precedent for item 2.
- `bc-hugo` and `bc-ko` duplication is already recorded in
  `clog-mrmxf/todo-hugo-sites-config-only.md` item 2, which retires both. Nothing
  to add, but note it makes item 3 cheaper: fewer snippets survive to delegate.
