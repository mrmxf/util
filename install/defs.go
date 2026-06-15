//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import "github.com/mrmxf/util/check"

// Platform identifies a target OS, distribution family, and architecture.
// Six platforms are supported; any tool that doesn't support a platform
// must declare it with Unsupported: true rather than omitting it.
type Platform string

const (
	PlatformDarwinArm64   Platform = "darwin/arm64"
	PlatformDarwinAmd64   Platform = "darwin/amd64"
	PlatformLinuxDebArm64 Platform = "linux-deb/arm64"
	PlatformLinuxDebAmd64 Platform = "linux-deb/amd64"
	PlatformLinuxRpmArm64 Platform = "linux-rpm/arm64"
	PlatformLinuxRpmAmd64 Platform = "linux-rpm/amd64"
)

// AllPlatforms lists every supported platform key in a stable order.
var AllPlatforms = []Platform{
	PlatformDarwinArm64,
	PlatformDarwinAmd64,
	PlatformLinuxDebArm64,
	PlatformLinuxDebAmd64,
	PlatformLinuxRpmArm64,
	PlatformLinuxRpmAmd64,
}

// VersionSpec describes how to resolve the required version of a tool.
type VersionSpec struct {
	Strategy   string       `yaml:"strategy"`              // go-mod | hugo-module | github-latest | github-tags | pinned
	File       string       `yaml:"file,omitempty"`        // go-mod, hugo-module: path to read
	Repo       string       `yaml:"repo,omitempty"`        // github-latest, github-tags: "owner/repo"
	TagPrefix  string       `yaml:"tag-prefix,omitempty"`  // github-latest, github-tags: strip from tag to form version
	TagFilter  string       `yaml:"tag-filter,omitempty"`  // github-tags: regexp; only matching tags are candidates
	TagExclude string       `yaml:"tag-exclude,omitempty"` // github-tags: regexp to skip (default: rc/beta/alpha/pre)
	Value      string       `yaml:"value,omitempty"`       // pinned: exact version string to return
	Fallback   *VersionSpec `yaml:"fallback,omitempty"`    // try this strategy if primary fails
}

// SLSASpec describes how to verify SLSA provenance for a downloaded artifact.
type SLSASpec struct {
	URLTemplate string `yaml:"url-template"` // provenance download URL; supports {version}, {os}, {arch}
	SourceURI   string `yaml:"source-uri"`   // expected source URI passed to slsa-verifier (e.g. github.com/ko-build/ko)
}

// DownloadSpec describes how to fetch the tool artifact.
type DownloadSpec struct {
	Strategy string    `yaml:"strategy"`         // url-template | github-release-asset | curl-script
	URL      string    `yaml:"url,omitempty"`    // url-template: supports {version}, {os}, {arch}
	Repo     string    `yaml:"repo,omitempty"`   // github-release-asset: "owner/repo"
	Filter   []string  `yaml:"filter,omitempty"` // github-release-asset: all substrings must match asset name
	Script   string    `yaml:"script,omitempty"` // curl-script: the script URL
	Args     string    `yaml:"args,omitempty"`   // curl-script: arguments appended to the curl | bash call
	SLSA     *SLSASpec `yaml:"slsa,omitempty"`   // optional SLSA provenance verification
}

// InstallSpec describes how to install the downloaded (or fetched) artifact.
type InstallSpec struct {
	Strategy string `yaml:"strategy"`          // brew-install | apt-install | dnf-install | lnx-native | go-install | run-script | extract-tar | extract-tar-binary | install-deb | install-rpm | copy-binary
	Package  string `yaml:"package,omitempty"` // brew-install, apt-install, dnf-install
	Import   string `yaml:"import,omitempty"`  // go-install: full import path@version
	Script   string `yaml:"script,omitempty"`  // run-script: shell script body
	Dest     string `yaml:"dest,omitempty"`    // extract-tar, copy-binary: destination path
	Binary   string `yaml:"binary,omitempty"`  // extract-tar-binary: path inside archive; supports {version} token
	Chmod    string `yaml:"chmod,omitempty"`   // copy-binary: permission bits (e.g. "0755")
	Sudo     bool   `yaml:"sudo,omitempty"`

	// lnx-native: the tool ships its own signed apt AND dnf repos. The installer
	// picks the apt or dnf sequence from the platform family and runs it
	// idempotently (old keyring/repo removed, then re-added) before installing
	// inst.Package. See decision D13.
	RepoName   string `yaml:"repo-name,omitempty"`    // short id for keyring/repo filenames (e.g. "helm")
	AptKeyURL  string `yaml:"apt-key-url,omitempty"`  // armored GPG key URL; dearmored to /etc/apt/keyrings/<repo-name>.gpg
	AptRepo    string `yaml:"apt-repo,omitempty"`     // sources.list line; {keyring} expands to the keyring path
	DnfRepoURL string `yaml:"dnf-repo-url,omitempty"` // .repo URL added via dnf config-manager
	DnfKeyURL  string `yaml:"dnf-key-url,omitempty"`  // optional rpm GPG key imported with rpm --import
}

// PostStep describes a step to run after successful installation.
type PostStep struct {
	Strategy string `yaml:"strategy"`          // path-append | go-install
	Line     string `yaml:"line,omitempty"`    // path-append: shell line to insert
	Profile  string `yaml:"profile,omitempty"` // path-append: target profile file
	Package  string `yaml:"package,omitempty"` // go-install: full import path
}

// PlatformEntry holds platform-specific values. Fields override the
// corresponding top-level ToolEntry fields when non-nil / non-empty.
// PostInstall, if non-empty, replaces (not extends) entry.PostInstall.
type PlatformEntry struct {
	Unsupported bool          `yaml:"unsupported,omitempty"` // true = tool cannot be installed on this platform
	OS          string        `yaml:"os,omitempty"`          // token replacement in URLs (e.g. "linux", "Linux", "darwin")
	Arch        string        `yaml:"arch,omitempty"`        // token replacement in URLs (e.g. "amd64", "arm64", "x86_64")
	Version     *VersionSpec  `yaml:"version,omitempty"`
	Download    *DownloadSpec `yaml:"download,omitempty"`
	Install     *InstallSpec  `yaml:"install,omitempty"`
	PostInstall []PostStep    `yaml:"post-install,omitempty"` // platform-specific post-install; overrides entry-level
}

// ToolEntry is the full recipe for one tool. When RecipePath is non-empty the
// entry is loaded from that file (embedfs-first, then OS filesystem) and all
// other fields come from the resolved file — full-replace semantics, no merge.
type ToolEntry struct {
	RecipePath  string                     `yaml:"recipepath,omitempty"`
	Description string                     `yaml:"description,omitempty"`
	Check       *check.Block               `yaml:"check,omitempty"`
	Version     *VersionSpec               `yaml:"version,omitempty"`
	Download    *DownloadSpec              `yaml:"download,omitempty"`
	Install     *InstallSpec               `yaml:"install,omitempty"`
	PostInstall []PostStep                 `yaml:"post-install,omitempty"`
	Platforms   map[Platform]PlatformEntry `yaml:"platforms,omitempty"`
}

// Manifest is the top-level index of all tools, loaded from index.yaml.
type Manifest struct {
	Tools map[string]ToolEntry `yaml:"tools"`
}
