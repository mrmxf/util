//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package source

import (
	"fmt"

	"github.com/spf13/cobra"
)

// search the children of the command for the next arg
func matchNextArg(cmd *cobra.Command, arg string) (*cobra.Command, error) {
	for _, c := range cmd.Commands() {
		if c.Name() == arg {
			return c, nil
		}
	}
	return nil, fmt.Errorf("command %s not found", arg)
}

func FindSnippet(rootCmd *cobra.Command, args []string) (*cobra.Command, error) {
	c := rootCmd
	var err error
	for _, arg := range args {
		c, err = matchNextArg(c, arg)
		if err != nil {
			return nil, err
		}
	}
	// we found a command at the right depth - return it
	return c, nil
}
