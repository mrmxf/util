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
  clog Install <tool> --recipe   print the recipe YAML and exit
  clog Install have <tool>       check if the tool is already installed
  clog Install list              list all available tools

Recipes can be overridden by setting recipepath: in your clog.yaml to a path
on the local filesystem (bare paths check the embedded recipe store first).`

const haveHelp = `check whether a tool is installed by running its check.try command.

Exit 0 = tool is present. Non-zero = tool is absent or command failed.`

const listHelp = `list all tools that have recipes in the embedded install manifest.`
