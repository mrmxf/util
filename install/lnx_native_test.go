//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"strings"
	"testing"
)

func TestLnxNativeScript_Deb(t *testing.T) {
	inst := &InstallSpec{
		Package:   "trivy",
		RepoName:  "trivy",
		AptKeyURL: "https://aquasecurity.github.io/trivy-repo/deb/public.key",
		AptRepo:   "deb [signed-by={keyring}] https://aquasecurity.github.io/trivy-repo/deb generic main",
	}
	script, err := lnxNativeScript(inst, "deb")
	if err != nil {
		t.Fatal(err)
	}
	wantContains := []string{
		"mkdir -p /etc/apt/keyrings",
		`rm -f "/etc/apt/keyrings/trivy.gpg" "/etc/apt/sources.list.d/trivy.list"`, // idempotent
		"gpg --dearmor",
		"signed-by=/etc/apt/keyrings/trivy.gpg", // {keyring} substituted
		"apt-get update",
		`apt-get install -y "trivy"`,
	}
	for _, w := range wantContains {
		if !strings.Contains(script, w) {
			t.Errorf("deb script missing %q\n---\n%s", w, script)
		}
	}
}

func TestLnxNativeScript_Rpm(t *testing.T) {
	inst := &InstallSpec{
		Package:    "trivy",
		RepoName:   "trivy",
		DnfRepoURL: "https://aquasecurity.github.io/trivy-repo/rpm/releases/$releasever/$basearch/",
		DnfKeyURL:  "https://aquasecurity.github.io/trivy-repo/rpm/public.key",
	}
	script, err := lnxNativeScript(inst, "rpm")
	if err != nil {
		t.Fatal(err)
	}
	wantContains := []string{
		`rm -f "/etc/yum.repos.d/trivy.repo"`, // idempotent
		"dnf config-manager --add-repo",
		"rpm --import",
		`dnf install -y "trivy"`,
	}
	for _, w := range wantContains {
		if !strings.Contains(script, w) {
			t.Errorf("rpm script missing %q\n---\n%s", w, script)
		}
	}
}

func TestLnxNativeScript_Errors(t *testing.T) {
	if _, err := lnxNativeScript(&InstallSpec{}, "deb"); err == nil {
		t.Error("expected error when package is empty")
	}
	if _, err := lnxNativeScript(&InstallSpec{Package: "x"}, "deb"); err == nil {
		t.Error("expected error when apt fields missing")
	}
	if _, err := lnxNativeScript(&InstallSpec{Package: "x"}, "rpm"); err == nil {
		t.Error("expected error when dnf-repo-url missing")
	}
	if _, err := lnxNativeScript(&InstallSpec{Package: "x"}, "darwin"); err == nil {
		t.Error("expected error for non-linux family")
	}
}
