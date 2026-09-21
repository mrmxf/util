//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrmxf/util/buildinfo"
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
// deploy in each mode (dev, prod).
type Policy struct {
	Build  []Event               `json:"build"`
	Deploy map[string]DeployRule `json:"deploy"`
	// Actors, when set, limits builds and deploys to runs triggered by these
	// accounts (GitHub login / GitLab username), matched case-insensitively.
	Actors []string `json:"actors"`
}

// DeployRule allows a deploy in one mode. A run deploys when its ref
// matches Branches or Tags, or it is a scheduled run and Schedule is set.
// ReleasesYAML is deprecated and ignored: releases.yaml is history.
type DeployRule struct {
	Branches     stringList `json:"branches"`      // globs, * matches anything incl "/"
	Tags         stringList `json:"tags"`          // globs
	Schedule     bool       `json:"schedule"`      // scheduled runs may deploy
	Dispatch     bool       `json:"dispatch"`      // manual runs may deploy
	ReleasesYAML string     `json:"releases-yaml"` // DEPRECATED, ignored (warns): prod comes from the release tag
}

// Decision is what ci.policy says about the current run. Build mode and deploy
// mode are the same value (D-I.13a), so there is one Mode.
type Decision struct {
	Mode         string   `json:"mode"` // dev | prod
	ModeReason   string   `json:"mode_reason"`
	Event        Event    `json:"event"`
	Ref          string   `json:"ref"`
	Actor        string   `json:"actor"`
	IsTag        bool     `json:"is_tag"`
	Build        bool     `json:"build"`
	BuildReason  string   `json:"build_reason"`
	Deploy       bool     `json:"deploy"`
	DeployReason string   `json:"deploy_reason"`
	Targets      []string `json:"targets"` // deploy targets for this mode
}

