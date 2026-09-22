//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// A stack is one named build unit inside a repo: a thing that has its own
// toolchain, its own make steps and its own linter settings. Most repos have
// exactly one and never write the word down. A repo that ships more than one
// kind of artifact from one tree - a site beside two small services, or esp32
// firmware beside the host tool that flashes it - declares several, and the
// verbs take a name.
//
// See ci/SCANNING.md for the design and the decisions behind it.

// Stack types. The type picks the defaults; the name is what a verb takes.
const (
	StackHugo      = "hugo"
	StackGolang    = "golang"
	StackGolangLib = "golang-lib"
	StackContainer = "container"
	StackTinygo    = "tinygo"
)

var knownStackTypes = []string{StackHugo, StackGolang, StackGolangLib, StackContainer, StackTinygo}

// SelectorAll asks a verb to act on every stack. It is also a reserved name.
const SelectorAll = "all"

// reservedStackNames cannot be stack names. `all` would make `clog build all`
// ambiguous; `dev` and `prod` would make `clog build dev` ambiguous, because
// the verbs disambiguate their arguments by value rather than by position.
var reservedStackNames = []string{SelectorAll, ModeDev, ModeProd}

// StackKey is the .clog.yaml key holding the stack list.
const StackKey = ConfigKey + ".stack"

// Stack is one entry of ci.stack. Every field but Name and Type is optional and
// falls back to the type's defaults.
type Stack struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Tools []string `json:"tools"` // what `clog Install` must have already put on PATH
	Watch string   `json:"watch"` // the interactive inner loop
	Chk   []string `json:"chk"`   // check phases, run before make
	Make  []string `json:"make"`  // build phases
}

// stackDefaults is what a type supplies when the stack leaves a field empty.
type stackDefaults struct {
	Tools []string
	Watch string
	Chk   []string
	Make  []string
}

// stackTypeDefaults carries the upfront work so the common repo writes one
// line. `scan` and `lint` are ordinary check phases: that is the whole point,
// because it makes them local commands rather than CI-only workflow steps.
var stackTypeDefaults = map[string]stackDefaults{
	StackHugo: {
		Tools: []string{"hugo", "ko", "trivy"},
		Watch: "hugo server -D",
		Chk:   []string{"pre-build", "lint", "scan"},
		Make:  []string{"hugo", "ko"},
	},
	StackGolang: {
		Tools: []string{"golang", "trivy", "golangci-lint", "staticcheck"},
		Watch: "go run .",
		Chk:   []string{"pre-build", "lint", "test", "scan"},
		Make:  []string{"golang"},
	},
	StackGolangLib: {
		Tools: []string{"golang", "trivy", "golangci-lint", "staticcheck"},
		Watch: "gotestsum --watch",
		Chk:   []string{"pre-build", "lint", "test", "scan"},
		Make:  []string{"golang"},
	},
	StackContainer: {
		Tools: []string{"golang", "ko", "trivy", "golangci-lint", "staticcheck"},
		Watch: "docker compose watch",
		Chk:   []string{"pre-build", "lint", "test", "scan"},
		Make:  []string{"golang", "ko"},
	},
	StackTinygo: {
		Tools: []string{"tinygo", "trivy"},
		Watch: "tinygo flash && tinygo monitor",
		Chk:   []string{"pre-build", "lint", "scan"},
		Make:  []string{"tinygo"},
	},
}

// StackList is ci.stack. It accepts three shapes, because the one-stack repo
// should not pay for the multi-stack one:
//
//	stack: hugo                            one stack named hugo, type hugo
//	stack: [tinygo, golang]                two stacks, each named for its type
//	stack: [{name: bonfire, type: hugo}]   the long form
type StackList []Stack

// UnmarshalJSON accepts a bare string, a list of strings, or a list of objects.
func (sl *StackList) UnmarshalJSON(b []byte) error {
	// stack: hugo
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*sl = StackList{{Name: one, Type: one}}
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(b, &items); err != nil {
		return fmt.Errorf("%s must be a stack name or a list of them, not %s", StackKey, string(b))
	}
	out := make(StackList, 0, len(items))
	for i, raw := range items {
		// stack: [tinygo, golang]
		var name string
		if err := json.Unmarshal(raw, &name); err == nil {
			out = append(out, Stack{Name: name, Type: name})
			continue
		}
		// stack: [{name: …, type: …}]
		type stackAlias Stack // avoid recursing into this method
		var s stackAlias
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("%s[%d] is neither a stack name nor a {name, type} block: %w", StackKey, i, err)
		}
		st := Stack(s)
		// `- name: hugo` alone is as good as the string form.
		if strings.TrimSpace(st.Type) == "" {
			st.Type = st.Name
		}
		out = append(out, st)
	}
	*sl = out
	return nil
}

