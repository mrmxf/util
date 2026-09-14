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

// Command is the `clog Install` cobra command.
var Command = &cobra.Command{
	Use:           "Install",
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
	Command.Flags().StringVar(&useFlag, "use", "", "install this version instead of the recipe's: a version number, latest or lts")
	Command.Flags().StringVar(&atFlag, "at", "", "alias for --use")
	Command.PersistentFlags().BoolVar(&Verbose, "verbose", false, "log the resolved configuration and its source")
	Command.AddCommand(haveCmd, listCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	sourcePath, entry, err := loadEntry(toolName)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	if recipeFlag {
		return printRecipe(toolName, sourcePath, entry)
	}

	platform, err := resolvePlatform()
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	dryRun, dryRunSource := isDryRun()
	if dryRun {
		vlog("dry-run enabled", "source", dryRunSource)
	}

	if err := Run(entry, toolName, platform, dryRun); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func runHave(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	_, entry, err := loadEntry(toolName)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	if err := RunCheck(toolName, entry.Check); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

// loadEntry finds the recipe for toolName. A kfg override (install.<tool> in
// the user's clog.yaml) wins; otherwise the embedded manifest is used. The
// returned sourcePath says where the recipe came from.
func loadEntry(toolName string) (sourcePath string, entry ToolEntry, err error) {
	key := "install." + toolName
	found, sourcePath, entry, err := kfgOverride(toolName)
	if err != nil {
		return "", ToolEntry{}, err
	}
	if found {
		vlog("view the full recipe with: clog Install " + toolName + " --recipe")
		vlog("recipe source", "tool", toolName, "source", sourcePath, "kfg-key", key)
		return sourcePath, entry, nil
	}
	vlog("no kfg override, using embedded manifest", "tool", toolName, "kfg-key", key)

	m, err := LoadManifest()
	if err != nil {
		return "", ToolEntry{}, err
	}
	sourcePath, entry, err = LoadTool(m, toolName, nil)
	if err != nil {
		return "", ToolEntry{}, err
	}
	vlog("view the full recipe with: clog Install " + toolName + " --recipe")
	vlog("recipe source", "tool", toolName, "source", sourcePath, "index", "embedfs:index.yaml")
	return sourcePath, entry, nil
}

// kfgOverride checks whether kfg config contains install.<tool> and, if so,
// parses it as a ToolEntry (full-replace semantics). found is false when no
// override exists; err is non-nil only when an override exists but is invalid.
func kfgOverride(toolName string) (found bool, sourcePath string, entry ToolEntry, err error) {
	if kfg.Raw == nil {
		return false, "", ToolEntry{}, nil
	}
	raw := kfg.Raw.Get("install." + toolName)
	if raw == nil {
		return false, "", ToolEntry{}, nil
	}
	// Round-trip through YAML to convert the kfg interface{} tree to ToolEntry.
	data, err := yaml.Marshal(raw)
	if err != nil {
		return true, "", ToolEntry{}, fmt.Errorf("kfg override marshal for %q: %w", toolName, err)
	}
	if err := yaml.Unmarshal(data, &entry); err != nil {
		return true, "", ToolEntry{}, fmt.Errorf("kfg override parse for %q: %w", toolName, err)
	}
	// Resolve recipepath if the override itself points to a file.
	if entry.RecipePath != "" {
		sp, resolved, resolveErr := ResolveRecipe(toolName, entry, EmbedFS)
		if resolveErr != nil {
			return true, "", ToolEntry{}, resolveErr
		}
		return true, "kfg→" + sp, resolved, nil
	}
	return true, "kfg:install." + toolName, entry, nil
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
func printRecipe(toolName, sourcePath string, entry ToolEntry) error {
	fmt.Fprintf(os.Stdout, "# source: %s\n", sourcePath)
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
		vlog("platform", "platform", platformFlag, "source", "--platform flag")
		return Platform(platformFlag), nil
	}
	p, err := DetectPlatform()
	if err == nil {
		vlog("platform", "platform", p, "source", "auto-detected (GOOS/GOARCH + distro family)")
	}
	return p, err
}
