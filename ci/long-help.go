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

See 'clog ci resolve --help' and 'clog ci mode --help' for details.
Setting up CI secrets (Infisical OIDC, identities, .clog.yaml keys): clog ci --config-help`

const modeHelp = `ci mode - dev or prod? and what does that mean here?

  clog ci mode                    dev | prod
  clog ci mode show               this mode's ci.modes settings, as KEY=value lines
  clog ci mode get base-url       one setting
  CLOG_MODE=prod clog ci mode     force the mode (a laptop reproducing a CI run)

There is no staging (D-I.12). A run is dev or prod, and the same value drives the
build and the deploy:

  dev    a laptop, a pull/merge request, a branch push, a manual run
  prod   a tag push or a scheduled run that ci.policy.deploy.prod accepts
         (tag glob + releases.yaml build: prod)

The mode picks the Infisical environment and identity, and which block of each
deploy target applies. Per-mode BUILD settings live in ci.modes:

  ci:
    modes:
      dev:  {base-url: "http://localhost:1313/", hugo-flags: "--buildDrafts"}
      prod: {base-url: "https://example.com/",   hugo-flags: ""}

  hugo build --baseURL "$(clog ci mode get base-url)" $(clog ci mode get hugo-flags)

$CLOG_ENV is the old name for $CLOG_MODE and still works with a warning;
CLOG_MODE=stage is an error rather than a guess.`

const resolveHelp = `ci resolve - resolve the active CI event into normalized ref/repo/verb

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

  clog ci resolve
  clog ci resolve --format env`

const runHelp = `ci run - run a command with this repo's Infisical secrets in its environment

  clog ci run -- clog deploy
  clog ci run --dry-run -- clog deploy     # log in + fetch, report names, run nothing

Where the secrets come from (config: ci.infisical in .clog.yaml):
  GitHub Actions  OIDC login as ci.infisical.identity.github.<dev|prod>-uuid
                  needs  permissions: {id-token: write}
  GitLab CI       OIDC login as ci.infisical.identity.gitlab.<dev|prod>-uuid
                  needs  id_tokens: {INFISICAL_ID_TOKEN: {aud: <ci.infisical.domain>}}
  laptop          your  infisical login  session
  pull request    none - the command runs without secrets

The mode (clog ci mode: dev|prod) picks the Infisical env via ci.infisical.env
and the identity: prod uses prod-uuid, dev uses dev-uuid.

Secrets are added to the command's environment only (not to $GITHUB_ENV), each
value is registered with ::add-mask:: on GitHub, and only names are logged.
The command also gets CLOG_CI_RUN=1. Its exit code becomes clog's.

Setup: clog ci --config-help`

const requireHelp = `ci require - fail early if what a verb needs is missing

  clog ci require deploy

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

  if ! clog ci require deploy; then
    [ -z "$CI" ] && [ "$CLOG_CI_RUN" != "1" ] && exec clog ci run -- clog deploy "$@"
    exit 1
  fi`

const policyHelp = `ci policy / ci should - does this run build? does it deploy?

  clog ci policy                     JSON: mode, event, ref, build, deploy, targets + reasons
  clog ci policy --format env        build_mode= deploy_mode= do_build= do_deploy= deploy_targets=
                                     (for $GITHUB_ENV / a GitLab dotenv report)
  clog ci should deploy && clog deploy
  CLOG_MODE=dev  clog ci policy      laptop preview: judged as a push of the current branch
  CLOG_MODE=prod clog ci policy      laptop on a tag: judged as a push of that tag

Reads ci.policy from .clog.yaml:

  ci:
    policy:
      actors: [mrmxf]                        # optional: only these accounts build/deploy in CI
      build: [branch, tag, dispatch]         # events that build; missing list = build always
      deploy:                                # keyed by mode (see clog ci mode): dev, prod
        dev:  {branches: [main, rc, dev]}
        prod: {tags: ["v*"], releases-yaml: prod, schedule: false}

Events: branch (push), tag (tag push), dispatch (manual/api), schedule, pr, local.
A run deploys when the rule for its mode matches:
  branch, dispatch, local  ref matches branches      (globs; * also matches "/")
  tag                      ref matches tags
  schedule                 schedule: true
and, if releases-yaml is set, the top releases.yaml entry has that build value.

Hard rules: pull/merge requests never deploy; no rule for the mode = no deploy;
no ci.policy.build = everything builds; a run that does not build does not deploy.`

const targetsHelp = `ci targets / ci target - where does this run deploy to?

  clog ci targets                       target names for this mode, one per line
  clog ci targets --kind container-registry
  CLOG_TARGET=bucket clog ci target get prefix
  CLOG_TARGET=bucket clog ci target get kind

A deploy sends the build to one destination per target, so a project with two
destinations declares two targets and the deploy step runs twice (D-I.14):

  for t in $(clog ci targets); do
    CLOG_TARGET="$t" clog ci run -- bash -c 'clog ci require deploy && clog deploy' || exit 1
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
clog ci require deploy also checks the current target's require list.`
