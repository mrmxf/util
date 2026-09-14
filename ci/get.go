//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getRequiredFlag bool

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "print a non-secret value from the clog config, e.g. ci.artifact",
	Long: `ci get - print one value from the merged clog config (.clog.yaml overlays)

  clog ci get ci.artifact
  echo "ci_artifact=$(clog ci get ci.artifact)" >> "$GITHUB_ENV"

Scalars print as-is; lists print one item per line. A missing key prints nothing
and exits 0, so optional settings compose in scripts; add --required to fail.
Secrets are not in the config - see clog ci run.`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val, err := Get(args[0], getRequiredFlag)
		if err != nil {
			return err
		}
		if val != "" {
			fmt.Fprintln(cmd.OutOrStdout(), val)
		}
		return nil
	},
}

// Get renders a config value for a shell. Missing values return "" unless
// required is set.
func Get(key string, required bool) (string, error) {
	v := ConfigValue(key)
	if isEmptyValue(v) {
		if required {
			return "", fmt.Errorf("%s is not set in the clog config (.clog.yaml)", key)
		}
		return "", nil
	}
	return renderValue(v), nil
}
