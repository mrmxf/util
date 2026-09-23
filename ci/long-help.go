//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

const longHelp = `ci - normalize CI/CD event context for build steps

The 'ci' command turns the environment of whichever CI system is active
(GitHub Actions, GitLab CI, or a local working copy) into one explicit,
CI-agnostic description. Build steps read that description instead of raw
GITHUB_* / CI_* variables, so the same 'clog build' / 'clog deploy' path runs
identically on every platform and on a developer laptop.

Sub-commands:
  resolve   print the normalized event as JSON or KEY=value env lines
  mode      print this run's mode (dev|prod) or one of its ci.modes settings
  targets   list the deploy targets for this run's mode
  target    print one value from $CLOG_TARGET's data
  run       run a command with this repo's Infisical secrets in its environment
  require   fail early if the secrets/config a verb needs are missing
  policy    print what ci.policy decides for this run (build? deploy? why?)
  should    exit 0/1: does ci.policy allow build | deploy for this run?
  get       print a non-secret config value (ci.artifact, ci.title, ...)

See 'clog CI show event --help' and 'clog CI mode --help' for details.
Setting up CI secrets (Infisical OIDC, identities, .clog.yaml keys): clog CI --config-help`

const modeHelp = `ci mode - dev or prod? and what does that mean here?

  clog CI mode                    dev | prod
  clog CI mode show               this mode's ci.modes settings, as KEY=value lines
  clog CI mode get base-url       one setting
  CLOG_MODE=prod clog CI mode     force the mode (a laptop reproducing a CI run)

There is no staging (D-I.12). A run is dev or prod, and the same value drives the
build and the deploy:

  dev    a laptop, a pull/merge request, a branch push, a manual run
  prod   a tag push or a scheduled run that ci.policy.deploy.prod accepts
         (tag glob + a vX.Y.Z release tag; v1.2.0-rc1 is dev)

The mode picks the Infisical environment and identity, and which block of each
deploy target applies. Per-mode BUILD settings live in ci.modes:

  ci:
    modes:
      dev:  {base-url: "http://localhost:1313/", hugo-flags: "--buildDrafts"}
      prod: {base-url: "https://example.com/",   hugo-flags: ""}

  hugo build --baseURL "$(clog CI mode get base-url)" $(clog CI mode get hugo-flags)

$CLOG_ENV is the old name for $CLOG_MODE and still works with a warning;
CLOG_MODE=stage is an error rather than a guess.`

const resolveHelp = `CI show event - resolve the active CI event into normalized ref/repo/verb

Detects the active platform (GITHUB_ACTIONS / GITLAB_CI, else local git) and
emits a normalized result:

  verb           PR | PUSH | SCHEDULE | DISPATCH | LOCAL
  ref            git ref or SHA to check out
  repo           owner/name to check out from
  url            human URL of the repo (for logging)
  depth          checkout fetch-depth (0 = full history, 1 = shallow)
  actor          who triggered the event
  is_production  true → check out the production tag (scheduled runs)
  is_tag         true → the ref is a tag

Output:
  --format json  (default) indented JSON
  --format env   KEY=value lines for '>> $GITHUB_ENV' or a dotenv file

Locally (no CI vars set) it resolves from your git working copy, so you can
preview exactly what CI would check out before you push:

  clog CI show event
  clog CI show event --format env`

const runHelp = `ci run - run a command with this repo's Infisical secrets in its environment

  clog CI run -- clog deploy
  clog CI run --dry-run -- clog deploy     # log in + fetch, report names, run nothing

Where the secrets come from (config: ci.infisical in .clog.yaml):
  GitHub Actions  OIDC login as ci.infisical.identity.github.<dev|prod>-uuid
                  needs  permissions: {id-token: write}
  GitLab CI       OIDC login as ci.infisical.identity.gitlab.<dev|prod>-uuid
                  needs  id_tokens: {INFISICAL_ID_TOKEN: {aud: <ci.infisical.domain>}}
  laptop          your  infisical login  session
  pull request    none - the command runs without secrets

The mode (clog CI mode: dev|prod) picks the Infisical env via ci.infisical.env
and the identity: prod uses prod-uuid, dev uses dev-uuid.

Secrets are added to the command's environment only (not to $GITHUB_ENV), each
value is registered with ::add-mask:: on GitHub, and only names are logged.
The command also gets CLOG_CI_RUN=1. Its exit code becomes clog's.

Setup: clog CI --config-help`

