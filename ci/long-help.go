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

See 'clog ci resolve --help' and 'clog ci env --help' for details.`

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
