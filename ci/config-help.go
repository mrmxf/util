//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

// configHelp is printed by `clog ci --config-help`. It is plain text kept in
// its own file so it can be edited without touching code. Audience: a coder
// (or an AI) who knows Infisical and GitHub/GitLab but has not wired this
// particular setup before. Keep it terse, literal and copy-pasteable; no
// backticks (Go raw string).
const configHelp = `clog ci --config-help : CI secrets via Infisical OIDC

STATUS
  implemented: config contract, clog ci run / require / policy / should / get / mode / targets /
  target, util workflows v1. No staging: a run is dev or prod (D-I.12).

MODEL
  CI yaml     : triggers + OIDC permission + one "clog ci run -- clog <verb>" per job. No secrets.
  .clog.yaml  : every non-secret value, incl. which Infisical project/env/identity to use.
  Infisical   : secrets only. CI logs in with the platform OIDC token (no stored credential).
  workflows   : mrmxf/util/.github/workflows/{build-golang,build-hugo,deploy-s3}.yaml@workflows-v1
                  (first step mrmxf/util/.github/actions/clog-prepare); GitLab: util/gitlab/clog.gitlab-ci.yml
  gate        : clog ci policy --format env >> $GITHUB_ENV
                  -> build_mode deploy_mode do_build do_deploy deploy_targets
                  -> if: env.do_build == 'true'   /   if: env.do_deploy == 'true'
                  (or in a script:  clog ci should deploy || exit 0)
  deploy      : one run per target (D-I.14):
                  for t in $deploy_targets; do
                    CLOG_TARGET="$t" clog ci run -- bash -c 'clog ci require deploy && clog deploy' || exit 1
                  done
  flow        : clog ci run -- clog <verb>
                = clog ci mode -> platform OIDC token -> POST <domain>/api/v1/auth/oidc-auth/login
                  -> GET <domain>/api/v4/secrets (imports merged) -> exec <verb> with secrets in env
                  (no Infisical CLI needed in CI; laptop uses the infisical login session)

MODE -> INFISICAL (clog ci mode decides; see clog ci mode --help)
  mode  when                                        infisical env  identity   login
  dev   laptop, PR, branch push, dispatch           dev            dev-uuid   OIDC (laptop: user login)
  prod  tag push / schedule that ci.policy.deploy.  prod           prod-uuid  OIDC
        prod accepts (tag glob + releases.yaml prod)
  Build mode and deploy mode are always the same. A PR gets NO secrets, ever.
  Force a mode: CLOG_MODE=prod (CLOG_ENV still read with a warning; =stage is an error).

.CLOG.YAML (keys are lowercase; values below are the contract)
  ci:
    artifact: "<name>"                       # workflows: build uploads / deploy downloads it
    title: "<slack title>"
    policy:                                  # clog ci policy --help
      actors: [<login>]                      # optional: only these accounts build/deploy in CI
      build: [branch, tag, dispatch]         # branch|tag|dispatch|schedule|pr|local
      deploy:                                # keyed by MODE; PRs never deploy
        dev:  {branches: [main, rc, dev]}    # globs, * also matches "/"
        prod: {tags: ["v*"], releases-yaml: prod}    # + schedule: true to let scheduled runs deploy
    modes:                                   # per-mode BUILD settings: clog ci mode get <key>
      dev:  {base-url: "http://localhost:1313/", hugo-flags: "--buildDrafts"}
      prod: {base-url: "https://example.com/"}
    targets:                                 # deploy destinations: clog ci targets / target get
      bucket:
        kind: bucket                         # container-registry | bucket | package
                                             # cloudflare-pages | github-pages | gitlab-pages
        require: [AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY]   # secrets this destination needs
        # modes: [prod]                      # optional; default: the modes with a data block
        dev:  {bucket: "<s3-bucket>", prefix: "bin/dev"}
        prod: {bucket: "<s3-bucket>", prefix: "bin/{tag}"}    # tokens {tag} {version} {sha} {mode}
    require:                                 # extra checks, merged with the target's require
      deploy: {optional: [HOOK_SLACK]}
    infisical:
      domain: "https://eu.infisical.com"     # exact instance URL (EU != app.infisical.com)
      project-id: "<uuid>"                   # Project Settings -> Project ID; same as .infisical.json workspaceId
      project-slug: "<slug>"
      path: "/<repo>"                        # folder with this repo's secrets
      audience: ""                           # optional; OIDC aud clog requests, default = domain
      env: {dev: dev, prod: prod}            # mode -> Infisical env slug
      identity:
        github: {dev-uuid: "<uuid>", prod-uuid: "<uuid>"}
        gitlab: {dev-uuid: "<uuid>", prod-uuid: "<uuid>"}
  identity uuid = Access Control -> Identities -> <name> -> Identity ID. NOT the name.
  also commit .infisical.json {"workspaceId": "<project-id>"} for laptop infisical CLI.

INFISICAL PROJECT LAYOUT
  envs     : dev, prod
  folders  : /shared (org-wide: HOOK_SLACK, DOCKER_PAT), /<repo> (repo secrets)
  import   : in EACH env, /<repo> -> Add secret import -> env=<same env>, path=/shared
             path is the FOLDER (/shared). NOT /shared/HOOK_SLACK (silently imports nothing).
  verify   : infisical export --projectId=<id> --env=dev --path=/<repo> --format=json
             must list the repo secrets AND the /shared ones.

MACHINE IDENTITIES (one per platform x {dev,prod}; name e.g. gh-<org>-dev, gl-<org>-prod)
  1 create at org level, add to the project.
  2 project role: custom, read-only, ONE env.
      ci-read-dev : secrets read + folders read + secret-imports read, environment=dev
      ci-read-prod: same, environment=prod
    NOT "viewer": viewer reads every env, so a branch push could read prod.
  3 Authentication -> Add -> OIDC Auth (the identity is useless until this exists):

    GitHub Actions
      OIDC Discovery URL : https://token.actions.githubusercontent.com
      Issuer             : https://token.actions.githubusercontent.com
      Audiences          : <ci.infisical.domain>          # clog requests the token with this audience
      Subject  dev-uuid  : repo:<owner>/<repo>:ref:refs/*        # refs/* not refs/heads/*: a tag
                                                                   # whose release is not prod is a dev build
      Subject  prod-uuid : repo:<owner>/<repo>:ref:refs/tags/v*
      Claims (optional)  : job_workflow_ref = <owner>/util/.github/workflows/*@refs/tags/workflows-v1
    GitLab CI (gitlab.com)
      OIDC Discovery URL : https://gitlab.com
      Issuer             : https://gitlab.com
      Audiences          : <ci.infisical.domain>          # must equal id_tokens aud in .gitlab-ci.yml
      Subject  dev-uuid  : project_path:<group>/<project>:ref_type:*:ref:*
      Subject  prod-uuid : project_path:<group>/<project>:ref_type:tag:ref:v*
    Access Token TTL     : 900 (a job, not a day)

GOTCHAS
  - Audiences blank = audience check SKIPPED (signature, issuer, subject still checked).
    OK while reading claims; set it before real secrets, or any token minted for another
    service with a matching subject can be replayed to Infisical.
  - Subject/claims match exact OR bash glob, where * also matches "/":
    refs/heads/* covers feature/x; repo:<owner>/* would admit EVERY repo of the owner.
  - Subject format drifts: GitHub repos created after 2026-07-15 may present
    repo:<owner>@<owner-id>/<repo>@<repo-id>:... Read the real claims (below) before binding.
  - Reusable workflows: sub is the CALLER repo; the util workflow shows up in job_workflow_ref.
  - Scheduled prod runs present a BRANCH subject (refs/heads/<default>). A tags-only prod
    identity rejects them. Either do not schedule prod, or bind prod to a GitHub Environment:
    job "environment: prod" -> sub = repo:<owner>/<repo>:environment:prod.
  - Pull requests (esp. forks) never reach Infisical: clog ci run skips the fetch and runs the command without secrets.
  - GitHub needs "permissions: id-token: write" on the caller workflow AND the reusable job.
  - GitLab needs, per job:   id_tokens: {INFISICAL_ID_TOKEN: {aud: <ci.infisical.domain>}}
  - infisical CLI flag names differ by version (--jwt vs --oidc-jwt); clog ci run avoids the CLI in CI.
  - infisical run does not mask output. Never echo secrets; on GitHub clog emits ::add-mask::.

READ THE REAL OIDC CLAIMS (print claims, never the token)
  GitHub step (job has id-token: write):
    curl -sH "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
      "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=https://eu.infisical.com" \
    | python3 -c 'import sys,json,base64;p=json.load(sys.stdin)["value"].split(".")[1];print(json.dumps({k:v for k,v in json.loads(base64.urlsafe_b64decode(p+"==")).items() if k in("iss","aud","sub","ref","job_workflow_ref")},indent=1))'
  GitLab job (with id_tokens above):
    python3 -c 'import os,json,base64;p=os.environ["INFISICAL_ID_TOKEN"].split(".")[1];c=json.loads(base64.urlsafe_b64decode(p+"=="));print({k:c.get(k) for k in("iss","aud","sub","ref","ref_type")})'

VERIFY FROM A LAPTOP (user login: infisical login --domain=<ci.infisical.domain>)
  names only : infisical export --projectId=<id> --env=<env> --path=/<repo> --format=json | jq -r '.[].key'
  oidc bound : curl -sH "Authorization: Bearer $(infisical user get token --plain)" \
                 <domain>/api/v1/auth/oidc-auth/identities/<identity-uuid> | jq .identityOidcAuth
               "does not have OIDC Auth attached" = step 3 not done.
  imports    : curl -sH "Authorization: Bearer $(infisical user get token --plain)" \
                 "<domain>/api/v2/secret-imports?projectId=<id>&environment=<env>&path=/<repo>" | jq '.secretImports[]|{importPath}'
  local run  : clog ci run --dry-run -- clog deploy     (then without --dry-run)`
