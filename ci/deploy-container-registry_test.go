//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registryConfig is a two-stack repo whose container stack owns a prod-only
// registry target - the www-mrmxf-com shape.
func registryConfig(prod map[string]any) Config {
	return Config{
		Stack: StackList{{Name: "site", Type: StackHugo}, {Name: "box", Type: StackContainer}},
		Targets: map[string]Target{
			"registry": {Kind: KindRegistry, Stack: "box", Modes: []string{ModeProd}, Prod: prod},
		},
	}
}

// ociLayout makes the directory bc-podman would have written for a stack,
// inside a fresh working directory.
func ociLayout(t *testing.T, stack string) {
	t.Helper()
	t.Chdir(t.TempDir())
	dir := OCILayoutPath(stack)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(`{"schemaVersion":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixedVersion(t *testing.T, v string) {
	t.Helper()
	prev := ReleaseVersion
	ReleaseVersion = func() string { return v }
	t.Cleanup(func() { ReleaseVersion = prev })
}

func TestDeployContainerRegistryPushesEveryTagFromTheLayout(t *testing.T) {
	ociLayout(t, "box")
	fixedVersion(t, "v1.2.3")
	calls, restore := recordExec(t, "")
	defer restore()

	var out bytes.Buffer
	cfg := registryConfig(map[string]any{"image": "docker.io/acme/site:{tag}", "also-tags": []any{"latest"}})
	if err := Deploy(laptop(nil, fakeGit{}), cfg, ModeProd, "", false, &out); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	joined := strings.Join(*calls, "\n")
	for _, want := range []string{
		"podman manifest create localhost/clog-deploy-registry",
		"podman manifest add --all localhost/clog-deploy-registry oci:",
		filepath.Join("_clog_build", "artifacts", "box.oci"),
		"manifest push --all --tls-verify=true localhost/clog-deploy-registry docker://docker.io/acme/site:v1.2.3",
		"manifest push --all --tls-verify=true localhost/clog-deploy-registry docker://docker.io/acme/site:latest",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q, got:\n%s", want, joined)
		}
	}
	// the throwaway list is removed afterwards, not left to leak into the next deploy
	if last := (*calls)[len(*calls)-1]; last != "podman manifest rm localhost/clog-deploy-registry" {
		t.Errorf("last call = %q, want the list removed", last)
	}
}

// A dev version carries a '+' that no registry accepts as a tag.
func TestDeployContainerRegistryMakesTheTagDockerSafe(t *testing.T) {
	ociLayout(t, "box")
	fixedVersion(t, "v1.2.3+dev.3.gabc1234")
	calls, restore := recordExec(t, "")
	defer restore()

	cfg := registryConfig(map[string]any{"image": "acme/site:{tag}"})
	if err := Deploy(laptop(nil, fakeGit{}), cfg, ModeProd, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "docker://acme/site:v1.2.3-dev.3.gabc1234") {
		t.Errorf("tag not made docker-safe:\n%s", strings.Join(*calls, "\n"))
	}
}

// The token logs in through a temporary auth file: never argv, never left behind.
func TestDeployContainerRegistryKeepsTheTokenOutOfArgv(t *testing.T) {
	ociLayout(t, "box")
	fixedVersion(t, "v1.2.3")
	var authfile string
	var calls []string
	prev := execCommand
	execCommand = func(dir, name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		for i, a := range args {
			if a == "--authfile" {
				authfile = args[i+1]
				body, err := os.ReadFile(authfile)
				if err != nil || !strings.Contains(string(body), `"docker.io"`) {
					t.Errorf("auth file unreadable or wrong host: %v %s", err, body)
				}
			}
		}
		return "", nil
	}
	defer func() { execCommand = prev }()

	cfg := registryConfig(map[string]any{"image": "docker.io/acme/site:{tag}", "user": "acme"})
	env := laptop(map[string]string{"DOCKER_PAT": "s3cr3t"}, fakeGit{})
	if err := Deploy(env, cfg, ModeProd, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if strings.Contains(strings.Join(calls, "\n"), "s3cr3t") {
		t.Error("the token reached argv")
	}
	if authfile == "" {
		t.Fatal("no --authfile passed with DOCKER_PAT set")
	}
	if _, err := os.Stat(authfile); !os.IsNotExist(err) {
		t.Errorf("auth file %s outlived the deploy", authfile)
	}
}

func TestDeployContainerRegistryRefuses(t *testing.T) {
	fixedVersion(t, "v1.2.3")
	_, restore := recordExec(t, "")
	defer restore()

	cases := []struct {
		name   string
		layout bool
		env    map[string]string
		prod   map[string]any
		want   string
	}{
		{"no layout", false, nil, map[string]any{"image": "acme/site:{tag}"}, "clog build"},
		{"no tag", true, nil, map[string]any{"image": "acme/site"}, "has no tag"},
		{"digest", true, nil, map[string]any{"image": "acme/site@sha256:abc:1"}, "digest"},
		{"token without user", true, map[string]string{"DOCKER_PAT": "x"}, map[string]any{"image": "acme/site:{tag}"}, "user"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.layout {
				ociLayout(t, "box")
			} else {
				t.Chdir(t.TempDir())
			}
			err := Deploy(laptop(c.env, fakeGit{}), registryConfig(c.prod), ModeProd, "", false, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestDeployContainerRegistryDryRunTouchesNothing(t *testing.T) {
	ociLayout(t, "box")
	fixedVersion(t, "v1.2.3")
	calls, restore := recordExec(t, "")
	defer restore()

	var out bytes.Buffer
	cfg := registryConfig(map[string]any{"image": "acme/site:{tag}", "also-tags": "latest"})
	if err := Deploy(laptop(nil, fakeGit{}), cfg, ModeProd, "", true, &out); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("dry run ran %v", *calls)
	}
	if !strings.Contains(out.String(), "acme/site:v1.2.3") || !strings.Contains(out.String(), "acme/site:latest") {
		t.Errorf("dry run should name both tags, got %q", out.String())
	}
}

// A prod-only registry target does not deploy in dev, even though its stack
// builds in dev: `clog deploy` on a laptop publishes Pages alone.
func TestRegistryTargetStaysOutOfDev(t *testing.T) {
	names, err := TargetNames(registryConfig(map[string]any{"image": "a/b:{tag}"}), ModeDev, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("dev targets = %v, want none", names)
	}
}

func TestRegistryHost(t *testing.T) {
	for in, want := range map[string]string{
		"acme/site":                 "docker.io",
		"docker.io/acme/site":       "docker.io",
		"ghcr.io/acme/site":         "ghcr.io",
		"localhost:5000/site":       "localhost:5000",
		"localhost/site":            "localhost",
		"registry.gitlab.com/a/b/c": "registry.gitlab.com",
	} {
		if got := registryHost(in); got != want {
			t.Errorf("registryHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTargetGetStackResolvesTheOwner(t *testing.T) {
	cfg := registryConfig(map[string]any{"image": "a/b:{tag}"})
	cfg.Targets["pages"] = Target{Kind: KindCloudflarePage, Prod: map[string]any{"dir": "kodata"}}
	for target, want := range map[string]string{"registry": "box", "pages": "site"} {
		got, err := TargetGet(laptop(nil, fakeGit{}), cfg, ModeProd, target, "stack", false)
		if err != nil || got != want {
			t.Errorf("stack of %s = %q, %v; want %q", target, got, err, want)
		}
	}
}

func TestStackWith(t *testing.T) {
	s := Stack{With: map[string]any{
		"containerfile": "www-podserver/Containerfile",
		"platforms":     []any{"linux/amd64", "linux/arm64"},
		"empty":         "",
	}}
	if got := StackWith(s, "containerfile"); got != "www-podserver/Containerfile" {
		t.Errorf("scalar = %q", got)
	}
	if got := StackWith(s, "platforms"); got != "linux/amd64\nlinux/arm64" {
		t.Errorf("list = %q, want one per line", got)
	}
	if got := StackWith(s, "empty") + StackWith(s, "missing"); got != "" {
		t.Errorf("unset = %q, want empty", got)
	}
}

// A prod-only registry target built by podman is swept from its OCI layout on
// a dev build, instead of failing for want of a dev: image reference.
func TestRegistryScanUsesTheLayoutInEveryMode(t *testing.T) {
	cfg := registryConfig(map[string]any{"image": "a/b:{tag}"})
	cfg.Stack[1].With = map[string]any{"containerfile": "Containerfile"}
	for _, mode := range []string{ModeDev, ModeProd} {
		a, err := ArtifactScan(laptop(nil, fakeGit{}), cfg, mode, "registry")
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if a.Vuln != VulnImage || a.Ref != OCILayoutPath("box") {
			t.Errorf("%s: sweep = %s %q, want image %q", mode, a.Vuln, a.Ref, OCILayoutPath("box"))
		}
	}
	// without a containerfile it is a ko-era target and keeps the image ref
	cfg.Stack[1].With = nil
	if a, err := ArtifactScan(laptop(nil, fakeGit{}), cfg, ModeProd, "registry"); err != nil || !strings.HasPrefix(a.Ref, "a/b:") {
		t.Errorf("ko-era ref = %q, %v", a.Ref, err)
	}
}
