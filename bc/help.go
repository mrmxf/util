//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

const longHelp = `BC (build-control) delivers mrmxf common build functions
Command help is given below. BC's goal is to allow flows like:

clog BC flow --check "pre-build tools" --build "golang deploy"

There must be a json/yaml file that track the releases you're trying to build: form:

 # flow:"stage",  build:"dev"  = auto-add to staging server
 # flow:"main",   build:"prod" = to flag as a production release
 # golang projects MUST have tags must start with a "v", other project NO letter v
- {version: "v0.10.0", date: 2025-10-02, flow: main, build: dev, note: refactor - kfg system}

The .clog.yaml file has a number of optional elements:

clog:
  releases-path: "path/to/releases.yaml"   # see above for format
	stash-path:     tmp/BcStash.yaml         # default place for remembering progress data

Use 'clog bc <command> --help' for more information about specific commands.
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
