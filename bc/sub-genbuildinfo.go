//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrmxf/util/buildinfo"
	"github.com/spf13/cobra"
)

var (
	genFormat  string
	genName    string
	genTitle   string
	genFlavour string
)

const genHelp = `Generate build information from the state of the git repository.
releases.yaml is history and is not read.

  clog BC genBuildinfo                       dev build info as JSON
  clog BC genBuildinfo prod                  prod: HEAD must be a clean release tag
  clog BC genBuildinfo prod --format ldflags go build -ldflags "$(...)"
  clog BC genBuildinfo --format version      v0.11.12 | v0.11.12+dev.3.g34103be

Version (from git):
  HEAD on a release tag         v0.11.12            (release tag: vX.Y.Z or X.Y.Z)
  N commits after one           v0.11.12+dev.N.gSHA
  uncommitted changes           ...+dirty / ....dirty
  no release tag reachable      v0.0.0+dev.N.gSHA

prod is refused unless HEAD is exactly on a release tag and the tree is clean;
check out the release first (clog BC git checkout production). The date is the
commit date, so a rebuild of the same commit stamps the same data.

Formats:
  json        the linker data (default)
  ldflags     -X <linker path>='<json>' for go build -ldflags
  version     the version string only
  docker-tag  the version with + replaced by - (OCI tags forbid +)
  env         BUILD_MODE= BUILD_VERSION= BUILD_DOCKER_TAG= BUILD_HASH= BUILD_DATE=`

var genBuildinfoCmd = &cobra.Command{
	Use:           "genBuildinfo [dev|prod]",
	Aliases:       []string{"genbuildinfo", "buildinfo"},
	Short:         "Generate build version info from git (not releases.yaml)",
	Long:          genHelp,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := "dev"
		if len(args) == 1 {
			mode = strings.ToLower(args[0])
		}
		g, err := buildinfo.ReadGitState(".")
		if err != nil {
			return err
		}
		d, err := buildinfo.Generate(g, buildinfo.BuildOptions{
			Mode: mode, Name: genName, Title: genTitle, Flavour: genFlavour, Branch: currentBranch(),
		})
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		switch genFormat {
		case "json", "":
			fmt.Fprintln(out, d.JSON())
		case "ldflags":
			fmt.Fprintln(out, d.Ldflags())
		case "version":
			fmt.Fprintln(out, d.Tag)
		case "docker-tag":
			fmt.Fprintln(out, buildinfo.DockerTag(d.Tag))
		case "env":
			fmt.Fprintf(out, "BUILD_MODE=%s\nBUILD_VERSION=%s\nBUILD_DOCKER_TAG=%s\nBUILD_HASH=%s\nBUILD_DATE=%s\n",
				d.Build, d.Tag, buildinfo.DockerTag(d.Tag), d.Hash, d.Date)
		default:
			return fmt.Errorf("unknown --format %q (json, ldflags, version, docker-tag or env)", genFormat)
		}
		return nil
	},
}

// currentBranch is the checked-out branch, or the CI ref when the checkout is
// detached (CI checks out a commit, not a branch).
func currentBranch() string {
	if b, err := gitNet("branch", "--show-current"); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			return s
		}
	}
	for _, v := range []string{"CI_COMMIT_REF_NAME", "GITHUB_HEAD_REF", "GITHUB_REF_NAME"} {
		if s := os.Getenv(v); s != "" {
			return s
		}
	}
	return ""
}

func init() {
	genBuildinfoCmd.Flags().StringVar(&genFormat, "format", "json", "json | ldflags | version | docker-tag | env")
	genBuildinfoCmd.Flags().StringVar(&genName, "name", "", "command name (default: the built module's name)")
	genBuildinfoCmd.Flags().StringVar(&genTitle, "title", "", "printable name (default: the built module's name)")
	genBuildinfoCmd.Flags().StringVar(&genFlavour, "flavour", "", "build edition shown as +flavour, e.g. plain | mrmxf")
	Command.AddCommand(genBuildinfoCmd)
}