// resolve fills a stack's empty fields from its type's defaults.
func (s Stack) resolve() Stack {
	d, ok := stackTypeDefaults[s.Type]
	if !ok {
		return s
	}
	if len(s.Tools) == 0 {
		s.Tools = d.Tools
	}
	if strings.TrimSpace(s.Watch) == "" {
		s.Watch = d.Watch
	}
	if len(s.Chk) == 0 {
		s.Chk = d.Chk
	}
	if len(s.Make) == 0 {
		s.Make = d.Make
	}
	return s
}

// validate checks one stack. The reserved-name rule is enforced here, at parse
// time, rather than at the moment somebody's `clog build all` does the wrong
// thing quietly.
func (s Stack) validate(i int) error {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return fmt.Errorf("%s[%d] has no name: every stack needs one, because the verbs take it (clog build <name>)", StackKey, i)
	}
	for _, r := range reservedStackNames {
		if name == r {
			return fmt.Errorf("%s[%d] is named %q, which is reserved: `clog build %s` already means something else - rename the stack",
				StackKey, i, name, name)
		}
	}
	if strings.TrimSpace(s.Type) == "" {
		return fmt.Errorf("%s.%s has no type (one of %s)", StackKey, name, strings.Join(knownStackTypes, ", "))
	}
	for _, k := range knownStackTypes {
		if s.Type == k {
			return nil
		}
	}
	return fmt.Errorf("%s.%s type %q is not one of %s", StackKey, name, s.Type, strings.Join(knownStackTypes, ", "))
}

// Stacks returns every stack in declaration order, resolved and validated.
// Declaration order is meaningful: the first stack is what `clog watch`
// follows.
func Stacks(cfg Config) ([]Stack, error) {
	if len(cfg.Stack) == 0 {
		return nil, fmt.Errorf("%s is not set: name this repo's stack (one of %s) - clog ci --config-help",
			StackKey, strings.Join(knownStackTypes, ", "))
	}
	seen := make(map[string]int, len(cfg.Stack))
	out := make([]Stack, 0, len(cfg.Stack))
	for i, s := range cfg.Stack {
		if err := s.validate(i); err != nil {
			return nil, err
		}
		if prev, dup := seen[s.Name]; dup {
			return nil, fmt.Errorf("%s has two stacks named %q (entries %d and %d): names are how the verbs tell them apart",
				StackKey, s.Name, prev, i)
		}
		seen[s.Name] = i
		out = append(out, s.resolve())
	}
	return out, nil
}

// StackNames lists the names in declaration order.
func StackNames(stacks []Stack) []string {
	names := make([]string, 0, len(stacks))
	for _, s := range stacks {
		names = append(names, s.Name)
	}
	return names
}

// StackSelect resolves a selector for a batch verb - build and deploy.
//
//	""      every stack, because the rigorous thing is what you get for free
//	"all"   the same, said out loud
//	name    only that stack
//
// Narrowing is fine. Narrowing in silence is the fault this guards against, so
// callers pair this with StackEcho.
func StackSelect(cfg Config, selector string) (selected, all []Stack, err error) {
	all, err = Stacks(cfg)
	if err != nil {
		return nil, nil, err
	}
	sel := strings.TrimSpace(selector)
	if sel == "" || sel == SelectorAll {
		return all, all, nil
	}
	for _, s := range all {
		if s.Name == sel {
			return []Stack{s}, all, nil
		}
	}
	return nil, nil, unknownStack(sel, all)
}

// StackWatch resolves a selector for the one foreground verb. An empty selector
// is the first stack, because a watch command is an interactive process and
// there is no sensible way to run several in one terminal. `all` is refused
// rather than defaulted, for the same reason.
func StackWatch(cfg Config, selector string) (chosen Stack, all []Stack, err error) {
	all, err = Stacks(cfg)
	if err != nil {
		return Stack{}, nil, err
	}
	sel := strings.TrimSpace(selector)
	switch sel {
	case SelectorAll:
		return Stack{}, nil, fmt.Errorf("`clog watch %s` is refused: a watch command is an interactive foreground process, so pick one of %s",
			SelectorAll, strings.Join(StackNames(all), ", "))
	case "":
		return all[0], all, nil
	}
	for _, s := range all {
		if s.Name == sel {
			return s, all, nil
		}
	}
	return Stack{}, nil, unknownStack(sel, all)
}