// ReleaseVersion is an overridable hook returning this checkout's version from
// git, used by the {tag} / {version} target tokens: the release tag at HEAD
// (v0.11.12), else the dev version (v0.11.12+dev.3.g34103be) - never the last
// release, so a dev deploy cannot overwrite a release's files. releases.yaml is
// history and is not read.
var ReleaseVersion = func() string {
	g, err := buildinfo.ReadGitState(".")
	if err != nil {
		return ""
	}
	return g.Version()
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

// Decide evaluates ci.policy for the current run: which mode it is in, whether
// it builds, whether it deploys, and to which targets. Pull requests never
// deploy, whatever the policy says; a missing policy builds but never deploys.
func Decide(env Env) (Decision, error) {
	r, err := Resolve(env)
	if err != nil {
		return Decision{}, err
	}
	cfg, err := LoadConfig()
	if err != nil {
		return Decision{}, err
	}
	pol := cfg.Policy
	warnDeprecated(pol)

	d := Decision{Event: eventOf(r), Ref: refName(r), Actor: r.Actor, IsTag: r.IsTag}
	forced, err := modeOverride(env)
	if err != nil {
		return Decision{}, err
	}
	if r.CI == PlatformLocal {
		d = previewLocal(env, r, d, forced)
	}
	d.Mode, d.ModeReason = decideMode(pol, d, forced)

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
	if d.Deploy {
		if d.Targets, err = TargetNames(cfg, d.Mode, ""); err != nil {
			return Decision{}, err
		}
		if len(d.Targets) == 0 {
			d.Deploy, d.DeployReason = false, fmt.Sprintf("%s, but no ci.targets deploys in %s mode", d.DeployReason, d.Mode)
		}
	}
	return d, nil
}

// decideMode picks dev or prod (D-I.12): a tag push or a scheduled run is prod
// when ci.policy.deploy.prod would accept it (tag glob + a vX.Y.Z release tag,
// or schedule: true);
// everything else - branches, dispatch, pull requests, laptops - is dev.
func decideMode(pol Policy, d Decision, forced string) (string, string) {
	if forced != "" {
		return forced, "forced by $" + ModeOverrideVar
	}
	switch d.Event {
	case EventTag, EventSchedule:
		if ok, why := prodRuleAccepts(pol, d); ok {
			return ModeProd, why
		} else {
			return ModeDev, "dev: " + why
		}
	default:
		return ModeDev, fmt.Sprintf("%s events are always dev", d.Event)
	}
}

// prodRuleAccepts reports whether ci.policy.deploy.prod would accept this run,
// which is what makes it a production build.
func prodRuleAccepts(pol Policy, d Decision) (bool, string) {
	rule, ok := pol.Deploy[ModeProd]
	if !ok {
		return false, "no ci.policy.deploy.prod rule"
	}
	switch d.Event {
	case EventTag:
		glob := matchAny(rule.Tags, d.Ref)
		if glob == "" {
			return false, fmt.Sprintf("tag %s matches no ci.policy.deploy.prod.tags %v", d.Ref, []string(rule.Tags))
		}
		if !buildinfo.IsReleaseTag(d.Ref) {
			return false, fmt.Sprintf("tag %s matches %q but is not a release tag (vX.Y.Z)", d.Ref, glob)
		}
		return true, fmt.Sprintf("tag %s matches %q and is a release tag", d.Ref, glob)
	case EventSchedule:
		if !rule.Schedule {
			return false, "scheduled runs are not allowed by ci.policy.deploy.prod (schedule: true)"
		}
		return true, "scheduled production rebuild (of the newest release tag on the default branch)"
	default:
		return false, string(d.Event) + " events never reach prod mode"
	}
}

func decideDeploy(pol Policy, d Decision) (bool, string) {
	if d.Event == EventPR {
		return false, "pull/merge requests never deploy"
	}
	rule, ok := pol.Deploy[d.Mode]
	if !ok {
		return false, fmt.Sprintf("no ci.policy.deploy.%s rule", d.Mode)
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
	// A manual run's ref is the branch it was launched from, so it can never
	// match a tag glob - even when the job then checks out the release tag.
	// dispatch: true says "a human asking for this deploy is authorisation
	// enough", which is what makes "republish the current release" one click.
	case d.Event == EventDispatch && rule.Dispatch:
		matched = "manual run allowed (dispatch: true)"
	default: // branch, dispatch, local
		if glob := matchAny(rule.Branches, d.Ref); glob != "" {
			matched = fmt.Sprintf("branch %s matches %q", d.Ref, glob)
		}
	}
	if matched == "" {
		return false, fmt.Sprintf("%s %q is not allowed by ci.policy.deploy.%s", d.Event, d.Ref, d.Mode)
	}

	return true, fmt.Sprintf("%s → deploy %s", matched, d.Mode)
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
func refName(r Resolution) string {
	ref := strings.TrimPrefix(r.Ref, "refs/heads/")
	return strings.TrimPrefix(ref, "refs/tags/")
}

// previewLocal lets a laptop preview a CI decision. Without $CLOG_MODE a laptop
// run stays event "local" (dev mode: never deploys). With it set the run is
// judged as the push it imitates: a tag push when CLOG_MODE=prod and HEAD is
// exactly on a tag, otherwise a push of the checked-out branch.
func previewLocal(env Env, r Resolution, d Decision, forced string) Decision {
	if forced == "" {
		return d
	}
	if forced == ModeProd && r.IsTag {
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
// build_mode and deploy_mode are the same value (D-I.13a); deploy_mode is empty
// when this run does not deploy, so a step can gate on it.
func (d Decision) EnvLines() string {
	deployMode := ""
	if d.Deploy {
		deployMode = d.Mode
	}
	return "" +
		"build_mode=" + d.Mode + "\n" +
		"deploy_mode=" + deployMode + "\n" +
		"do_build=" + strconv.FormatBool(d.Build) + "\n" +
		"do_deploy=" + strconv.FormatBool(d.Deploy) + "\n" +
		"deploy_targets=" + strings.Join(d.Targets, " ") + "\n"
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

// warnDeprecated flags ci.policy keys that no longer do anything.
func warnDeprecated(pol Policy) {
	for mode, rule := range pol.Deploy {
		if rule.ReleasesYAML != "" {
			slog.Warn("ignored: releases.yaml is history - prod comes from the release tag (vX.Y.Z); remove the key",
				"key", "ci.policy.deploy."+mode+".releases-yaml")
		}
	}
}
