//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mrmxf/util/scripts"
)

// BlockResult captures the outcome of running one block.
type BlockResult struct {
	Truthy      bool // all evaluated conditions passed
	ThenExit    int  // exit code of the then/ok script (0 if none ran)
	ElseExit    int  // exit code of the else/catch script (0 if none ran)
	FinallyExit int  // exit code of the finally script (0 if none ran)
}

// RunBlock runs a single standalone block (section "", index 1 of 1) and
// returns an error when the block result is falsy — i.e. a condition failed.
// This is the entry point used by callers such as `clog install have <tool>`.
func RunBlock(b Block) error {
	res := RunBlockCtx(BlockContext{Section: "check", ID: 1, Count: 1}, b)
	if !res.Truthy {
		return fmt.Errorf("check %q failed", b.Name)
	}
	return nil
}

// BlockContext carries the identity of a block within its section so the
// $CHECK_* identity vars can be populated.
type BlockContext struct {
	Section string
	Before  string // section/group-level before, spliced ahead of block.Before
	ID      int    // 1-based index within the section
	Count   int    // total blocks in the section
}

// RunBlockCtx evaluates a block's conditions and runs then/else/finally with
// the full $CHECK_* environment populated. env conditions are evaluated first
// (pure Go); if all pass, try conditions are evaluated (shell or clog dispatch).
func RunBlockCtx(ctx BlockContext, b Block) BlockResult {
	before := strings.TrimSpace(strings.Join([]string{ctx.Before, b.Before}, "\n"))

	// identity vars — set once, never mutated.
	env := map[string]string{
		"CHECK_NAME":    b.Name,
		"CHECK_ID":      strconv.Itoa(ctx.ID),
		"CHECK_COUNT":   strconv.Itoa(ctx.Count),
		"CHECK_SECTION": ctx.Section,
	}

	var (
		conditionCount int
		truthyIdx      []string
		falsyIdx       []string
		allTruthy      = true
		lastValue      = ""
		// legacy try vars (last try condition)
		lastStdout = ""
		lastExit   = 0
	)

	record := func(passed bool, value string) {
		conditionCount++
		idx := strconv.Itoa(conditionCount)
		if passed {
			truthyIdx = append(truthyIdx, idx)
		} else {
			falsyIdx = append(falsyIdx, idx)
			allTruthy = false
		}
		if value != "" {
			lastValue = value
		}
		env["CHECK_CONDITION_COUNT"] = strconv.Itoa(conditionCount)
		env["CHECK_TRUTHY_CONDITIONS"] = strings.Join(truthyIdx, ",")
		env["CHECK_FALSY_CONDITIONS"] = strings.Join(falsyIdx, ",")
		if value != "" {
			env["CHECK_CONDITION_VALUE"] = value
		} else if passed {
			env["CHECK_CONDITION_VALUE"] = "pass"
		} else {
			env["CHECK_CONDITION_VALUE"] = "fail"
		}
	}

	// 1. env conditions (cheap, no shell). Evaluate all for full accounting.
	envOK := true
	for _, c := range b.Env {
		passed, val := evalEnv(c)
		record(passed, val)
		if !passed {
			envOK = false
		}
	}

	// 2. try conditions — skipped when any env condition failed (fail fast,
	//    no shell spawns), per decision D11.
	if envOK {
		for _, c := range b.Try {
			passed, stdout, exit := evalTry(c, before, env)
			lastStdout, lastExit = stdout, exit
			record(passed, stdout)
		}
	}

	// 3. overall result vars (empty-means-active convention).
	if allTruthy {
		env["CHECK_ISTRUTHY"] = ""
		env["CHECK_ISFALSY"] = "1"
	} else {
		env["CHECK_ISTRUTHY"] = "1"
		env["CHECK_ISFALSY"] = ""
	}

	// 4. legacy vars for the last try condition.
	env["STDOUTERR"] = lastStdout
	env["EXITCODE"] = strconv.Itoa(lastExit)
	_ = lastValue

	res := BlockResult{Truthy: allTruthy}

	// 5. then / else.
	if allTruthy {
		if b.Then != "" {
			res.ThenExit = runScript(before, b.Then, env)
		}
	} else {
		if b.Else != "" {
			res.ElseExit = runScript(before, b.Else, env)
		}
	}

	// 6. finally — always runs.
	if b.Finally != "" {
		res.FinallyExit = runScript(before, b.Finally, env)
	}

	return res
}

// runScript streams a then/else/finally script with the $CHECK_* environment.
func runScript(before, script string, env map[string]string) int {
	cmd := script
	if before != "" {
		cmd = before + "\n" + script
	}
	exit, err := scripts.AwaitShellSnippet(cmd, env, []string{})
	if err != nil {
		slog.Debug("check script error", "err", err)
	}
	return exit
}

// RunSection runs every block in a parsed section, preserving the legacy
// semantics: the section "fails" only when an else/catch script exits non-zero.
func RunSection(s Section) error {
	fail := 0
	for i, b := range s.Blocks {
		ctx := BlockContext{Section: s.Name, Before: s.Before, ID: i + 1, Count: len(s.Blocks)}
		res := RunBlockCtx(ctx, b)
		if !res.Truthy && res.ElseExit > 0 {
			fail++
		}
	}
	name := s.Name
	if name == "" {
		name = "check"
	}
	if fail == 0 {
		slog.Info(fmt.Sprintf("Check %s passed (%d blocks)", name, len(s.Blocks)))
		return nil
	}
	err := fmt.Errorf("check %s failed (%d/%d blocks errored)", name, fail, len(s.Blocks))
	slog.Error(err.Error())
	return err
}
