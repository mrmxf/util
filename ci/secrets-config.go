//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mrmxf/util/kfg"
)

// ConfigKey is the .clog.yaml section holding CI configuration. Everything in
// it is non-secret; secrets come from Infisical at run time.
const ConfigKey = "ci"

// Config is the non-secret CI configuration read from .clog.yaml. See
// `clog CI --config-help` for the contract.
type Config struct {
	Infisical InfisicalConfig        `json:"infisical"`
	Require   map[string]Requirement `json:"require"`
	Policy    Policy                 `json:"policy"`
	Targets   map[string]Target      `json:"targets"` // deploy destinations (D-I.12)
	Stack     StackList              `json:"stack"`   // named build units (ci/SCANNING.md)
	Scan      ScanConfig             `json:"scan"`    // source + artifact sweeps
}

// InfisicalConfig says where a repo's secrets live and which machine identity
// each CI platform logs in as.
type InfisicalConfig struct {
	Domain      string `json:"domain"`       // instance URL, e.g. https://eu.infisical.com
	ProjectID   string `json:"project-id"`   // project (workspace) id
	ProjectSlug string `json:"project-slug"` // informational; the API uses the id
	Path        string `json:"path"`         // secret folder, e.g. /clog-mrmxf
	Audience    string `json:"audience"`     // OIDC audience; defaults to Domain
	// Env maps a mode (dev|prod) to an Infisical env slug.
	Env map[string]string `json:"env"`
	// Identity maps platform (github|gitlab) → {dev-uuid, prod-uuid}.
	Identity map[string]map[string]string `json:"identity"`
}

// Requirement lists what a verb (build, deploy…) needs before it starts.
type Requirement struct {
	Env      []string `json:"env"`      // env vars that must be set and non-empty
	Optional []string `json:"optional"` // env vars that only warn when missing
	Config   []string `json:"config"`   // .clog.yaml keys that must be non-empty
}

// LoadConfig is an overridable hook returning the CI config: the default reads
// the merged clog config, tests inject fixtures.
var LoadConfig = func() (Config, error) {
	var cfg Config
	if kfg.Raw == nil {
		return cfg, nil
	}
	raw := kfg.Raw.Get(ConfigKey)
	if raw == nil {
		return cfg, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return cfg, fmt.Errorf("%s: cannot read config: %w", ConfigKey, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: config does not match the expected shape (clog CI --config-help): %w", ConfigKey, err)
	}
	return cfg, nil
}

// ConfigValue is an overridable hook returning a .clog.yaml value by dotted key.
var ConfigValue = func(key string) any {
	if kfg.Raw == nil {
		return nil
	}
	return kfg.Raw.Get(key)
}

// validate checks the fields a secret fetch needs, naming each missing key.
// isUnset reports that the repo declares no secret store at all. Not every
// repo has secrets - a docs site deploying to GitHub Pages needs only the
// token Actions already provides - and such a repo should not be forced to
// invent an Infisical project. A PARTIALLY filled block is still an error:
// a typo must not silently turn secrets off.
func (c InfisicalConfig) isUnset() bool {
	return strings.TrimSpace(c.Domain) == "" &&
		strings.TrimSpace(c.ProjectID) == "" &&
		strings.TrimSpace(c.Path) == "" &&
		len(c.Env) == 0
}

func (c InfisicalConfig) validate() error {
	var missing []string
	for key, val := range map[string]string{
		"domain": c.Domain, "project-id": c.ProjectID, "path": c.Path,
	} {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, "ci.infisical."+key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing .clog.yaml keys %s (clog CI --config-help)", strings.Join(missing, ", "))
	}
	return nil
}

// infisicalEnv maps a mode to its Infisical env slug.
func (c InfisicalConfig) infisicalEnv(mode string) (string, error) {
	slug := strings.TrimSpace(c.Env[mode])
	if slug == "" {
		return "", fmt.Errorf("ci.infisical.env.%s is not set: map mode %q to an Infisical env slug (clog CI --config-help)", mode, mode)
	}
	return slug, nil
}

// identityFor picks the machine identity for a platform and mode: prod uses
// prod-uuid, dev uses dev-uuid.
func (c InfisicalConfig) identityFor(platform Platform, mode string) (key, uuid string, err error) {
	key = "dev-uuid"
	if mode == ModeProd {
		key = "prod-uuid"
	}
	uuid = strings.TrimSpace(c.Identity[string(platform)][key])
	if uuid == "" {
		return key, "", fmt.Errorf("ci.infisical.identity.%s.%s is not set: paste the machine identity UUID (clog CI --config-help)", platform, key)
	}
	return key, uuid, nil
}

// audience is the OIDC audience clog requests and Infisical should bind.
func (c InfisicalConfig) audience() string {
	if a := strings.TrimSpace(c.Audience); a != "" {
		return a
	}
	return strings.TrimRight(c.Domain, "/")
}
