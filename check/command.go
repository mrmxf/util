//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package check creates a try-catch-finally block of scripts

package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/mrmxf/util/kfg"
	"github.com/mrmxf/util/scripts"
	"github.com/mrmxf/util/shell"
	"github.com/spf13/cobra"
)

var KfgKey = "check"

// define the try-catch-finally block keys:
type CheckBlock struct {
	Try              string `json:"try"`
	TryStdOutErr     string
	TryExitCode      int
	Ok               string `json:"ok"`
	Catch            string `json:"catch"`
	CatchStdOutErr   string
	Finally          string `json:"finally"`
	FinallyStdOutErr string
	FinallyExitCode  int
}

// validRequiredKeys is a reference map to check if the keys in the config
// are weird or valid. None of the keys are currently required
var validRequiredKeys = map[string]bool{
	"Try":     false,
	"Pass":    false,
	"Catch":   false,
	"Finally": false,
}

// a Check Group is a collection of Check Blocks, potentially with a log level
type CheckGroup struct {
	Name     string
	LogLevel slog.Level
	LogFile  *os.File
	Before   string `json:"before"`
	Blocks   []CheckBlock
}

var Command = &cobra.Command{
	Use:           "Check",
	Short:         "run all blocks in a check group defined in config",
	Long:          longHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {

		// check a group was specified
		if len(args) == 0 {
			msg := "cannot run Check, no group specified"
			slog.Error(msg, "len(args)", 0)
			cmd.Help()
		}

		if kfg.Raw.Get(KfgKey) == nil {
			slog.Error("cannot find check key in clog.yaml", "key", KfgKey)
			return fmt.Errorf("cannot find check key(%s) in clog.yaml", KfgKey)
		}

		// check which group we are checking
		groupKey := KfgKey + "." + args[0]
		if kfg.Raw.Get(groupKey) == nil {
			slog.Error("cannot find check group in clog.yaml", "key", groupKey)
			return fmt.Errorf("cannot find check group(%s) in clog.yaml", groupKey)
		}

		// parse the check key into a CheckGroup struct
		group := CheckGroup{
			Name:     args[0],
			LogLevel: slog.LevelInfo,
			LogFile:  nil,
			Blocks:   []CheckBlock{},
		}
		blocksKey := groupKey + ".blocks"
		err := parseBlocks(cmd, KfgKey, &group, kfg.Raw.Get(blocksKey))
		if err != nil {
			slog.Error("cannot parse check blocks", "key", blocksKey)
			return fmt.Errorf("cannot parse check blocks(%s)", blocksKey)
		}
		// set the group level keys
		group.Name = kfg.Raw.String(groupKey + ".name")
		if len(group.Name) == 0 {
			group.Name = groupKey
		}
		err = runBlocks(cmd, groupKey, group)
		if err != nil {
			return fmt.Errorf("check failed")
		}
		return nil
	},
}

// splice the before commands in front of the try commands
func splice(before string, stepStr string) string {
	cmdStr := ""
	if len(before) > 0 {
		cmdStr = before + "\n"
	}
	return cmdStr + stepStr
}

// exec a command with custom environment
func capture(before string, stepStr string, i int, stepName string, env map[string]string) (string, int, error) {
	cmdStr := splice(before, stepStr)
	outErr, exitCode, err := shell.CaptureShellSnippet(cmdStr, env)
	if err != nil {
		slog.Debug(fmt.Sprintf("            - %d (%s) failed", i, stepName), "err", err)
	}
	return outErr, exitCode, err
}

// stream a command with custom environment
func stream(before string, stepStr string, i int, stepName string, env map[string]string) (int, error) {
	cmdStr := splice(before, stepStr)
	exitStatus, err := scripts.AwaitShellSnippet(cmdStr, env, []string{})
	return exitStatus, err
}

