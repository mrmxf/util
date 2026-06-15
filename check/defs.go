//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package check runs YAML-defined check blocks: a set of conditions (env + try)
// evaluated as a logical AND, followed by then/else and finally scripts.
//
// Phase 8 schema (current):
//
//	check:
//	  section-name:
//	    blocks:
//	      - name: "human label"
//	        env:
//	          - SOME_VAR: truthy        # set and non-empty
//	          - OTHER_VAR: falsy        # unset or empty
//	          - EXACT_VAR: "value"      # exact match
//	          - RE_VAR:    "~v[0-9]+"   # regex (prefix ~)
//	        try:
//	          - "go version >/dev/null": truthy   # exit 0
//	          - "echo $HOME":            "~/.*"   # stdout regex
//	        then: 'echo all good'       # alias: ok
//	        else: 'echo something fell over'  # alias: catch
//	        finally: 'echo always runs'
//
// The legacy schema (string `try:`, group-level `before:`, `$STDOUTERR` /
// `$EXITCODE` / `$ERR` vars) remains supported so existing configs keep working:
// a scalar `try: "cmd"` is normalised to a single `{cmd: truthy}` condition.
package check

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// state spec keywords (the value side of an env/try condition)
const (
	StateTruthy = "truthy" // env: set & non-empty; try: exit code 0
	StateFalsy  = "falsy"  // env: unset/empty;     try: non-zero exit code
	// "~PATTERN" → regexp match; ">N"/"<N" → numeric; anything else → exact match
)

// Condition is a single env or try check: an expression (env var name or shell
// command) paired with the state spec it must satisfy.
type Condition struct {
	Expr  string // env var name (env conditions) or shell command (try conditions)
	State string // state spec: "truthy", "falsy", "~regex", ">N", "<N", or an exact literal
}

// Block is one check block: a set of env + try conditions AND-ed together,
// with then/else/finally scripts.
type Block struct {
	Name    string      `yaml:"name"`
	Env     []Condition `yaml:"env"`     // pure-Go env checks, evaluated first
	Try     []Condition `yaml:"try"`     // shell (or clog-internal) checks
	Then    string      `yaml:"then"`    // runs when ALL conditions truthy; alias of ok
	Ok      string      `yaml:"ok"`      // legacy alias of then
	Else    string      `yaml:"else"`    // runs when ANY condition falsy; alias of catch
	Catch   string      `yaml:"catch"`   // legacy alias of else
	Finally string      `yaml:"finally"` // always runs
	Before  string      `yaml:"before"`  // legacy: spliced before each script in this block
}

// Section is a named collection of blocks (the value of a check.<section> key).
type Section struct {
	Name   string  `yaml:"name"`
	Before string  `yaml:"before"` // legacy: spliced before every block's scripts
	Blocks []Block `yaml:"blocks"`
}

// UnmarshalYAML lets a block accept either the new array-of-maps form or the
// legacy scalar form for env/try, and merges the then/ok and else/catch aliases.
func (b *Block) UnmarshalYAML(node *yaml.Node) error {
	// raw mirror struct: capture env/try as generic nodes so we can normalise.
	var raw struct {
		Name    string    `yaml:"name"`
		Env     yaml.Node `yaml:"env"`
		Try     yaml.Node `yaml:"try"`
		Then    string    `yaml:"then"`
		Ok      string    `yaml:"ok"`
		Else    string    `yaml:"else"`
		Catch   string    `yaml:"catch"`
		Finally string    `yaml:"finally"`
		Before  string    `yaml:"before"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	envConds, err := decodeConditions(&raw.Env)
	if err != nil {
		return fmt.Errorf("env: %w", err)
	}
	tryConds, err := decodeConditions(&raw.Try)
	if err != nil {
		return fmt.Errorf("try: %w", err)
	}

	// merge then/ok and else/catch aliases — error if both forms are set.
	then := raw.Then
	if raw.Ok != "" {
		if then != "" {
			return fmt.Errorf("block %q: both 'then' and 'ok' set (they are aliases)", raw.Name)
		}
		then = raw.Ok
	}
	els := raw.Else
	if raw.Catch != "" {
		if els != "" {
			return fmt.Errorf("block %q: both 'else' and 'catch' set (they are aliases)", raw.Name)
		}
		els = raw.Catch
	}

	b.Name = raw.Name
	b.Env = envConds
	b.Try = tryConds
	b.Then = then
	b.Else = els
	b.Finally = raw.Finally
	b.Before = raw.Before
	return nil
}

// decodeConditions normalises a YAML node into []Condition. It accepts:
//   - a scalar string  → one {string: truthy} condition (legacy try)
//   - a sequence of single-key maps → one condition per entry
//   - an empty/absent node → nil
func decodeConditions(node *yaml.Node) ([]Condition, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Value == "" {
			return nil, nil
		}
		// legacy scalar try: run the command, exit 0 = truthy
		return []Condition{{Expr: node.Value, State: StateTruthy}}, nil
	case yaml.SequenceNode:
		conds := make([]Condition, 0, len(node.Content))
		for _, item := range node.Content {
			switch item.Kind {
			case yaml.ScalarNode:
				// bare command in a list → truthy
				conds = append(conds, Condition{Expr: item.Value, State: StateTruthy})
			case yaml.MappingNode:
				if len(item.Content) != 2 {
					return nil, fmt.Errorf("condition map must have exactly one key:value pair")
				}
				key := item.Content[0]
				val := item.Content[1]
				conds = append(conds, Condition{Expr: key.Value, State: scalarString(val)})
			default:
				return nil, fmt.Errorf("condition must be a string or single-key map, got kind %d", item.Kind)
			}
		}
		return conds, nil
	default:
		return nil, fmt.Errorf("expected scalar or sequence, got kind %d", node.Kind)
	}
}

// scalarString renders a scalar yaml node's value as a string. Keywords like
// truthy/falsy, regex (~...), numerics (>N) and literals all arrive as strings.
func scalarString(n *yaml.Node) string {
	return n.Value
}
