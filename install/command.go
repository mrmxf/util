//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/mrmxf/util/kfg"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var recipeFlag bool
var dryRunFlag bool
var platformFlag string // override platform detection (for testing)

// Command is the `clog install` cobra command.
var Command = &cobra.Command{
	Use:           "install",
	Short:         "install and verify development tools",
	Long:          longHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	Args:          cobra.MinimumNArgs(1),
	RunE:          runInstall,
}

var haveCmd = &cobra.Command{
	Use:           "have <tool>",
	Short:         "check if a tool is installed (runs its check: spec)",
	Long:          haveHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	Args:          cobra.ExactArgs(1),
	RunE:          runHave,
}

var listCmd = &cobra.Command{
	Use:           "list",
	Short:         "list available tools",
	Long:          listHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runList,
}

func init() {
	Command.Flags().BoolVar(&recipeFlag, "recipe", false, "print the recipe YAML and exit; do not install")
	Command.Flags().BoolVar(&dryRunFlag, "dry-run", false, "resolve version and URL without downloading or installing")
	Command.Flags().StringVar(&platformFlag, "platform", "", "override platform detection (for testing)")
	Command.AddCommand(haveCmd, listCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	// kfg override: if the user has install.<tool> in their clog.yaml, use it
	// as the complete recipe (full-replace semantics, same as recipepath).
	sourcePath, entry, err := kfgOverride(toolName)
	if err != nil {
		// Not an override error — fall through to manifest.
		sourcePath = ""
	}

	if sourcePath == "" {
		// No kfg override: load from embedded manifest.
		m, merr := LoadManifest()
		if merr != nil {
			slog.Error(merr.Error())
			return merr
		}
		indexEntry := m.Tools[toolName]
		sourcePath, entry, err = LoadTool(m, toolName, nil)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
		if recipeFlag {
			return printRecipe(toolName, sourcePath, indexEntry.RecipePath, entry)
		}
	} else if recipeFlag {
		return printRecipe(toolName, sourcePath, "", entry)
	}

	platform, err := resolvePlatform()
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	if err := Run(entry, toolName, platform, dryRunFlag); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func runHave(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	_, entry, err := kfgOverride(toolName)
	if err != nil {
		// No kfg override — fall through to manifest.
		m, merr := LoadManifest()
		if merr != nil {
			slog.Error(merr.Error())
			return merr
		}
		_, entry, err = LoadTool(m, toolName, nil)
		if err != nil {
			slog.Error(err.Error())
			return err
		}
	}

	if err := RunCheck(toolName, entry.Check); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

// kfgOverride checks whether kfg config contains install.<tool> and, if so,
// parses it as a ToolEntry (full-replace semantics). Returns a non-nil error
// when no override exists (caller should fall back to manifest).
func kfgOverride(toolName string) (sourcePath string, entry ToolEntry, err error) {
	raw := kfg.Raw.Get("install." + toolName)
	if raw == nil {
		return "", ToolEntry{}, fmt.Errorf("no kfg override for %q", toolName)
	}
	// Round-trip through YAML to convert the kfg interface{} tree to ToolEntry.
	data, err := yaml.Marshal(raw)
	if err != nil {
		return "", ToolEntry{}, fmt.Errorf("kfg override marshal for %q: %w", toolName, err)
	}
	if err := yaml.Unmarshal(data, &entry); err != nil {
		return "", ToolEntry{}, fmt.Errorf("kfg override parse for %q: %w", toolName, err)
	}
	// Resolve recipepath if the override itself points to a file.
	if entry.RecipePath != "" {
		sp, resolved, resolveErr := ResolveRecipe(toolName, entry, EmbedFS)
		if resolveErr != nil {
			return "", ToolEntry{}, resolveErr
		}
		return "kfg→" + sp, resolved, nil
	}
	return "kfg:install." + toolName, entry, nil
}

func runList(cmd *cobra.Command, args []string) error {
	m, err := LoadManifest()
	if err != nil {
		return err
	}

	names := make([]string, 0, len(m.Tools))
	for name := range m.Tools {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(os.Stdout, "%-22s  %s\n", "TOOL", "DESCRIPTION")
	fmt.Fprintf(os.Stdout, "%-22s  %s\n", "----", "-----------")
	for _, name := range names {
		_, entry, err := LoadTool(m, name, nil)
		desc := entry.Description
		if err != nil {
			desc = "(recipe unavailable: " + err.Error() + ")"
		}
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(os.Stdout, "%-22s  %s\n", name, desc)
	}
	return nil
}

// printRecipe prints the resolved recipe YAML with a source comment header.
func printRecipe(toolName, sourcePath, recipePath string, entry ToolEntry) error {
	if recipePath != "" {
		fmt.Fprintf(os.Stdout, "# source: %s\n", sourcePath)
	} else {
		fmt.Fprintf(os.Stdout, "# source: inline\n")
	}
	out, err := yaml.Marshal(map[string]ToolEntry{toolName: entry})
	if err != nil {
		return fmt.Errorf("recipe marshal: %w", err)
	}
	_, err = os.Stdout.Write(out)
	return err
}

// resolvePlatform returns the effective platform (flag override or auto-detect).
func resolvePlatform() (Platform, error) {
	if platformFlag != "" {
		return Platform(platformFlag), nil
	}
	return DetectPlatform()
}
