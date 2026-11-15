//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

const longHelp = `install and manage development tools from declarative recipes.

Each tool has a recipe that describes how to check, version-resolve, download,
and install it across six platforms:

  darwin/arm64  darwin/amd64
  linux-deb/arm64  linux-deb/amd64   (Debian / Ubuntu and derivatives)
  linux-rpm/arm64  linux-rpm/amd64   (RHEL / Fedora / Rocky and derivatives)

Usage:

  clog Install <tool>            install the tool for the current platform
  clog Install <tool> --dry-run  resolve version + URL without downloading
                                 (the global clog --dryrun does the same)
  clog Install <tool> --verbose  log the resolved recipe and where each part came from
  clog Install <tool> --use <v>  install version <v> instead of the recipe's (alias --at)
                                 <v> is a version (1.26.4, v1.26.4, go1.26.4), latest or lts
                                 latest ignores go.mod / module.yaml and takes the newest release
                                 lts is the newest patch of the previous minor line (e.g. 1.26.x
                                 while 1.27 is current); package-manager installs accept only latest
  clog Install clog              install the clog release .clog-version pins (alias: clog upgrade)
                                 add --pin to write the version --use chose into .clog-version
  clog Install <tool> --recipe   print the recipe YAML and exit
  clog Install have <tool>       check if the tool is already installed
  clog Install list              list all available tools

Recipes can be overridden by setting recipepath: in your clog.yaml to a path
on the local filesystem (bare paths check the embedded recipe store first).

A recipe's download may name a sha256sum-format checksums file:
  download: {strategy: url-template, url: ..., checksums: <url of checksums.txt>}
The asset must be listed with a matching SHA-256, or nothing is installed.
Recipes marked (sha256-verified) in clog Install list do this.`

const haveHelp = `check whether a tool is installed by running its check.try command.

Exit 0 = tool is present. Non-zero = tool is absent or command failed.`

const listHelp = `list all tools that have recipes in the embedded install manifest.`

const upgradeHelp = `install the clog release this repo pins, replacing the running clog.

clog upgrade is clog Install clog, and both follow get-clog.sh - the script
every CI job bootstraps with - so a laptop and CI resolve the same clog:

  version   --use / --at, else $CLOG_VERSION, else the first line of
            $CLOG_VERSION_FILE (default .clog-version) that is not a comment.
            No pin is an error, never a silent "latest".
  repo      $CLOG_REPO, else install.clog.install.repo in .clog.yaml, else the
            repo this clog was released from, else mrmxf/clog. Changing repo
            without CLOG_REPO is refused: it would swap the edition.
  asset     clog-<amd|arm>-<lnx|mac>, verified against the release's
            checksums.txt ($CLOG_SKIP_VERIFY=1 skips; not advised) and its SLSA
            provenance when slsa-verifier is on PATH.
  where     $CLOG_INSTALL_DIR/clog, else the running clog, replaced in place
            once the new binary has been verified and has run.

  clog upgrade                       match .clog-version
  clog upgrade --use latest          newest release; warns that CI still pins the old one
  clog upgrade --use latest --pin    newest release, and write it to .clog-version
  clog upgrade --use v1.3.5          that release
  clog upgrade --dry-run             print the plan (version= repo= asset= url= dir=)

Private releases: $CLOG_TOKEN, else $GITHUB_TOKEN / $GH_TOKEN / $GHAT.`
