//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Container registry target data, per mode:
//
//	registry:
//	  kind: container-registry
//	  stack: container
//	  require: [DOCKER_PAT]
//	  prod:
//	    image: docker.io/acme/site:{tag}     # the immutable reference   (required)
//	    also-tags: [latest]                  # moving tags, same bytes     (optional)
//	    user: acme                           # login user for $DOCKER_PAT  (optional)
//	    oci: _clog_build/artifacts/site.oci  # what bc-podman wrote        (optional)
//	    tls-verify: false                    # a local test registry only  (optional)
//
// The deployer publishes what the build wrote; it never builds. bc-podman
// leaves a multi-platform OCI layout at _clog_build/artifacts/<stack>.oci, the
// artifact sweep scans that layout, CI carries it from the build job to the
// deploy job, and this pushes it - so the bytes that passed the gates are the
// bytes in the registry. ko could not offer that: its multi-platform output
// only exists as a push.
//
// podman only, no daemon and no skopeo: `manifest add --all` reads the layout
// into a throwaway local list and `manifest push --all` sends every platform.
//
// With $DOCKER_PAT set, the login is a temporary auth file, removed when the
// push ends: the token never reaches argv (ps can read argv) and never
// outlives the deploy in ~/.config/containers. Without it, whatever login
// podman already has is used - a laptop that ran `podman login` just works.
func deployContainerRegistry(d Deployment) error {
	image, err := d.Get("image", true)
	if err != nil {
		return err
	}
	name, tag, err := splitImageRef(image)
	if err != nil {
		return fmt.Errorf("ci.targets.%s.%s.image: %w", d.Target, d.Mode, err)
	}
	extra, err := d.Get("also-tags", false)
	if err != nil {
		return err
	}
	tags := []string{tag}
	for _, t := range strings.Fields(extra) {
		if t = dockerTag(t); t != tag {
			tags = append(tags, t)
		}
	}
	user, err := d.Get("user", false)
	if err != nil {
		return err
	}
	tlsVerify, err := d.GetOr("tls-verify", "true")
	if err != nil {
		return err
	}

	oci, err := d.Get("oci", false)
	if err != nil {
		return err
	}
	if oci == "" {
		all, err := Stacks(d.Cfg)
		if err != nil {
			return err
		}
		owner, err := StackOf(d.Cfg.Targets[d.Target], all)
		if err != nil {
			return err
		}
		oci = OCILayoutPath(owner.Name)
	}
	abs, err := filepath.Abs(oci)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", oci, err)
	}
	if _, err := os.Stat(filepath.Join(abs, "index.json")); err != nil {
		return fmt.Errorf("no OCI layout at %s - `clog build` writes it (make phase podman), and a CI deploy job downloads it with the build artifact", oci)
	}

	slog.Info("container-registry", "image", name, "tags", tags, "oci", oci)

	if d.DryRun {
		for _, t := range tags {
			fmt.Fprintf(d.Out, "dry-run: would push %s/ (every platform) to %s:%s\n", oci, name, t)
		}
		return nil
	}

	var auth []string
	if pat := strings.TrimSpace(d.Env.Getenv("DOCKER_PAT")); pat != "" {
		if user == "" {
			return fmt.Errorf("$DOCKER_PAT is set but ci.targets.%s.%s.user is not: a token needs the user it belongs to", d.Target, d.Mode)
		}
		file, cleanup, err := writeAuthFile(registryHost(name), user, pat)
		if err != nil {
			return err
		}
		defer cleanup()
		auth = []string{"--authfile", file}
	}

	// A local list named for the target, removed before and after: a stale one
	// from an interrupted deploy would otherwise add its platforms to this one.
	list := "localhost/clog-deploy-" + d.Target
	_, _ = execCommand(".", "podman", "manifest", "rm", list)
	defer func() { _, _ = execCommand(".", "podman", "manifest", "rm", list) }()

	if out, err := execCommand(".", "podman", "manifest", "create", list); err != nil {
		return fmt.Errorf("podman manifest create: %w%s", err, indentOutput(out))
	}
	if out, err := execCommand(".", "podman", "manifest", "add", "--all", list, "oci:"+abs); err != nil {
		return fmt.Errorf("podman manifest add %s: %w%s", oci, err, indentOutput(out))
	}
	for _, t := range tags {
		args := append([]string{"manifest", "push", "--all", "--tls-verify=" + tlsVerify}, auth...)
		args = append(args, list, "docker://"+name+":"+t)
		if out, err := execCommand(".", "podman", args...); err != nil {
			return fmt.Errorf("podman manifest push %s:%s: %w%s", name, t, err, indentOutput(out))
		}
		fmt.Fprintf(d.Out, "pushed %s:%s\n", name, t)
	}
	return nil
}

// OCILayoutPath is where bc-podman writes a stack's image and where the
// deployer and the artifact sweep look for it.
func OCILayoutPath(stack string) string {
	return filepath.Join("_clog_build", "artifacts", stack+".oci")
}

// splitImageRef splits registry/name:tag. The tag is required - a deploy that
// only moves :latest cannot be rolled back to - and is made docker-safe, since
// a dev version (v1.2.3+dev.3.gabc1234) carries a '+' no registry accepts.
func splitImageRef(ref string) (name, tag string, err error) {
	ref = strings.TrimPrefix(strings.TrimSpace(ref), "docker://")
	i := strings.LastIndex(ref, ":")
	if i <= strings.LastIndex(ref, "/") || i == len(ref)-1 {
		return "", "", fmt.Errorf("%q has no tag: write it as <registry>/<name>:{tag}", ref)
	}
	if strings.Contains(ref, "@") {
		return "", "", fmt.Errorf("%q is a digest: a deploy pushes tags", ref)
	}
	return ref[:i], dockerTag(ref[i+1:]), nil
}

// dockerTag maps a version to the tag alphabet [A-Za-z0-9_.-].
func dockerTag(t string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
			return r
		}
		return '-'
	}, strings.TrimSpace(t))
}

// registryHost is the registry part of an image name. A first component with
// no dot, colon or "localhost" is a Docker Hub namespace, as docker reads it.
func registryHost(name string) string {
	first, _, found := strings.Cut(name, "/")
	if found && (strings.ContainsAny(first, ".:") || first == "localhost") {
		return first
	}
	return "docker.io"
}

// writeAuthFile writes a containers-auth.json for one registry into a private
// temp dir and returns its path and a cleanup func.
func writeAuthFile(host, user, token string) (string, func(), error) {
	dir, err := mkdirTemp("", "clog-auth-")
	if err != nil {
		return "", nil, fmt.Errorf("auth file: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	body, err := json.Marshal(map[string]any{"auths": map[string]any{
		host: map[string]string{"auth": base64.StdEncoding.EncodeToString([]byte(user + ":" + token))},
	}})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	file := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(file, body, 0o600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("auth file: %w", err)
	}
	return file, cleanup, nil
}
