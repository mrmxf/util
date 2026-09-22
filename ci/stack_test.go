//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"strings"
	"testing"
)

// bonfireConfig is the worked multi-stack example: a site first, so it is what
// `clog watch` follows, beside two small services released on their own.
func bonfireConfig() Config {
	return Config{Stack: StackList{
		{Name: "bonfire", Type: StackHugo},
		{Name: "form-contact", Type: StackGolang},
		{Name: "form-parking", Type: StackGolang},
	}}
}

func TestStackListAcceptsThreeShapes(t *testing.T) {
	for _, tc := range []struct {
		name, raw, want string
	}{
		{"bare string", `"hugo"`, "hugo:hugo"},
		{"list of names", `["tinygo","golang"]`, "tinygo:tinygo golang:golang"},
		{"long form", `[{"name":"bonfire","type":"hugo"}]`, "bonfire:hugo"},
		{"name only", `[{"name":"hugo"}]`, "hugo:hugo"},
	} {
		var sl StackList
		if err := json.Unmarshal([]byte(tc.raw), &sl); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		var got []string
		for _, s := range sl {
			got = append(got, s.Name+":"+s.Type)
		}
		if strings.Join(got, " ") != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, strings.Join(got, " "), tc.want)
		}
	}
}

func TestStackDefaultsComeFromTheType(t *testing.T) {
	stacks, err := Stacks(Config{Stack: StackList{{Name: "site", Type: StackHugo}}})
	if err != nil {
		t.Fatal(err)
	}
	s := stacks[0]
	if strings.Join(s.Chk, " ") != "pre-build lint scan" {
		t.Errorf("chk = %v, want the hugo default", s.Chk)
	}
	if strings.Join(s.Make, " ") != "hugo ko" {
		t.Errorf("make = %v, want the hugo default", s.Make)
	}
	if s.Watch != "hugo server -D" {
		t.Errorf("watch = %q, want the hugo default", s.Watch)
	}
	// An explicit field wins over the type default.
	over, err := Stacks(Config{Stack: StackList{{Name: "site", Type: StackHugo, Chk: []string{"lint"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(over[0].Chk, " ") != "lint" {
		t.Errorf("an explicit chk should win, got %v", over[0].Chk)
	}
}

// Reserved names are rejected at parse time rather than at the moment somebody's
// `clog build all` quietly does the wrong thing.
func TestReservedStackNamesAreRejected(t *testing.T) {
	for _, name := range []string{"all", "dev", "prod"} {
		_, err := Stacks(Config{Stack: StackList{{Name: name, Type: StackHugo}}})
		if err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("a stack named %q should be refused, got %v", name, err)
		}
		if err != nil && !strings.Contains(err.Error(), name) {
			t.Errorf("the error for %q should name the offending stack, got %v", name, err)
		}
	}
}

func TestStackValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		want string
	}{
		{"no stack at all", Config{}, "is not set"},
		{"unknown type", Config{Stack: StackList{{Name: "x", Type: "rust"}}}, "not one of"},
		{"no name", Config{Stack: StackList{{Type: StackHugo}}}, "has no name"},
		{"duplicate names", Config{Stack: StackList{
			{Name: "a", Type: StackHugo}, {Name: "a", Type: StackGolang},
		}}, "two stacks named"},
	} {
		if _, err := Stacks(tc.cfg); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want an error containing %q, got %v", tc.name, tc.want, err)
		}
	}
}

// build and deploy take everything by default: the rigorous thing is what you
// get for free, and adding a stack must never silently shrink what CI builds.
func TestStackSelectDefaultsToEveryStack(t *testing.T) {
	cfg := bonfireConfig()
	for _, sel := range []string{"", SelectorAll} {
		selected, all, err := StackSelect(cfg, sel)
		if err != nil {
			t.Fatal(err)
		}
		if len(selected) != 3 || len(all) != 3 {
			t.Errorf("selector %q gave %d of %d stacks, want all three", sel, len(selected), len(all))
		}
		if len(StackIgnored(selected, all)) != 0 {
			t.Errorf("selector %q should ignore nothing", sel)
		}
	}

	selected, all, err := StackSelect(cfg, "bonfire")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].Name != "bonfire" {
		t.Errorf("selected = %v, want just bonfire", StackNames(selected))
	}
	if got := strings.Join(StackIgnored(selected, all), " "); got != "form-contact form-parking" {
		t.Errorf("ignored = %q, want the other two in declaration order", got)
	}

	if _, _, err := StackSelect(cfg, "nope"); err == nil || !strings.Contains(err.Error(), "have: bonfire") {
		t.Errorf("an unknown stack should list what the repo does have, got %v", err)
	}
}

// watch is the one foreground verb: it takes the first stack, and refuses `all`.
func TestStackWatchTakesTheFirstAndRefusesAll(t *testing.T) {
	cfg := bonfireConfig()
	s, _, err := StackWatch(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "bonfire" {
		t.Errorf("watch default = %q, want the first stack", s.Name)
	}
	if _, _, err := StackWatch(cfg, SelectorAll); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("`watch all` should be refused, got %v", err)
	}
}

// Narrowing is fine. Narrowing in silence is the fault.
func TestStackEchoNamesWhatItIgnored(t *testing.T) {
	cfg := bonfireConfig()
	all, err := Stacks(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := StackEcho("building", all, all, ModeProd); got != "building bonfire, form-contact, form-parking (prod)" {
		t.Errorf("unnarrowed echo = %q", got)
	}
	one := []Stack{all[0]}
	if got := StackEcho("building", one, all, ModeDev); got != "building bonfire (dev), ignoring [form-contact, form-parking]" {
		t.Errorf("narrowed echo = %q", got)
	}
	if got := StackEcho("watching", one, all, ""); got != "watching bonfire, ignoring [form-contact, form-parking]" {
		t.Errorf("watch echo = %q", got)
	}
}

// A phase named by two stacks runs once.
func TestStackUnionDedupes(t *testing.T) {
	cfg := bonfireConfig()
	all, err := Stacks(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(StackChk(all), " "); got != "pre-build lint scan test" {
		t.Errorf("chk union = %q, want declaration order with no repeats", got)
	}
	if got := strings.Join(StackMake(all), " "); got != "hugo ko golang" {
		t.Errorf("make union = %q", got)
	}
	if got := strings.Join(StackTools(all), " "); got != "hugo ko trivy golang golangci-lint staticcheck" {
		t.Errorf("tools union = %q", got)
	}
}

// A target with no stack: belongs to the first one, so a single-stack repo
// never learns the field exists.
func TestTargetsBindToStacks(t *testing.T) {
	cfg := bonfireConfig()
	cfg.Targets = map[string]Target{
		"site":    {Kind: KindGitHubPages, Prod: map[string]any{"dir": "kodata"}},
		"parking": {Kind: KindRegistry, Stack: "form-parking", Prod: map[string]any{"image": "acme/p"}},
	}
	all, err := Stacks(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := TargetsForStacks(cfg, []Stack{all[0]}, all)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "site" {
		t.Errorf("bonfire's targets = %v, want the unbound one", got)
	}
	got, err = TargetsForStacks(cfg, all, all)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "parking site" {
		t.Errorf("all targets = %v", got)
	}

	cfg.Targets["orphan"] = Target{Kind: KindRegistry, Stack: "ghost"}
	if _, err := TargetsForStacks(cfg, all, all); err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("a target naming an unknown stack should fail by name, got %v", err)
	}
}
