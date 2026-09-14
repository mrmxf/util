//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrmxf/util/kfg"
	"github.com/spf13/cobra"
)

// Event is the policy vocabulary for what triggered a run.
type Event string

const (
	EventBranch   Event = "branch"   // push to a branch
	EventTag      Event = "tag"      // push of a tag
	EventDispatch Event = "dispatch" // manual / api trigger
	EventSchedule Event = "schedule" // scheduled run
	EventPR       Event = "pr"       // pull / merge request
	EventLocal    Event = "local"    // laptop, no CI
)

// Policy is ci.policy in .clog.yaml: which events build, and which refs
// deploy to each clog environment (stage, prod).
type Policy struct {
	Build  []Event               `json:"build"`
	Deploy map[string]DeployRule `json:"deploy"`
	// Actors, when set, limits builds and deploys to runs triggered by these
	// accounts (GitHub login / GitLab username), matched case-insensitively.
	Actors []string `json:"actors"`
}

// DeployRule allows a deploy to one environment. A run deploys when its ref
// matches Branches or Tags (or it is a scheduled run and Schedule is set), and
// the top releases.yaml entry's build equals ReleasesYAML when that is set.
type DeployRule struct {
	Branches     stringList `json:"branches"`      // globs, * matches anything incl "/"
	Tags         stringList `json:"tags"`          // globs
	Schedule     bool       `json:"schedule"`      // scheduled runs may deploy
	ReleasesYAML string     `json:"releases-yaml"` // e.g. prod: top releases.yaml build must be prod
}

// Decision is what ci.policy says about the current run.
type Decision struct {
	Env          string `json:"env"`
	Event        Event  `json:"event"`
	Ref          string `json:"ref"`
	Actor        string `json:"actor"`
	IsTag        bool   `json:"is_tag"`
	Build        bool   `json:"build"`
	BuildReason  string `json:"build_reason"`
	Deploy       bool   `json:"deploy"`
	DeployReason string `json:"deploy_reason"`
}

// ReleaseBuild is an overridable hook returning the top releases.yaml entry's
// build value (dev|prod), or "" when releases are not loaded.
var ReleaseBuild = func() string {
	if r := kfg.CurrentRelease(); r != nil {
		return r.Build
	}
	return ""
}

// stringList accepts either a YAML string or a list of strings.
type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*s = stringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return fmt.Errorf("want a string or a list of strings: %w", err)
	}
	*s = many
	return nil
}

// tagNamer is implemented by git resolvers that can name the tag at HEAD.
type tagNamer interface {
	ExactTag() string
}

// ExactTag names the tag HEAD points at, or "".
func (g execGit) ExactTag() string {
	return g.git("describe", "--exact-match", "--tags", "HEAD")
}

// Decide evaluates ci.policy for the current run. Pull requests never deploy,
// whatever the policy says; a missing policy builds but never deploys.
func Decide(env Env) (Decision, error) {
	r, err := Resolve(env)
	if err != nil {
		return Decision{}, err
	}
	clogEnv, err := ResolveEnvName(env)
	if err != nil {
		return Decision{}, err
	}
	cfg, err := LoadConfig()
	if err != nil {
		return Decision{}, err
	}
	pol := cfg.Policy

	d := Decision{Env: clogEnv, Event: eventOf(r), Ref: refName(env, r), Actor: r.Actor, IsTag: r.IsTag}
	if r.CI == PlatformLocal {
		d = previewLocal(env, r, d)
	}

	switch {
	case !actorAllowed(pol.Actors, r):
		d.BuildReason = fmt.Sprintf("actor %q is not in ci.policy.actors %v", r.Actor, pol.Actors)
	case len(pol.Build) == 0:
		d.Build, d.BuildReason = true, "no ci.policy.build list: every event builds"
	case containsEvent(pol.Build, d.Event):
		d.Build, d.BuildReason = true, fmt.Sprintf("event %s is in ci.policy.build", d.Event)
	default:
		d.BuildReason = fmt.Sprintf("event %s is not in ci.policy.build %v", d.Event, pol.Build)
	}

	d.Deploy, d.DeployReason = decideDeploy(pol, d)
	if !d.Build && d.Deploy {
		d.Deploy, d.DeployReason = false, "not building, so nothing to deploy ("+d.BuildReason+")"
	}
	return d, nil
}

