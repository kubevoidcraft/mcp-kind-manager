package registry

import (
	"strings"
	"testing"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/kind"
)

func TestGenerateMirrorConfig_Basic(t *testing.T) {
	overrides := []RegistryOverride{
		{Original: "docker.io", Mirror: "http://my-proxy:5000"},
	}

	cfg, err := GenerateMirrorConfig(overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.ContainerdPatches) != 1 {
		t.Errorf("expected 1 containerd patch, got %d", len(cfg.ContainerdPatches))
	}
	if !strings.Contains(cfg.ContainerdPatches[0], "config_path") {
		t.Error("containerd patch should contain config_path")
	}

	// 2 commands per override: mkdir + write hosts.toml
	if len(cfg.PostCreateCommands) != 2 {
		t.Errorf("expected 2 post-create commands, got %d", len(cfg.PostCreateCommands))
	}

	if len(cfg.ExtraMounts) != 0 {
		t.Errorf("expected 0 extra mounts without creds, got %d", len(cfg.ExtraMounts))
	}
}

func TestGenerateMirrorConfig_MultipleOverrides(t *testing.T) {
	overrides := []RegistryOverride{
		{Original: "docker.io", Mirror: "http://proxy:5000"},
		{Original: "ghcr.io", Mirror: "http://proxy:5001"},
		{Original: "quay.io", Mirror: "https://proxy:5002"},
	}

	cfg, err := GenerateMirrorConfig(overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 commands per override
	if len(cfg.PostCreateCommands) != 6 {
		t.Errorf("expected 6 post-create commands, got %d", len(cfg.PostCreateCommands))
	}
}

func TestGenerateMirrorConfig_WithCredentials(t *testing.T) {
	overrides := []RegistryOverride{
		{Original: "docker.io", Mirror: "http://proxy:5000"},
	}
	creds := &CredentialInfo{
		FilePath:   "/home/user/.docker/config.json",
		MountPath:  "/var/lib/kubelet/config.json",
		InlineAuth: true,
	}

	cfg, err := GenerateMirrorConfig(overrides, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.ExtraMounts) != 1 {
		t.Fatalf("expected 1 extra mount, got %d", len(cfg.ExtraMounts))
	}
	if cfg.ExtraMounts[0].HostPath != "/home/user/.docker/config.json" {
		t.Errorf("mount HostPath = %q", cfg.ExtraMounts[0].HostPath)
	}
	if !cfg.ExtraMounts[0].ReadOnly {
		t.Error("mount should be ReadOnly")
	}
}

func TestGenerateMirrorConfig_CredHelperSkipsMount(t *testing.T) {
	overrides := []RegistryOverride{
		{Original: "docker.io", Mirror: "http://proxy:5000"},
	}
	creds := &CredentialInfo{
		FilePath:   "/home/user/.docker/config.json",
		MountPath:  "/var/lib/kubelet/config.json",
		InlineAuth: false,
		CredStore:  "desktop",
	}

	cfg, err := GenerateMirrorConfig(overrides, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.ExtraMounts) != 0 {
		t.Errorf("should not mount when cred helper is used, got %d mounts", len(cfg.ExtraMounts))
	}
}

func TestGenerateMirrorConfig_Empty(t *testing.T) {
	_, err := GenerateMirrorConfig(nil, nil)
	if err == nil {
		t.Error("expected error for empty overrides")
	}
}

func TestGenerateHostsToml_DockerIO(t *testing.T) {
	override := RegistryOverride{Original: "docker.io", Mirror: "http://proxy:5000"}
	toml := generateHostsToml(override)

	if !strings.Contains(toml, "registry-1.docker.io") {
		t.Error("docker.io should use registry-1.docker.io as server")
	}
	if !strings.Contains(toml, "http://proxy:5000") {
		t.Error("should contain mirror URL")
	}
	if !strings.Contains(toml, "skip_verify = true") {
		t.Error("http mirrors should have skip_verify")
	}
	if !strings.Contains(toml, "pull") || !strings.Contains(toml, "resolve") {
		t.Error("should have pull and resolve capabilities")
	}
}

func TestGenerateHostsToml_HTTPS(t *testing.T) {
	override := RegistryOverride{Original: "ghcr.io", Mirror: "https://secure-proxy:5000"}
	toml := generateHostsToml(override)

	if !strings.Contains(toml, "https://ghcr.io") {
		t.Error("should use original registry as server")
	}
	if strings.Contains(toml, "skip_verify") {
		t.Error("https mirrors should not have skip_verify")
	}
}

func TestGenerateHostsToml_NoScheme(t *testing.T) {
	override := RegistryOverride{Original: "quay.io", Mirror: "proxy:5000"}
	toml := generateHostsToml(override)

	if !strings.Contains(toml, "http://proxy:5000") {
		t.Error("should default to http:// when no scheme")
	}
}

func TestFilterNodes(t *testing.T) {
	nodes := []string{"test-control-plane", "test-worker", "test-worker2"}

	all := filterNodes(nodes, "all")
	if len(all) != 3 {
		t.Errorf("all: got %d, want 3", len(all))
	}

	cp := filterNodes(nodes, "control-plane")
	if len(cp) != 1 || cp[0] != "test-control-plane" {
		t.Errorf("control-plane: got %v", cp)
	}

	workers := filterNodes(nodes, "worker")
	if len(workers) != 2 {
		t.Errorf("worker: got %d, want 2", len(workers))
	}
}

func TestMirrorConfig_MountStruct(t *testing.T) {
	m := kind.Mount{
		HostPath:      "/a",
		ContainerPath: "/b",
		ReadOnly:      true,
	}
	if m.HostPath != "/a" || m.ContainerPath != "/b" || !m.ReadOnly {
		t.Errorf("mount = %+v", m)
	}
}
