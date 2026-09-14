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
  env       print the environment this event builds, or one of its settings
  run       run a command with this repo's Infisical secrets in its environment
  require   fail early if the secrets/config a verb needs are missing
  policy    print what ci.policy decides for this run (build? deploy? why?)
  should    exit 0/1: does ci.policy allow build | deploy for this run?
  get       print a non-secret config value (ci.artifact, ci.title, ...)

See 'clog ci resolve --help' and 'clog ci env --help' for details.
Setting up CI secrets (Infisical OIDC, identities, .clog.yaml keys): clog ci --config-help`

const envHelp = `ci env - which environment does this event build, and with what settings?

'ci resolve' answers "what happened". 'ci env' answers "so what do I build?".
It maps the resolved event onto exactly one environment name:

  dev     a laptop build, or a pull/merge request. Never publishes
  stage   a branch push or a manual dispatch
  prod    a tag push, or a scheduled run (which rebuilds the production tag)

A pull request is dev even when its ref is a tag: untrusted input must never
select a deployment target.

The SETTINGS for each environment live in the consuming repo's .clog.yaml under
an 'environments' key, because a URL or an image tag is site-specific while this
mapping is not:

  environments:
    dev:   {base-url: "http://localhost:1313/", hugo-flags: "--buildDrafts", image-tags: []}
    stage: {base-url: "https://staging.example.com/", image-tags: ["latest-stage"]}
    prod:  {base-url: "https://example.com/", image-tags: ["latest"]}

Usage:
  clog ci env                    print the environment name
  clog ci env show               print the whole resolved row as KEY=value lines
  clog ci env get base-url       print one setting

Scalars print as-is; list values print one item per line, so both of these work:

  hugo build --baseURL "$(clog ci env get base-url)"
  for tag in $(clog ci env get image-tags); do ko build --tags "$tag"; done

Override the mapping with $CLOG_ENV to reproduce another environment locally:

  CLOG_ENV=prod clog ci env show

An unknown $CLOG_ENV is an error rather than a fallback — a typo should fail the
build, not quietly publish the wrong site.`

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

The clog environment (clog ci env: dev|stage|prod) picks the Infisical env via
ci.infisical.env and the identity: prod uses prod-uuid, dev and stage use dev-uuid.

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

  clog ci policy                     JSON: env, event, ref, build, deploy + reasons
  clog ci policy --format env        clog_env= do_build= do_deploy=  (for $GITHUB_ENV)
  clog ci should deploy && clog deploy
  CLOG_ENV=stage clog ci policy      laptop preview: judged as a push of the current branch
  CLOG_ENV=prod  clog ci policy      laptop on a tag: judged as a push of that tag

Reads ci.policy from .clog.yaml:

  ci:
    policy:
      actors: [mrmxf]                        # optional: only these accounts build/deploy in CI
      build: [branch, tag, dispatch]         # events that build; missing list = build always
      deploy:                                # per clog env (see clog ci env): stage, prod
        stage: {branches: [main, rc, dev]}
        prod:  {tags: ["v*"], releases-yaml: prod, schedule: false}

Events: branch (push), tag (tag push), dispatch (manual/api), schedule, pr, local.
A run deploys when the rule for its clog env matches:
  branch, dispatch, local  ref matches branches      (globs; * also matches "/")
  tag                      ref matches tags
  schedule                 schedule: true
and, if releases-yaml is set, the top releases.yaml entry has that build value.

Hard rules: pull/merge requests never deploy; no rule for the env = no deploy;
no ci.policy.build = everything builds; a run that does not build does not deploy.`
