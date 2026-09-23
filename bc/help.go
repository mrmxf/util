//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

const longHelp = `BC (build-control) delivers mrmxf common build functions
Command help is given below. BC's goal is to allow flows like:

clog BC flow --check "pre-build tools" --build "golang deploy"

Versions and production come from git tags, not a file:

  release tag      vX.Y.Z (or X.Y.Z), nothing after it - v1.2.0-rc1 is not one
  production       the newest release tag by date   clog BC git tag prod
  build version    clog BC genBuildinfo --format version
                   v1.2.3 on a release tag, else v1.2.3+dev.N.gSHA
  run mode         clog ci mode (dev|prod) - a failed check aborts only in prod

releases.yaml is optional history (a changelog): BC never decides from it.

  - {version: "v1.2.3", date: 2026-09-19, build: prod, note: "what changed"}

The .clog.yaml file has a number of optional elements:

clog:
  releases-path: "path/to/releases.yaml"   # history, see above
  stash-path:     tmp/BcStash.yaml         # default place for remembering progress data

Use 'clog BC <command> --help' for more information about specific commands.
`

const stashLogLongHelp = `Log a message using slog and simultaneously record it in the stash file.

The stash entry is organized by flow, phase, and step for tracking a build/deployment processes.
the special word STATUS allows an overall summary status to be logged for a flow/phase/step`

const stashLogExample = `
In a hugo build you might have phases check, prep, hugo, post and deploy. These might be logged:
	clog BC stashLog -1 check     -2 tooling   -3 "tooling"  -I "running hugo tooling check"
	clog BC stashLog -1 check     -2 tooling   -3 "tooling"  -S "tooling check ok"
	clog BC stashLog -1 check     -2 tagging   -3 "tagging"  -I "checking git tags"
	clog BC stashLog -1 check     -2 tagging   -3 "tagging"  -S "git tags ok"
	clog BC stashLog -1 build     -2 push      -3 "docker push" -S "pushed image successfully"
	clog BC stashLog -1 build     -2 unit      -3 "test pkg"    -E "tests failed"
Then run 
  clog BC stash has error            # returns non-0 if there are any errors in the stash
	clog BC stash --build  has error   # returns non-0 if there are any errors in build
	`
const stashLongHelp = `get status or extract information from the stash.

The stash entry is organized by flow, phase, and step for tracking a build/deployment processes.
the special word STATUS allows an overall summary status to be logged for a flow/phase/step`

const stashExample = `
  clog BC stash hasError            # returns non-0 if there are any errors in the stash
	clog BC stash --build has error   # returns non-0 if there are any errors in build
	clog BC stash --build get error   # prints the error string for the build phase to stdout
	`