// run all the blocks in the group
func runBlocks(cmd *cobra.Command, key string, group CheckGroup) error {
	fail := 0
	var env map[string]string
	var err error
	for i, b := range group.Blocks {
		//step 1: try
		if len(b.Try) > 0 {
			b.TryStdOutErr, b.TryExitCode, err = capture(group.Before, b.Try, i, "try", nil)

			//preserve the output of try for the next steps
			env = map[string]string{
				"STDOUTERR": b.TryStdOutErr,
				"EXITCODE":  fmt.Sprintf("%d", b.TryExitCode),
			}
			if err != nil {
				env["ERR"] = err.Error()
			}

			//step 2. ok or catch
			if b.TryExitCode == 0 {
				if len(b.Ok) > 0 {
					//step 2. ok command exists
					stream(group.Before, b.Ok, i, "ok", env)
				}
			} else {
				if len(b.Catch) > 0 {
					//step 2. catch exists
					exit, _ := stream(group.Before, b.Catch, i, "catch", env)
					// fail is only incremented if a catch returns an error
					if exit > 0 {
						fail++
					}
				}
			}
		}
		//step 3. finally
		if len(b.Finally) > 0 {
			//step 3. finally exists
			stream(group.Before, b.Finally, i, "catch", env)
		}
	}
	if fail == 0 {
		slog.Info(fmt.Sprintf("Check %s passed (%d blocks)", group.Name, len(group.Blocks)))
		return nil
	}
	msg := fmt.Errorf("check %s failed (%d/%d blocks errored)", group.Name, fail, len(group.Blocks))
	slog.Error(msg.Error())
	return msg
}

func validateRawBlockKeys(key string, iBlk int, block map[string]interface{}) (*CheckBlock, bool) {
	errCount := 0
	newBlock := CheckBlock{}
	// check all the keys from clog.yaml against reference keys
	for k := range block {
		if _, isValid := validRequiredKeys[k]; isValid {
			errCount++
			slog.Warn((fmt.Sprintf("%s block #%d has foreign key (%s)", key, iBlk, k)))
		}
	}
	// use json library to populated the struct via a json string
	jsonBody, err := json.Marshal(block)
	if err != nil {
		errCount++
		slog.Warn((fmt.Sprintf("%s block #%d cannot be parsed", key, iBlk)))
	}
	if err := json.Unmarshal(jsonBody, &newBlock); err != nil {
		errCount++
		slog.Warn("%s block #%d cannot be unmarshaled", key, iBlk)
		slog.Warn("      yaml vs.bash quotes? e.g. - try: \"[ -n \\\"$VAR\\\" ]\"")
	}
	if errCount == 0 {
		return &newBlock, true
	}
	return nil, false
}

// load the blocks into the CheckBlock struct
// typical block structure is:
//   - name: check origin hash
//     before: clog Log -I "? HEAD hash == origin hash"
//     try: [[ "$(clog tag hash head)" == "$(clog tag hash origin)" ]]
//     catch: clog Log -W "  HEAD hash != origin hash"
//     finally
func parseBlocks(parentCmd *cobra.Command, key string, group *CheckGroup, rawBlocksArray any) error {
	slog.Debug((fmt.Sprintf("%s raw blocks of type %T", key, rawBlocksArray)))
	allBlocks := []CheckBlock{}
	// validate homogeneity of rawBlocksArray map[string]any
	ok := true
	for i, block := range rawBlocksArray.([]any) {
		switch block.(type) {
		case map[string]interface{}:
			b, parseOk := validateRawBlockKeys(key, i, block.(map[string]interface{}))
			ok = ok && parseOk
			if parseOk {
				allBlocks = append(allBlocks, *b)
			}
		default:
			slog.Error((fmt.Sprintf("%s block #%d must be array of keys, not (%T)", key, i, block)))
			ok = false
		}
	}
	if !ok {
		return errors.New("invalid syntax in config " + key)
	}
	group.Blocks = allBlocks
	slog.Debug(fmt.Sprintf("%s - %d checks blocks found", key, len(allBlocks)))
	return nil
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