const requireHelp = `ci require - fail early if what a verb needs is missing

  clog CI require deploy

Reads ci.require.<verb> from .clog.yaml:

  ci:
    require:
      deploy: {env: [AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY], optional: [HOOK_SLACK], config: [ci.deploy.bucket]}

  env       must be set and non-empty      → error, exit 1
  optional  may be missing                 → warning
  config    .clog.yaml keys, non-empty     → error, exit 1

Messages name the variable and where it comes from (Infisical env + path), never
its value. A verb with no entry passes. Put it first in a snippet so nothing
runs half-configured; to re-run under secrets on a laptop, the way
chiddingfoldbonfire re-runs under infisical:

  if ! clog CI require deploy; then
    [ -z "$CI" ] && [ "$CLOG_CI_RUN" != "1" ] && exec clog CI run -- clog deploy "$@"
    exit 1
  fi`

const policyHelp = `CI show policy / CI should - does this run build? does it deploy?

  clog CI show policy                     JSON: mode, event, ref, build, deploy, targets + reasons
  clog CI show policy --format env        build_mode= deploy_mode= do_build= do_deploy= deploy_targets=
                                     (for $GITHUB_ENV / a GitLab dotenv report)
  clog CI should deploy && clog deploy
  CLOG_MODE=dev  clog CI show policy      laptop preview: judged as a push of the current branch
  CLOG_MODE=prod clog CI show policy      laptop on a tag: judged as a push of that tag

Reads ci.policy from .clog.yaml:

  ci:
    policy:
      actors: [mrmxf]                        # optional: only these accounts build/deploy in CI
      build: [branch, tag, dispatch]         # events that build; missing list = build always
      deploy:                                # keyed by mode (see clog CI mode): dev, prod
        dev:  {branches: [main, rc, dev]}
        prod: {tags: ["v*"], schedule: false}

Events: branch (push), tag (tag push), dispatch (manual/api), schedule, pr, local.
A run deploys when the rule for its mode matches:
  branch, dispatch, local  ref matches branches      (globs; * also matches "/")
  tag                      ref matches tags
  schedule                 schedule: true
A prod tag must also be a release tag: vX.Y.Z or X.Y.Z, nothing after it.
releases.yaml is history and is not read; releases-yaml: is ignored with a warning.
A scheduled prod run should build the newest release tag on the default branch:
  clog BC git checkout production --branch origin/main

Hard rules: pull/merge requests never deploy; no rule for the mode = no deploy;
no ci.policy.build = everything builds; a run that does not build does not deploy.`

const deployHelp = `ci deploy - publish the build to this run's targets

  clog CI deploy                    every target that deploys in this mode
  clog CI deploy --target pages     just that one
  clog CI deploy --dry-run          say what would happen, change nothing

Each ci.targets.<name> has a kind, and the kind decides how it is published.
Targets run in config order; the first failure stops the run, because a
half-deployed release is worse than a stopped one.

  github-pages    force-push a directory to a Pages branch

Kinds that clog CI target list accepts but that are not shown above are declared and
validated, but have no deployer yet; naming one is an error, not a silent skip.

github-pages target data, per mode:

  targets:
    pages:
      kind: github-pages
      prod: {dir: kodata, branch: gh-pages, cname: example.org}

  dir     directory to publish            required
  branch  branch to force-push to         default gh-pages
  repo    owner/name to push to           default: this repo's origin
  cname   custom domain, written as CNAME optional

It writes .nojekyll, because Pages otherwise runs Jekyll and drops files whose
names begin with an underscore.

Credentials: $GH_TOKEN or $GITHUB_TOKEN in CI; the origin remote (so your ssh
key) on a laptop. The token never reaches a log - it is redacted from the push
URL and from any git error output.

Pages must already point at the branch. This will publish the branch but never
turns Pages on or changes a live site's serving source: that is a one-time
setting, not something a deploy should do behind your back.
`

const targetsHelp = `CI target list / CI target get - where does this run deploy to?

  clog CI target list                       target names for this mode, one per line
  clog CI target list --kind container-registry
  CLOG_TARGET=bucket clog CI target get prefix
  CLOG_TARGET=bucket clog CI target get kind

A deploy sends the build to one destination per target, so a project with two
destinations declares two targets and the deploy step runs twice (D-I.14):

  for t in $(clog CI target list); do
    CLOG_TARGET="$t" clog CI run -- bash -c 'clog CI require deploy && clog deploy' || exit 1
  done

Config - kind, the secrets the destination needs, and a block per mode:

  ci:
    targets:
      bucket:
        kind: bucket        # container-registry | bucket | package
                            # cloudflare-pages | github-pages | gitlab-pages
        require: [AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY]
        modes: [dev, prod]  # optional; default: the modes that have a block
        dev:  {bucket: my-bucket, prefix: bin/dev}
        prod: {bucket: my-bucket, prefix: "bin/{tag}"}

Values expand {tag} {version} {sha} {mode}. "kind" is readable as a key.
clog CI require deploy also checks the current target's require list.`
