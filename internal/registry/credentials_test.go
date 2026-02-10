package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

func TestFindCredentials_DockerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := dockerConfig{
		Auths: map[string]authEntry{
			"https://index.docker.io/v1/": {Auth: "dXNlcjpwYXNz"},
			"ghcr.io":                     {Auth: "dXNlcjpwYXNz"},
		},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	t.Setenv("DOCKER_CONFIG", tmpDir)

	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimeDocker}
	info, err := FindCredentials(ri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.FilePath != configPath {
		t.Errorf("FilePath = %q, want %q", info.FilePath, configPath)
	}
	if len(info.Registries) != 2 {
		t.Errorf("Registries count = %d, want 2", len(info.Registries))
	}
	if !info.InlineAuth {
		t.Error("expected InlineAuth = true when no cred helper")
	}
	if info.MountPath != "/var/lib/kubelet/config.json" {
		t.Errorf("MountPath = %q", info.MountPath)
	}
	if info.Source != "docker" {
		t.Errorf("Source = %q, want docker", info.Source)
	}
}

func TestFindCredentials_WithCredHelper(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := dockerConfig{
		CredsStore: "desktop",
		Auths:      map[string]authEntry{},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(configPath, data, 0644)

	t.Setenv("DOCKER_CONFIG", tmpDir)

	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimeDocker}
	info, err := FindCredentials(ri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.InlineAuth {
		t.Error("expected InlineAuth = false with cred helper")
	}
	if info.CredStore != "desktop" {
		t.Errorf("CredStore = %q, want desktop", info.CredStore)
	}
	if info.Notes == "" {
		t.Error("expected non-empty Notes for cred helper")
	}
}

func TestFindCredentials_PodmanAuthFile(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	cfg := dockerConfig{
		Auths: map[string]authEntry{
			"quay.io": {Auth: "dXNlcjpwYXNz"},
		},
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(authPath, data, 0644)

	t.Setenv("REGISTRY_AUTH_FILE", authPath)

	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimePodman}
	info, err := FindCredentials(ri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Source != "podman" {
		t.Errorf("Source = %q, want podman", info.Source)
	}
	if len(info.Registries) != 1 || info.Registries[0] != "quay.io" {
		t.Errorf("Registries = %v", info.Registries)
	}
}

func TestFindCredentials_NotFound(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", "/nonexistent/path")
	t.Setenv("REGISTRY_AUTH_FILE", "")
	t.Setenv("HOME", "/nonexistent")

	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimeDocker}
	_, err := FindCredentials(ri)
	if err == nil {
		t.Error("expected error when no credentials found")
	}
}

func TestFindCredentials_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(configPath, []byte("not json"), 0644)

	t.Setenv("DOCKER_CONFIG", tmpDir)
	t.Setenv("HOME", "/nonexistent")

	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimeDocker}
	_, err := FindCredentials(ri)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	got := expandPath("~/test/path")
	want := filepath.Join(home, "test/path")
	if got != want {
		t.Errorf("expandPath(~/test/path) = %q, want %q", got, want)
	}

	got = expandPath("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("expandPath(/absolute/path) = %q", got)
	}
}

func TestCandidatePaths_Docker(t *testing.T) {
	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimeDocker}
	paths := candidatePaths(ri)
	if len(paths) == 0 {
		t.Error("expected at least one candidate path for Docker")
	}
}

func TestCandidatePaths_Podman(t *testing.T) {
	ri := rtdetect.RuntimeInfo{Runtime: rtdetect.RuntimePodman}
	paths := candidatePaths(ri)
	hasDocker := false
	for _, p := range paths {
		if p.source == "docker" {
			hasDocker = true
		}
	}
	if !hasDocker {
		t.Error("expected Podman paths to include Docker fallback")
	}
}
