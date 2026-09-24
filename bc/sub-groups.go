//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"github.com/mrmxf/util/retire"
	"github.com/spf13/cobra"
)

// The verb groups that give BC its v1.0.0 grammar.
//
//	clog BC <namespace> <verb> <noun>
//
// The verb is always the first word after the namespace, and it says what you
// get back: `get` prints one value, `is`/`has` return only an exit code. Before
// v1.0.0 the noun sat in the verb slot - `clog BC git tag prod` - so nothing in
// the command told you whether it printed, tested or acted. The nouns are
// unchanged; they have simply moved one word to the right, behind a verb.
//
// All the re-parenting lives here rather than in each leaf's init(), so the
// shape of the tree can be read in one place instead of reconstructed from
// forty files.
var (
	// gitGetCmd - `clog BC git get <branch|suffix>`
	gitGetCmd = &cobra.Command{
		Use:           "get",
		Short:         "print one value about the working tree",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}

	// tagGetCmd - `clog BC git tag get <head|origin|prod|ref>`
	tagGetCmd = &cobra.Command{
		Use:           "get",
		Short:         "print one tag",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}

	// hashGetCmd - `clog BC git hash get <head|origin|prod|ref>`
	hashGetCmd = &cobra.Command{
		Use:           "get",
		Short:         "print one commit hash",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}

	// releasesGetCmd - `clog BC releases get <version|date|flow|note|build|path>`
	releasesGetCmd = &cobra.Command{
		Use:           "get",
		Short:         "print one field of the newest release",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}

	// bcGetCmd - `clog BC get <linkerpath>`
	bcGetCmd = &cobra.Command{
		Use:           "get",
		Short:         "print one build-control value",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}

	// bcGenCmd - `clog BC gen <buildinfo>`
	bcGenCmd = &cobra.Command{
		Use:           "gen",
		Short:         "generate build metadata",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           helpRun,
		Args:          cobra.NoArgs,
	}
)

// helpRun is what a grouping command does: it has no work of its own.
func helpRun(cmd *cobra.Command, args []string) {
	cmd.Help() //nolint:errcheck // help output is best-effort
}

func init() {
	gitCmd.AddCommand(gitGetCmd)
	gitGetCmd.AddCommand(branchCmd, suffixCmd)

	tagCmd.AddCommand(tagGetCmd)
	tagGetCmd.AddCommand(headCmd, originCmd, prodTagCmd, refCmd)

	hashCmd.AddCommand(hashGetCmd)
	hashGetCmd.AddCommand(hashHeadCmd, hashOriginCmd, productionHashCmd, refHashCmd)

	releasesCmd.AddCommand(releasesGetCmd)
	releasesGetCmd.AddCommand(releaseVersionCmd, releaseDateCmd, releaseFlowCmd,
		releaseNoteCmd, releaseBuildCmd, releaseYamlCmd)

	Command.AddCommand(bcGetCmd)
	bcGetCmd.AddCommand(linkerPathCmd)

	Command.AddCommand(bcGenCmd)
	bcGenCmd.AddCommand(genBuildinfoCmd)

	// `is` belonged to releases all along: it compares a field of the newest
	// release. At the top level `clog BC is main prod` read as a question about
	// BC itself.
	releasesCmd.AddCommand(isCmd)

	// stashLog was the only camelCase leaf in the tree, and it is a stash
	// operation, so it belongs under stash like `has` and `get`.
	stashCmd.AddCommand(stashLogCmd)
}

// semverGroupCmd keeps `semver` as the namespace it reads like, with the
// question it asks moved into the verb slot: `clog BC semver satisfies a b`
// says what it does, where a bare `clog BC semver a b` looked like a printer
// and was in fact a silent predicate.
var semverGroupCmd = &cobra.Command{
	Use:           "semver",
	Short:         "semantic version comparisons",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run:           helpRun,
	Args:          cobra.NoArgs,
}

func init() {
	Command.AddCommand(semverGroupCmd)
	semverGroupCmd.AddCommand(semverCmd)
	// Without this the old form printed help and exited 0: a version check
	// written `try: clog BC semver a b` passed on every version.
	retire.Namespace(semverGroupCmd, "clog BC semver satisfies <needs> <have>",
		"a comparison is a question, so the verb names the question")
}
