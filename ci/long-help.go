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

See 'clog ci resolve --help' for details.`

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
