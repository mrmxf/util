//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package check creates a try-catch-finally block of scripts

package check

const longHelp = `
usage: clog Check thingy

this will search for a "check.thingy" key in your clog.yaml config file.

A check block has the format given below. The name field is optional. The script
in the try section is executed. If the exit code is non-zero then the catch script
is executed, otherwise the ok script is executed.

The clog command will ONLY return an error if one or more of the catch blocks
exits with a non-zero exit code, otherwise the command will be a success.

For consistent checks:
 - use clog Log -E "message"   when a catch block returns error
 - use clog Log -W "message"   when a catch block returns success
 - the output of the try command is available in ok/catch as $STDOUTERR
 - the exit status of the try command is available in ok/catch as $EXITCODE

Sample clog.yaml
================

check:
  thingy:
    before: eval "$(clog Crayon)"
    blocks:
      - try:     clog git tree clean
        ok:      clog Log -I "Ok working tree clean"
        catch:   clog Log -W "   working tree NOT clean"
				finally: echo "git tree cleanliness complete"
      - name: golang
        try: |
          vv="$(go version|cat go.mod|grep '^go '|grep -oE '[0-9]\.[0-9]+\.[0-9]+')" 
          [[ "$vv" == "$(clog project needs golang)" ]]
        catch: |
          clog Log -E "wrong go version. Need $(clog project needs golang)"
          exit 1 # ensure clog Check returns an error that can be caught
`
