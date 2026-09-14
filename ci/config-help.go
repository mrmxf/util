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
  implemented: config contract, clog ci run / require / policy / should / get, util workflows v1.

MODEL
  CI yaml     : triggers + OIDC permission + one "clog ci run -- clog <verb>" per job. No secrets.
  .clog.yaml  : every non-secret value, incl. which Infisical project/env/identity to use.
  Infisical   : secrets only. CI logs in with the platform OIDC token (no stored credential).
  workflows   : mrmxf/util/.github/workflows/{build-golang,build-hugo,deploy-s3}.yaml@workflows-v1
                  (first step mrmxf/util/.github/actions/clog-prepare); GitLab: util/gitlab/clog.gitlab-ci.yml
  gate        : clog ci policy --format env >> $GITHUB_ENV  -> if: env.do_deploy == 'true'
                  (or in a script:  clog ci should deploy || exit 0)
  flow        : clog ci run -- clog <verb>
                = clog ci env -> platform OIDC token -> POST <domain>/api/v1/auth/oidc-auth/login
                  -> GET <domain>/api/v4/secrets (imports merged) -> exec <verb> with secrets in env
                  (no Infisical CLI needed in CI; laptop uses the infisical login session)

CLOG ENV -> INFISICAL (clog ci env decides; see clog ci env --help)
  clog env  when                        infisical env  identity   login
  dev       laptop, pull/merge request  dev            -          laptop: user login. PR: NO secrets, never.
  stage     branch push, dispatch       dev            dev-uuid   OIDC
  prod      tag push, schedule          prod           prod-uuid  OIDC

.CLOG.YAML (keys are lowercase; values below are the contract)
  ci:
    artifact: "<name>"                       # workflows: build uploads / deploy downloads it
    title: "<slack title>"
    docker-ns: "<docker hub account>"        # optional; DOCKER_PAT comes from Infisical
    policy:                                  # clog ci policy --help
      actors: [<login>]                      # optional: only these accounts build/deploy in CI
      build: [branch, tag, dispatch]         # branch|tag|dispatch|schedule|pr|local
      deploy:                                # keyed by clog env; PRs never deploy
        stage: {branches: [main, rc, dev]}   # globs, * also matches "/"
        prod:  {tags: ["v*"], releases-yaml: prod}   # + schedule: true to let scheduled runs deploy
    deploy:
      bucket: "<s3-bucket>"                  # non-secret config, not in Infisical
    require:                                 # fail before any work if missing
      deploy: {env: [AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY], optional: [HOOK_SLACK], config: [ci.deploy.bucket]}
    infisical:
      domain: "https://eu.infisical.com"     # exact instance URL (EU != app.infisical.com)
      project-id: "<uuid>"                   # Project Settings -> Project ID; same as .infisical.json workspaceId
      project-slug: "<slug>"
      path: "/<repo>"                        # folder with this repo's secrets
      audience: ""                           # optional; OIDC aud clog requests, default = domain
      env: {dev: dev, stage: dev, prod: prod}   # clog env -> Infisical env slug (all three keys)
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
      Subject  dev-uuid  : repo:<owner>/<repo>:ref:refs/heads/*
      Subject  prod-uuid : repo:<owner>/<repo>:ref:refs/tags/v*
      Claims (optional)  : job_workflow_ref = <owner>/util/.github/workflows/*@refs/tags/workflows-v1
    GitLab CI (gitlab.com)
      OIDC Discovery URL : https://gitlab.com
      Issuer             : https://gitlab.com
      Audiences          : <ci.infisical.domain>          # must equal id_tokens aud in .gitlab-ci.yml
      Subject  dev-uuid  : project_path:<group>/<project>:ref_type:branch:ref:*
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