func decideDeploy(pol Policy, d Decision) (bool, string) {
	if d.Event == EventPR {
		return false, "pull/merge requests never deploy"
	}
	rule, ok := pol.Deploy[d.Env]
	if !ok {
		return false, fmt.Sprintf("no ci.policy.deploy.%s rule", d.Env)
	}

	matched := ""
	switch {
	case d.Event == EventTag:
		if glob := matchAny(rule.Tags, d.Ref); glob != "" {
			matched = fmt.Sprintf("tag %s matches %q", d.Ref, glob)
		}
	case d.Event == EventSchedule:
		if rule.Schedule {
			matched = "scheduled runs allowed (schedule: true)"
		}
	default: // branch, dispatch, local
		if glob := matchAny(rule.Branches, d.Ref); glob != "" {
			matched = fmt.Sprintf("branch %s matches %q", d.Ref, glob)
		}
	}
	if matched == "" {
		return false, fmt.Sprintf("%s %q is not allowed by ci.policy.deploy.%s", d.Event, d.Ref, d.Env)
	}

	if want := rule.ReleasesYAML; want != "" {
		if got := ReleaseBuild(); got != want {
			return false, fmt.Sprintf("%s, but releases.yaml build is %q, want %q", matched, got, want)
		}
		matched += fmt.Sprintf(" and releases.yaml build is %s", want)
	}
	return true, fmt.Sprintf("%s → deploy %s", matched, d.Env)
}

func eventOf(r Resolution) Event {
	switch r.Verb {
	case VerbPR:
		return EventPR
	case VerbSchedule:
		return EventSchedule
	case VerbDispatch:
		return EventDispatch
	case VerbLocal:
		return EventLocal
	}
	if r.IsTag {
		return EventTag
	}
	return EventBranch
}

// refName returns the bare branch or tag name the policy matches against.
func refName(env Env, r Resolution) string {
	ref := strings.TrimPrefix(r.Ref, "refs/heads/")
	return strings.TrimPrefix(ref, "refs/tags/")
}

// previewLocal lets a laptop preview a CI decision. Without CLOG_ENV a laptop
// run stays event "local" (env dev: never deploys). With CLOG_ENV set it is
// judged as the push it imitates: a tag push when CLOG_ENV=prod and HEAD is
// exactly on a tag, otherwise a push of the checked-out branch.
func previewLocal(env Env, r Resolution, d Decision) Decision {
	if strings.TrimSpace(env.Getenv(EnvOverrideVar)) == "" {
		return d
	}
	if d.Env == EnvProd && r.IsTag {
		if tn, ok := env.Git.(tagNamer); ok {
			if tag := tn.ExactTag(); tag != "" {
				d.Event, d.Ref, d.IsTag = EventTag, tag, true
				return d
			}
		}
	}
	d.Event, d.IsTag = EventBranch, false
	return d
}

// actorAllowed applies ci.policy.actors in CI; laptops are always allowed.
func actorAllowed(actors []string, r Resolution) bool {
	if len(actors) == 0 || r.CI == PlatformLocal {
		return true
	}
	for _, a := range actors {
		if strings.EqualFold(strings.TrimSpace(a), r.Actor) {
			return true
		}
	}
	return false
}

func containsEvent(list []Event, e Event) bool {
	for _, x := range list {
		if x == e {
			return true
		}
	}
	return false
}

// matchAny returns the first glob matching s, or "". In a glob, * matches any
// run of characters including "/" (as Infisical subjects do) and ? one character.
func matchAny(globs []string, s string) string {
	for _, g := range globs {
		if globRegexp(g).MatchString(s) {
			return g
		}
	}
	return ""
}

func globRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// EnvLines renders a decision as KEY=value lines for $GITHUB_ENV or dotenv.
func (d Decision) EnvLines() string {
	return "" +
		"clog_env=" + d.Env + "\n" +
		"do_build=" + strconv.FormatBool(d.Build) + "\n" +
		"do_deploy=" + strconv.FormatBool(d.Deploy) + "\n"
}

var policyFormatFlag string

var policyCmd = &cobra.Command{
	Use:          "policy",
	Short:        "print what ci.policy decides for this run: build? deploy? why?",
	Long:         policyHelp,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := Decide(DefaultEnv())
		if err != nil {
			return err
		}
		return writeDecision(cmd.OutOrStdout(), d, policyFormatFlag)
	},
}

var shouldCmd = &cobra.Command{
	Use:          "should <build|deploy>",
	Short:        "exit 0 if ci.policy allows build/deploy for this run, 1 if not",
	Long:         policyHelp,
	SilenceUsage: true,
	// the "no" answer is an exit code, not an error worth printing
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	ValidArgs:     []string{"build", "deploy"},
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := Decide(DefaultEnv())
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
			return err
		}
		yes, reason, err := d.allows(args[0])
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%s %s: %s\n", args[0], yesNo(yes), reason)
		if !yes {
			exit(1)
		}
		return nil
	},
}

func (d Decision) allows(what string) (bool, string, error) {
	switch what {
	case "build":
		return d.Build, d.BuildReason, nil
	case "deploy":
		return d.Deploy, d.DeployReason, nil
	default:
		return false, "", fmt.Errorf("`ci should` takes build or deploy, not %q", what)
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func writeDecision(w io.Writer, d Decision, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(d)
	case "env":
		_, err := fmt.Fprint(w, d.EnvLines())
		return err
	default:
		return fmt.Errorf("unknown --format %q (want json or env)", format)
	}
}
