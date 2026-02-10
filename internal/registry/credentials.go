package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

// CredentialInfo holds discovered registry credential information.
type CredentialInfo struct {
	FilePath    string            `json:"file_path"`
	Registries  []string          `json:"registries"`
	CredStore   string            `json:"cred_store,omitempty"`
	CredHelpers map[string]string `json:"cred_helpers,omitempty"`
	InlineAuth  bool              `json:"inline_auth"`
	MountPath   string            `json:"mount_path"`
	Source      string            `json:"source"`
	Notes       string            `json:"notes,omitempty"`
}

// dockerConfig represents the structure of Docker/Podman config.json / auth.json.
type dockerConfig struct {
	Auths       map[string]authEntry `json:"auths"`
	CredsStore  string               `json:"credsStore,omitempty"`
	CredHelpers map[string]string    `json:"credHelpers,omitempty"`
}

type authEntry struct {
	Auth string `json:"auth,omitempty"`
}

// FindCredentials discovers registry credentials based on the container runtime and OS.
func FindCredentials(ri rtdetect.RuntimeInfo) (*CredentialInfo, error) {
	paths := candidatePaths(ri)

	for _, candidate := range paths {
		expanded := expandPath(candidate.path)
		if _, err := os.Stat(expanded); err != nil {
			continue
		}

		data, err := os.ReadFile(expanded)
		if err != nil {
			continue
		}

		var cfg dockerConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}

		info := &CredentialInfo{
			FilePath:    expanded,
			CredStore:   cfg.CredsStore,
			CredHelpers: cfg.CredHelpers,
			MountPath:   "/var/lib/kubelet/config.json",
			Source:      candidate.source,
		}

		for reg := range cfg.Auths {
			info.Registries = append(info.Registries, reg)
		}

		// Credentials are inline if there's no external helper
		info.InlineAuth = cfg.CredsStore == "" && len(cfg.CredHelpers) == 0

		if cfg.CredsStore != "" {
			info.Notes = fmt.Sprintf(
				"Credentials are managed by credential helper %q. "+
					"The config file may not contain inline credentials. "+
					"To use with Kind, you may need to extract credentials using "+
					"'docker-credential-%s get' and create a standalone config.json, "+
					"or use imagePullSecrets in your Kubernetes manifests.",
				cfg.CredsStore, cfg.CredsStore)
		}

		if !info.InlineAuth && len(info.Registries) == 0 {
			info.Notes += " No inline auth entries found in the config file."
		}

		return info, nil
	}

	return nil, fmt.Errorf("no registry credentials found; searched paths: %s",
		strings.Join(pathStrings(paths), ", "))
}

type candidatePath struct {
	path   string
	source string
}

func candidatePaths(ri rtdetect.RuntimeInfo) []candidatePath {
	var paths []candidatePath

	switch ri.Runtime {
	case rtdetect.RuntimePodman:
		if envPath := os.Getenv("REGISTRY_AUTH_FILE"); envPath != "" {
			paths = append(paths, candidatePath{envPath, "podman"})
		}
		switch goruntime.GOOS {
		case "linux":
			if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
				paths = append(paths, candidatePath{
					filepath.Join(xdg, "containers", "auth.json"), "podman"})
			}
			paths = append(paths, candidatePath{"~/.config/containers/auth.json", "podman"})
		case "darwin", "windows":
			paths = append(paths, candidatePath{"~/.config/containers/auth.json", "podman"})
		}
		// Fallback to Docker config
		paths = append(paths, candidatePath{"~/.docker/config.json", "docker"})

	case rtdetect.RuntimeDocker:
		if envPath := os.Getenv("DOCKER_CONFIG"); envPath != "" {
			paths = append(paths, candidatePath{
				filepath.Join(envPath, "config.json"), "docker"})
		}
		paths = append(paths, candidatePath{"~/.docker/config.json", "docker"})

	default:
		// Try both
		paths = append(paths,
			candidatePath{"~/.docker/config.json", "docker"},
			candidatePath{"~/.config/containers/auth.json", "podman"},
		)
	}

	return paths
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func pathStrings(paths []candidatePath) []string {
	var result []string
	for _, p := range paths {
		result = append(result, expandPath(p.path))
	}
	return result
}