// unknownStack names what the repo does have, so the fix is in the error.
func unknownStack(sel string, all []Stack) error {
	return fmt.Errorf("no %s named %q (have: %s)", StackKey, sel, strings.Join(StackNames(all), ", "))
}

// StackIgnored lists, in declaration order, the stacks a verb left out.
func StackIgnored(selected, all []Stack) []string {
	keep := make(map[string]bool, len(selected))
	for _, s := range selected {
		keep[s.Name] = true
	}
	var out []string
	for _, s := range all {
		if !keep[s.Name] {
			out = append(out, s.Name)
		}
	}
	return out
}

// StackEcho renders what a verb resolved to, naming the stacks rather than
// echoing the word "all" - which doubles as the cheapest possible check that
// the config says what somebody thought it said. When a verb acts on fewer
// stacks than the repo has, it says which it left out:
//
//	building bonfire, form-contact, form-parking (prod)
//	building bonfire (dev), ignoring [form-contact, form-parking]
//	watching bonfire, ignoring [form-contact, form-parking]
//
// Only the unnarrowed default stays quiet about omissions, because it omits
// nothing. mode may be "" for verbs that have none.
func StackEcho(verb string, selected, all []Stack, mode string) string {
	var b strings.Builder
	b.WriteString(verb)
	b.WriteString(" ")
	b.WriteString(strings.Join(StackNames(selected), ", "))
	if m := strings.TrimSpace(mode); m != "" {
		b.WriteString(" (" + m + ")")
	}
	if ignored := StackIgnored(selected, all); len(ignored) > 0 {
		b.WriteString(", ignoring [" + strings.Join(ignored, ", ") + "]")
	}
	return b.String()
}

// unionStrings concatenates in order and drops repeats, so `pre-build` and
// `scan` run once however many stacks are in play.
func unionStrings(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, v := range list {
			v = strings.TrimSpace(v)
			if v == "" || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// StackTools, StackChk and StackMake union a field across the selected stacks.
func StackTools(stacks []Stack) []string {
	return unionField(stacks, func(s Stack) []string { return s.Tools })
}
func StackChk(stacks []Stack) []string {
	return unionField(stacks, func(s Stack) []string { return s.Chk })
}
func StackMake(stacks []Stack) []string {
	return unionField(stacks, func(s Stack) []string { return s.Make })
}

func unionField(stacks []Stack, pick func(Stack) []string) []string {
	lists := make([][]string, 0, len(stacks))
	for _, s := range stacks {
		lists = append(lists, pick(s))
	}
	return unionStrings(lists...)
}

// StackOf returns the stack a target belongs to. A target that names no stack
// belongs to the first one, so a single-stack repo never learns the field
// exists.
func StackOf(t Target, all []Stack) (Stack, error) {
	name := strings.TrimSpace(t.Stack)
	if name == "" {
		if len(all) == 0 {
			return Stack{}, fmt.Errorf("%s is not set, so a target cannot be bound to a stack", StackKey)
		}
		return all[0], nil
	}
	for _, s := range all {
		if s.Name == name {
			return s, nil
		}
	}
	return Stack{}, fmt.Errorf("target stack %q is not in %s (have: %s)", name, StackKey, strings.Join(StackNames(all), ", "))
}

// TargetsForStacks filters targets down to those belonging to the selected
// stacks, so `clog deploy bonfire` never publishes an artifact this run did not
// build. Returned in sorted name order, matching TargetNames.
func TargetsForStacks(cfg Config, selected, all []Stack) ([]string, error) {
	keep := make(map[string]bool, len(selected))
	for _, s := range selected {
		keep[s.Name] = true
	}
	var names []string
	for _, name := range sortedKeys(cfg.Targets) {
		owner, err := StackOf(cfg.Targets[name], all)
		if err != nil {
			return nil, fmt.Errorf("ci.targets.%s: %w", name, err)
		}
		if keep[owner.Name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}
