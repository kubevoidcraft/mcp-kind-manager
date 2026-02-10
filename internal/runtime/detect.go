package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Runtime represents a container runtime type.
type Runtime string

const (
	RuntimeDocker  Runtime = "docker"
	RuntimePodman  Runtime = "podman"
	RuntimeUnknown Runtime = "unknown"
)

// Backend represents the container runtime backend/engine.
type Backend string

const (
	BackendDockerDesktop  Backend = "docker-desktop"
	BackendColima         Backend = "colima"
	BackendWSL            Backend = "wsl"
	BackendPodmanMachine  Backend = "podman-machine"
	BackendNative         Backend = "native"
	BackendRancherDesktop Backend = "rancher-desktop"
	BackendLima           Backend = "lima"
	BackendUnknown        Backend = "unknown"
)

// RuntimeInfo holds information about the detected container runtime.
type RuntimeInfo struct {
	Runtime    Runtime `json:"runtime"`
	Backend    Backend `json:"backend"`
	Version    string  `json:"version"`
	SocketPath string  `json:"socket_path,omitempty"`
	OS         OSInfo  `json:"os"`
	Available  bool    `json:"available"`
	Error      string  `json:"error,omitempty"`
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	LookPath(name string) (string, error)
}

// ExecCommandRunner is the real implementation using os/exec.
type ExecCommandRunner struct{}

// Run executes a command and returns combined output.
func (r *ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// LookPath searches for an executable in PATH.
func (r *ExecCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// Detector detects container runtime information.
type Detector struct {
	runner CommandRunner
}

// NewDetector creates a new Detector with the given CommandRunner.
func NewDetector(runner CommandRunner) *Detector {
	if runner == nil {
		runner = &ExecCommandRunner{}
	}
	return &Detector{runner: runner}
}

// dockerInfo is a subset of docker info JSON output.
type dockerInfo struct {
	ServerVersion   string `json:"ServerVersion"`
	OperatingSystem string `json:"OperatingSystem"`
	OSType          string `json:"OSType"`
	Architecture    string `json:"Architecture"`
	Name            string `json:"Name"`
}

// podmanInfo is a subset of podman info JSON output.
type podmanInfo struct {
	Host struct {
		RemoteSocket struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"remoteSocket"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
		Version struct {
			Version string `json:"Version"`
		} `json:"version"`
	} `json:"host"`
}

// Detect detects the container runtime and backend.
func (d *Detector) Detect(ctx context.Context) RuntimeInfo {
	osInfo := DetectOS()
	info := RuntimeInfo{
		Runtime:   RuntimeUnknown,
		Backend:   BackendUnknown,
		Available: false,
		OS:        osInfo,
	}

	// Try Docker first
	if _, err := d.runner.LookPath("docker"); err == nil {
		if ri, err := d.detectDocker(ctx, osInfo); err == nil {
			return ri
		}
	}

	// Try Podman
	if _, err := d.runner.LookPath("podman"); err == nil {
		if ri, err := d.detectPodman(ctx, osInfo); err == nil {
			return ri
		}
	}

	info.Error = "no container runtime detected; install Docker or Podman"
	return info
}

func (d *Detector) detectDocker(ctx context.Context, osInfo OSInfo) (RuntimeInfo, error) {
	info := RuntimeInfo{
		Runtime:   RuntimeDocker,
		Available: true,
		OS:        osInfo,
	}

	out, err := d.runner.Run(ctx, "docker", "info", "--format", "{{json .}}")
	if err != nil {
		return info, fmt.Errorf("docker info failed: %w", err)
	}

	var di dockerInfo
	if err := json.Unmarshal(out, &di); err != nil {
		return info, fmt.Errorf("parsing docker info: %w", err)
	}

	info.Version = di.ServerVersion
	info.Backend = detectDockerBackend(di, osInfo)
	info.SocketPath = detectDockerSocket()

	return info, nil
}

func (d *Detector) detectPodman(ctx context.Context, osInfo OSInfo) (RuntimeInfo, error) {
	info := RuntimeInfo{
		Runtime:   RuntimePodman,
		Available: true,
		OS:        osInfo,
	}

	out, err := d.runner.Run(ctx, "podman", "info", "--format", "json")
	if err != nil {
		return info, fmt.Errorf("podman info failed: %w", err)
	}

	var pi podmanInfo
	if err := json.Unmarshal(out, &pi); err != nil {
		return info, fmt.Errorf("parsing podman info: %w", err)
	}

	info.Version = pi.Host.Version.Version
	info.SocketPath = pi.Host.RemoteSocket.Path
	info.Backend = d.detectPodmanBackend(ctx, osInfo)

	return info, nil
}

func detectDockerBackend(di dockerInfo, osInfo OSInfo) Backend {
	osField := strings.ToLower(di.OperatingSystem)
	nameField := strings.ToLower(di.Name)

	if strings.Contains(osField, "docker desktop") {
		return BackendDockerDesktop
	}
	if strings.Contains(osField, "colima") || strings.Contains(nameField, "colima") {
		return BackendColima
	}
	if strings.Contains(nameField, "rancher") {
		return BackendRancherDesktop
	}
	if strings.Contains(nameField, "lima") {
		return BackendLima
	}

	// Check socket path for hints
	socketPath := detectDockerSocket()
	if strings.Contains(socketPath, ".colima") {
		return BackendColima
	}
	if strings.Contains(socketPath, ".lima") {
		return BackendLima
	}
	if strings.Contains(socketPath, ".rd") {
		return BackendRancherDesktop
	}

	// WSL detection on Linux
	if osInfo.OS == "linux" && isWSL() {
		return BackendWSL
	}

	// Native Linux
	if osInfo.OS == "linux" {
		return BackendNative
	}

	return BackendDockerDesktop // macOS/Windows default assumption
}

func (d *Detector) detectPodmanBackend(ctx context.Context, osInfo OSInfo) Backend {
	switch osInfo.OS {
	case "darwin", "windows":
		// On macOS/Windows, Podman always uses a machine
		return BackendPodmanMachine
	case "linux":
		if isWSL() {
			return BackendWSL
		}
		return BackendNative
	}
	return BackendUnknown
}

func detectDockerSocket() string {
	// Check DOCKER_HOST first
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		if strings.HasPrefix(host, "unix://") {
			return strings.TrimPrefix(host, "unix://")
		}
		return host
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, ".colima", "default", "docker.sock"),
			filepath.Join(home, ".docker", "run", "docker.sock"),
			filepath.Join(home, ".rd", "docker.sock"),
			"/var/run/docker.sock",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	case "linux":
		var candidates []string
		if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			candidates = append(candidates, filepath.Join(xdg, "docker.sock"))
		}
		candidates = append(candidates, "/var/run/docker.sock")
		home, _ := os.UserHomeDir()
		candidates = append(candidates, filepath.Join(home, ".docker", "run", "docker.sock"))
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	case "windows":
		return `\\.\pipe\docker_engine`
	}

	return "/var/run/docker.sock"
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
