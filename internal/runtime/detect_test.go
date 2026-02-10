package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// mockRunner implements CommandRunner for testing.
type mockRunner struct {
	lookPathResults map[string]error
	runResults      map[string]runResult
}

type runResult struct {
	output []byte
	err    error
}

func (m *mockRunner) LookPath(name string) (string, error) {
	if err, ok := m.lookPathResults[name]; ok {
		return "", err
	}
	return "/usr/bin/" + name, nil
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if r, ok := m.runResults[key]; ok {
		return r.output, r.err
	}
	return nil, fmt.Errorf("no mock for %q", key)
}

func TestDetect_DockerDesktop(t *testing.T) {
	di := dockerInfo{
		ServerVersion:   "27.0.3",
		OperatingSystem: "Docker Desktop",
		Name:            "docker-desktop",
	}
	diJSON, _ := json.Marshal(di)

	runner := &mockRunner{
		lookPathResults: map[string]error{},
		runResults: map[string]runResult{
			"docker info": {output: diJSON},
		},
	}

	d := NewDetector(runner)
	ri := d.Detect(context.Background())

	if ri.Runtime != RuntimeDocker {
		t.Errorf("Runtime = %q, want %q", ri.Runtime, RuntimeDocker)
	}
	if !ri.Available {
		t.Error("Expected Available = true")
	}
	if ri.Version != "27.0.3" {
		t.Errorf("Version = %q, want %q", ri.Version, "27.0.3")
	}
	if ri.Backend != BackendDockerDesktop {
		t.Errorf("Backend = %q, want %q", ri.Backend, BackendDockerDesktop)
	}
}

func TestDetect_Colima(t *testing.T) {
	di := dockerInfo{
		ServerVersion:   "24.0.7",
		OperatingSystem: "Ubuntu 22.04",
		Name:            "colima",
	}
	diJSON, _ := json.Marshal(di)

	runner := &mockRunner{
		lookPathResults: map[string]error{},
		runResults: map[string]runResult{
			"docker info": {output: diJSON},
		},
	}

	d := NewDetector(runner)
	ri := d.Detect(context.Background())

	if ri.Runtime != RuntimeDocker {
		t.Errorf("Runtime = %q, want %q", ri.Runtime, RuntimeDocker)
	}
	if ri.Backend != BackendColima {
		t.Errorf("Backend = %q, want %q", ri.Backend, BackendColima)
	}
}

func TestDetect_PodmanFallback(t *testing.T) {
	pi := podmanInfo{}
	pi.Host.Version.Version = "5.0.0"
	pi.Host.RemoteSocket.Path = "/run/podman/podman.sock"
	piJSON, _ := json.Marshal(pi)

	runner := &mockRunner{
		lookPathResults: map[string]error{
			"docker": fmt.Errorf("not found"),
		},
		runResults: map[string]runResult{
			"podman info": {output: piJSON},
		},
	}

	d := NewDetector(runner)
	ri := d.Detect(context.Background())

	if ri.Runtime != RuntimePodman {
		t.Errorf("Runtime = %q, want %q", ri.Runtime, RuntimePodman)
	}
	if !ri.Available {
		t.Error("Expected Available = true")
	}
	if ri.Version != "5.0.0" {
		t.Errorf("Version = %q, want %q", ri.Version, "5.0.0")
	}
}

func TestDetect_NoRuntime(t *testing.T) {
	runner := &mockRunner{
		lookPathResults: map[string]error{
			"docker": fmt.Errorf("not found"),
			"podman": fmt.Errorf("not found"),
		},
		runResults: map[string]runResult{},
	}

	d := NewDetector(runner)
	ri := d.Detect(context.Background())

	if ri.Runtime != RuntimeUnknown {
		t.Errorf("Runtime = %q, want %q", ri.Runtime, RuntimeUnknown)
	}
	if ri.Available {
		t.Error("Expected Available = false")
	}
	if ri.Error == "" {
		t.Error("Expected non-empty Error")
	}
}

func TestDetect_DockerFailsFallsToPodman(t *testing.T) {
	pi := podmanInfo{}
	pi.Host.Version.Version = "4.9.0"
	piJSON, _ := json.Marshal(pi)

	runner := &mockRunner{
		lookPathResults: map[string]error{},
		runResults: map[string]runResult{
			"docker info": {err: fmt.Errorf("cannot connect")},
			"podman info": {output: piJSON},
		},
	}

	d := NewDetector(runner)
	ri := d.Detect(context.Background())

	if ri.Runtime != RuntimePodman {
		t.Errorf("Runtime = %q, want %q", ri.Runtime, RuntimePodman)
	}
	if ri.Version != "4.9.0" {
		t.Errorf("Version = %q, want %q", ri.Version, "4.9.0")
	}
}
