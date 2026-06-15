//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package check creates a try/then/else/finally block of scripts

package check

const longHelp = `
usage: clog Check <section>

Runs every block in the "check.<section>" key of your clog.yaml.

Each block ANDs together its env: and try: conditions. The value beside a
condition is the state it must satisfy:

  truthy        env var set & non-empty        / try cmd exits 0
  falsy         env var unset or empty         / try cmd exits non-zero
  "literal"     exact match (env value or trimmed stdout)
  "~regexp"     Go regexp match
  ">N" / "<N"   numeric comparison

env: conditions are pure Go (no shell) and are evaluated first; if any fail the
try: conditions are skipped. then: runs when ALL conditions are truthy; else:
runs when ANY is falsy. ok:/catch: are accepted aliases of then:/else:.
finally: always runs.

clog Check returns an error only when an else:/catch: script exits non-zero.

These $CHECK_* vars are available to then:/else:/finally::
  $CHECK_NAME $CHECK_ID $CHECK_COUNT $CHECK_SECTION        (identity)
  $CHECK_CONDITION_COUNT $CHECK_TRUTHY_CONDITIONS          (accounting)
  $CHECK_FALSY_CONDITIONS $CHECK_CONDITION_VALUE
  $CHECK_ISTRUTHY $CHECK_ISFALSY    (empty = active; opposite states)
Legacy $STDOUTERR / $EXITCODE (last try condition) are still set.

Flags:
  --dry-run                 validate YAML structure; run nothing
  --dump <section>.<id>     print a block (1-based id); run nothing
  --dump ... --conditions 3,4   explain only those 1-based conditions

Sample clog.yaml
================

check:
  tools:
    blocks:
      - name: golang
        env:
          - CI: falsy
        try:
          - "go version >/dev/null 2>&1": truthy
        then:    clog Log -I "Ok golang installed"
        else:    clog Log -W "   golang missing"
        finally: clog Log -I "$CHECK_NAME ($CHECK_ID/$CHECK_COUNT)"

Legacy form (still supported): a scalar try: with ok:/catch: strings.
      - try:   clog git tree clean
        ok:    clog Log -I "Ok working tree clean"
        catch: clog Log -W "   working tree NOT clean"
`
