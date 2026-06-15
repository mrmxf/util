//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// readFile is a tiny helper returning trimmed file contents (empty if absent).
func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func TestRunBlock_ThenElse(t *testing.T) {
	dir := t.TempDir()
	thenF := filepath.Join(dir, "then")
	elseF := filepath.Join(dir, "else")
	finF := filepath.Join(dir, "fin")

	t.Run("all truthy runs then + finally", func(t *testing.T) {
		b := Block{
			Name:    "ok block",
			Try:     []Condition{{Expr: "true", State: StateTruthy}},
			Then:    "echo then > " + thenF,
			Else:    "echo else > " + elseF,
			Finally: "echo fin > " + finF,
		}
		res := RunBlockCtx(BlockContext{Section: "t", ID: 1, Count: 1}, b)
		if !res.Truthy {
			t.Fatal("expected truthy")
		}
		if readFile(t, thenF) != "then" {
			t.Fatal("then did not run")
		}
		if readFile(t, elseF) != "" {
			t.Fatal("else should not have run")
		}
		if readFile(t, finF) != "fin" {
			t.Fatal("finally did not run")
		}
	})

	os.Remove(thenF)

	t.Run("a falsy condition runs else not then", func(t *testing.T) {
		b := Block{
			Name: "fail block",
			Try:  []Condition{{Expr: "false", State: StateTruthy}},
			Then: "echo then > " + thenF,
			Else: "echo else > " + elseF,
		}
		res := RunBlockCtx(BlockContext{Section: "t", ID: 1, Count: 1}, b)
		if res.Truthy {
			t.Fatal("expected falsy")
		}
		if readFile(t, thenF) != "" {
			t.Fatal("then should not have run")
		}
		if readFile(t, elseF) != "else" {
			t.Fatal("else did not run")
		}
	})
}

func TestRunBlock_CheckVars(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "vars")

	// two try conditions: first truthy, second falsy → falsy=2, truthy=1
	b := Block{
		Name: "var block",
		Try: []Condition{
			{Expr: "true", State: StateTruthy},
			{Expr: "false", State: StateTruthy},
		},
		Finally: "printf '%s|%s|%s|%s|%s\\n' " +
			"\"$CHECK_NAME\" \"$CHECK_FALSY_CONDITIONS\" \"$CHECK_TRUTHY_CONDITIONS\" " +
			"\"$CHECK_ISTRUTHY\" \"$CHECK_ISFALSY\" > " + out,
	}
	RunBlockCtx(BlockContext{Section: "tools", ID: 1, Count: 1}, b)

	got := readFile(t, out)
	fields := strings.Split(got, "|")
	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %q", got)
	}
	if fields[0] != "var block" {
		t.Errorf("CHECK_NAME=%q", fields[0])
	}
	if fields[1] != "2" {
		t.Errorf("CHECK_FALSY_CONDITIONS=%q want 2", fields[1])
	}
	if fields[2] != "1" {
		t.Errorf("CHECK_TRUTHY_CONDITIONS=%q want 1", fields[2])
	}
	// any falsy → ISTRUTHY="1", ISFALSY=""
	if fields[3] != "1" {
		t.Errorf("CHECK_ISTRUTHY=%q want 1", fields[3])
	}
	if fields[4] != "" {
		t.Errorf("CHECK_ISFALSY=%q want empty", fields[4])
	}
}

func TestRunBlock_EnvGatesTry(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "try-ran")

	// env condition fails → try must be skipped (marker file never created).
	b := Block{
		Name: "gated",
		Env:  []Condition{{Expr: "CHK_DEFINITELY_UNSET", State: StateTruthy}},
		Try:  []Condition{{Expr: "touch " + marker, State: StateTruthy}},
	}
	res := RunBlockCtx(BlockContext{Section: "t", ID: 1, Count: 1}, b)
	if res.Truthy {
		t.Fatal("expected falsy (env failed)")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("try condition ran despite env gate failing")
	}
}

func TestBlock_Aliases(t *testing.T) {
	t.Run("ok parses into Then, catch into Else", func(t *testing.T) {
		var b Block
		yml := "name: x\ntry: \"true\"\nok: echo good\ncatch: echo bad\n"
		if err := yaml.Unmarshal([]byte(yml), &b); err != nil {
			t.Fatal(err)
		}
		if b.Then != "echo good" {
			t.Errorf("Then=%q", b.Then)
		}
		if b.Else != "echo bad" {
			t.Errorf("Else=%q", b.Else)
		}
		// legacy scalar try normalised to one truthy condition
		if len(b.Try) != 1 || b.Try[0].State != StateTruthy {
			t.Errorf("Try=%+v", b.Try)
		}
	})

	t.Run("both then and ok is an error", func(t *testing.T) {
		var b Block
		yml := "then: a\nok: b\n"
		if err := yaml.Unmarshal([]byte(yml), &b); err == nil {
			t.Fatal("expected error when both then and ok set")
		}
	})

	t.Run("array try with state specs", func(t *testing.T) {
		var b Block
		yml := "try:\n  - \"go version\": truthy\n  - \"echo hi\": \"~hi\"\n"
		if err := yaml.Unmarshal([]byte(yml), &b); err != nil {
			t.Fatal(err)
		}
		if len(b.Try) != 2 {
			t.Fatalf("expected 2 try conditions, got %d", len(b.Try))
		}
		if b.Try[1].Expr != "echo hi" || b.Try[1].State != "~hi" {
			t.Errorf("Try[1]=%+v", b.Try[1])
		}
	})
}
