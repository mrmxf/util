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

  clog install <tool>            install the tool for the current platform
  clog install <tool> --dry-run  resolve version + URL without downloading
  clog install <tool> --recipe   print the recipe YAML and exit
  clog install have <tool>       check if the tool is already installed
  clog install list              list all available tools

Recipes can be overridden by setting recipepath: in your clog.yaml to a path
on the local filesystem (bare paths check the embedded recipe store first).`

const haveHelp = `check whether a tool is installed by running its check.try command.

Exit 0 = tool is present. Non-zero = tool is absent or command failed.`

const listHelp = `list all tools that have recipes in the embedded install manifest.`
