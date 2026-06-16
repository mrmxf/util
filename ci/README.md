# util/ci — `clog ci`

Normalize the environment of whichever CI system is active — **GitHub Actions,
GitLab CI, or a developer laptop** — into one explicit, CI-agnostic struct.

## Intent

CI YAML traditionally carries a wall of inline `case "$GITHUB_EVENT_NAME"` bash
to work out *what to check out*: which ref, which repo, how deep, is this a
production run. That logic is untestable, duplicated per workflow, and drifts
from what a developer runs locally — the classic "works on my machine" gap.

`clog ci` moves that logic into testable Go. A build step asks `clog ci resolve`
once and reads a normalized result instead of raw `GITHUB_*` / `CI_*`
variables, so the **same `clog build` / `clog deploy` path runs identically on
GitHub, GitLab, and your laptop**. Thin YAML, fat clog.

The resolver detects the platform (`GITHUB_ACTIONS` / `GITLAB_CI`, else local
git) and emits:

| field | meaning |
| --- | --- |
| `ci` | `github` \| `gitlab` \| `local` |
| `verb` | `PR` \| `PUSH` \| `SCHEDULE` \| `DISPATCH` \| `LOCAL` |
| `ref` | git ref or SHA to check out |
| `repo` | `owner/name` to check out from |
| `url` | human URL of the repo (for logging) |
| `depth` | checkout fetch-depth (`0` = full history, `1` = shallow) |
| `actor` | who triggered the event |
| `is_production` | `true` → check out the production tag (scheduled runs) |
| `is_tag` | `true` → the ref is a tag |

## Usage

### 1. Preview locally what CI will check out

Run it on your laptop, before pushing, to see exactly what CI would compute
(resolved from your git working copy):

```console
$ clog ci resolve
{
  "ci": "local",
  "verb": "LOCAL",
  "ref": "feature/widgets",
  "repo": "mrmxf/clog",
  "url": "git@github.com:mrmxf/clog.git",
  "depth": 1,
  "actor": "Bruce",
  "is_production": false,
  "is_tag": false
}
```

### 2. Drive a GitHub Actions checkout (env form)

Write the normalized fields into `$GITHUB_ENV` (the lowercase keys
`verb/ref/repo/url/depth` match the names the legacy workflows used, so this is
a drop-in replacement for the old inline `case`):

```yaml
- name: resolve event → ref/repo/depth/verb
  run: clog ci resolve --format env >> "$GITHUB_ENV"
- name: checkout
  uses: actions/checkout@<sha>
  with:
    ref: ${{ env.ref }}
    repository: ${{ env.repo }}
    fetch-depth: ${{ env.depth }}
- name: production checkout (scheduled runs)
  if: env.is_production == 'true'
  run: clog BC git checkout production
```

### 3. Branch a script on the result (JSON form)

The same command works under GitLab CI or anywhere; consume the JSON with `jq`:

```bash
verb="$(clog ci resolve | jq -r .verb)"
case "$verb" in
  PR)       echo "validating a pull/merge request" ;;
  SCHEDULE) echo "nightly production build" ;;
  *)        echo "branch push ($verb)" ;;
esac
```

## Notes

- `--format json` (default) or `--format env` (KEY=value lines for `$GITHUB_ENV`
  / a dotenv file).
- Local mode is best-effort: outside a git repo it still prints a sensible
  `LOCAL` result with empty fields rather than failing.
- Event/ref mapping mirrors the original `mrmxf/clog` `build-golang` workflow;
  `workflow_dispatch` resolves refs identically to `PUSH` but reports the
  distinct `DISPATCH` verb for observability.
