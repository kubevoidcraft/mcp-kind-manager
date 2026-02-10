package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/kind"
)

// RegistryOverride defines a mapping from an original registry to a local mirror.
type RegistryOverride struct {
	Original string `json:"original"`
	Mirror   string `json:"mirror"`
}

// MirrorConfig holds the generated containerd mirror configuration.
type MirrorConfig struct {
	ContainerdPatches  []string      `json:"containerd_patches"`
	ExtraMounts        []kind.Mount  `json:"extra_mounts,omitempty"`
	PostCreateCommands []NodeCommand `json:"post_create_commands"`
}

// NodeCommand represents a command to run on a Kind node after cluster creation.
type NodeCommand struct {
	NodeSelector string   `json:"node_selector"`
	Description  string   `json:"description"`
	Command      []string `json:"command"`
}

// GenerateMirrorConfig generates containerd mirror configuration for the given registry overrides.
func GenerateMirrorConfig(overrides []RegistryOverride, credInfo *CredentialInfo) (*MirrorConfig, error) {
	if len(overrides) == 0 {
		return nil, fmt.Errorf("at least one registry override is required")
	}

	config := &MirrorConfig{}

	// Enable containerd registry config path
	config.ContainerdPatches = []string{
		`[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"`,
	}

	// Generate post-create commands to write hosts.toml for each override
	for _, override := range overrides {
		registryDir := override.Original
		hostsToml := generateHostsToml(override)

		config.PostCreateCommands = append(config.PostCreateCommands, NodeCommand{
			NodeSelector: "all",
			Description:  fmt.Sprintf("Create registry config directory for %s", override.Original),
			Command: []string{
				"mkdir", "-p", fmt.Sprintf("/etc/containerd/certs.d/%s", registryDir),
			},
		})

		config.PostCreateCommands = append(config.PostCreateCommands, NodeCommand{
			NodeSelector: "all",
			Description:  fmt.Sprintf("Configure mirror for %s -> %s", override.Original, override.Mirror),
			Command: []string{
				"bash", "-c",
				fmt.Sprintf("cat > /etc/containerd/certs.d/%s/hosts.toml << 'EOF'\n%s\nEOF", registryDir, hostsToml),
			},
		})
	}

	// If credential info is provided and has inline auth, mount the cred file
	if credInfo != nil && credInfo.InlineAuth {
		config.ExtraMounts = append(config.ExtraMounts, kind.Mount{
			HostPath:      credInfo.FilePath,
			ContainerPath: credInfo.MountPath,
			ReadOnly:      true,
		})
	}

	return config, nil
}

// generateHostsToml creates a hosts.toml file content for a registry override.
func generateHostsToml(override RegistryOverride) string {
	var sb strings.Builder

	if override.Original == "docker.io" {
		sb.WriteString("server = \"https://registry-1.docker.io\"\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("server = \"https://%s\"\n\n", override.Original))
	}

	mirrorURL := override.Mirror
	if !strings.HasPrefix(mirrorURL, "http://") && !strings.HasPrefix(mirrorURL, "https://") {
		mirrorURL = "http://" + mirrorURL
	}

	sb.WriteString(fmt.Sprintf("[host.\"%s\"]\n", mirrorURL))
	sb.WriteString("  capabilities = [\"pull\", \"resolve\"]\n")

	if strings.HasPrefix(mirrorURL, "http://") {
		sb.WriteString("  skip_verify = true\n")
	}

	return sb.String()
}

// ApplyMirrorConfig applies mirror configuration to a running Kind cluster.
func ApplyMirrorConfig(ctx context.Context, mgr *kind.Manager, clusterName string, mirrorCfg *MirrorConfig) ([]string, error) {
	nodes, err := mgr.GetClusterNodes(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("getting cluster nodes: %w", err)
	}

	var results []string

	for _, cmd := range mirrorCfg.PostCreateCommands {
		targetNodes := filterNodes(nodes, cmd.NodeSelector)
		for _, node := range targetNodes {
			out, err := mgr.ExecOnNode(ctx, node, cmd.Command)
			if err != nil {
				results = append(results, fmt.Sprintf("FAILED [%s] %s: %v", node, cmd.Description, err))
			} else {
				msg := fmt.Sprintf("OK [%s] %s", node, cmd.Description)
				if trimmed := strings.TrimSpace(out); trimmed != "" {
					msg += ": " + trimmed
				}
				results = append(results, msg)
			}
		}
	}

	// Restart containerd on all nodes to pick up the new config
	for _, node := range nodes {
		out, err := mgr.ExecOnNode(ctx, node, []string{"systemctl", "restart", "containerd"})
		if err != nil {
			results = append(results, fmt.Sprintf("FAILED [%s] restart containerd: %v", node, err))
		} else {
			msg := fmt.Sprintf("OK [%s] restarted containerd", node)
			if trimmed := strings.TrimSpace(out); trimmed != "" {
				msg += ": " + trimmed
			}
			results = append(results, msg)
		}
	}

	return results, nil
}

// filterNodes filters node names based on the selector.
func filterNodes(nodes []string, selector string) []string {
	if selector == "all" {
		return nodes
	}

	var filtered []string
	for _, n := range nodes {
		switch selector {
		case "control-plane":
			if strings.Contains(n, "control-plane") {
				filtered = append(filtered, n)
			}
		case "worker":
			if !strings.Contains(n, "control-plane") {
				filtered = append(filtered, n)
			}
		}
	}
	return filtered
}
