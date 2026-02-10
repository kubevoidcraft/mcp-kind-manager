package kind

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

// Manager wraps the Kind CLI for cluster lifecycle operations.
type Manager struct {
	runner  rtdetect.CommandRunner
	runtime rtdetect.RuntimeInfo
	logger  *slog.Logger
}

// ClusterStatus holds the status of a Kind cluster and its nodes.
type ClusterStatus struct {
	Name  string       `json:"name"`
	Nodes []NodeStatus `json:"nodes"`
}

// NodeStatus holds status information for a single node.
type NodeStatus struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// NewManager creates a new Kind CLI manager.
func NewManager(runner rtdetect.CommandRunner, ri rtdetect.RuntimeInfo, logger *slog.Logger) *Manager {
	if runner == nil {
		runner = &rtdetect.ExecCommandRunner{}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &Manager{
		runner:  runner,
		runtime: ri,
		logger:  logger,
	}
}

// kindArgs returns extra args for the kind CLI based on the runtime (e.g. podman provider).
func (m *Manager) kindArgs() []string {
	if m.runtime.Runtime == rtdetect.RuntimePodman {
		return []string{"--runtime", "podman"}
	}
	return nil
}

// CreateCluster creates a Kind cluster from the given config YAML.
func (m *Manager) CreateCluster(ctx context.Context, name string, configYAML string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cluster name is required")
	}
	if err := ValidateConfig(configYAML); err != nil {
		return "", fmt.Errorf("invalid config: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "kind-config-*.yaml")
	if err != nil {
		return "", fmt.Errorf("creating temp config file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configYAML); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing config to temp file: %w", err)
	}
	tmpFile.Close()

	args := append(m.kindArgs(), "create", "cluster", "--name", name, "--config", tmpFile.Name())

	m.logger.Info("creating kind cluster", "name", name)
	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return string(out), fmt.Errorf("kind create cluster failed: %w\nOutput: %s", err, string(out))
	}

	return string(out), nil
}

// DeleteCluster deletes a Kind cluster by name.
func (m *Manager) DeleteCluster(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cluster name is required")
	}

	args := append(m.kindArgs(), "delete", "cluster", "--name", name)

	m.logger.Info("deleting kind cluster", "name", name)
	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return string(out), fmt.Errorf("kind delete cluster failed: %w\nOutput: %s", err, string(out))
	}

	return string(out), nil
}

// ListClusters returns a list of Kind cluster names.
func (m *Manager) ListClusters(ctx context.Context) ([]string, error) {
	m.logger.Debug("listing kind clusters")
	args := append(m.kindArgs(), "get", "clusters")

	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return nil, fmt.Errorf("kind get clusters failed: %w\nOutput: %s", err, string(out))
	}

	output := strings.TrimSpace(string(out))
	if output == "" || output == "No kind clusters found." {
		return []string{}, nil
	}

	var clusters []string
	for _, c := range strings.Split(output, "\n") {
		c = strings.TrimSpace(c)
		if c != "" {
			clusters = append(clusters, c)
		}
	}

	return clusters, nil
}

// GetKubeconfig returns the kubeconfig for a Kind cluster.
func (m *Manager) GetKubeconfig(ctx context.Context, name string, internal bool) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cluster name is required")
	}

	m.logger.Debug("getting kubeconfig", "cluster", name, "internal", internal)
	args := append(m.kindArgs(), "get", "kubeconfig", "--name", name)
	if internal {
		args = append(args, "--internal")
	}

	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return "", fmt.Errorf("kind get kubeconfig failed: %w\nOutput: %s", err, string(out))
	}

	return string(out), nil
}

// GetClusterStatus returns the status of a Kind cluster including node states.
func (m *Manager) GetClusterStatus(ctx context.Context, name string) (*ClusterStatus, error) {
	if name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	m.logger.Debug("getting cluster status", "cluster", name)
	args := append(m.kindArgs(), "get", "nodes", "--name", name)

	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return nil, fmt.Errorf("kind get nodes failed: %w\nOutput: %s", err, string(out))
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, fmt.Errorf("cluster %q not found or has no nodes", name)
	}

	status := &ClusterStatus{Name: name}

	runtimeBin := "docker"
	if m.runtime.Runtime == rtdetect.RuntimePodman {
		runtimeBin = "podman"
	}

	for _, nodeName := range strings.Split(output, "\n") {
		nodeName = strings.TrimSpace(nodeName)
		if nodeName == "" {
			continue
		}

		ns := NodeStatus{Name: nodeName}

		if strings.Contains(nodeName, "control-plane") {
			ns.Role = "control-plane"
		} else {
			ns.Role = "worker"
		}

		inspectOut, err := m.runner.Run(ctx, runtimeBin, "inspect",
			"--format", "{{.State.Status}}", nodeName)
		if err != nil {
			ns.Status = "unknown"
		} else {
			ns.Status = strings.TrimSpace(string(inspectOut))
		}

		status.Nodes = append(status.Nodes, ns)
	}

	return status, nil
}

// ExecOnNode runs a command on a Kind node container.
func (m *Manager) ExecOnNode(ctx context.Context, nodeName string, cmd []string) (string, error) {
	m.logger.Debug("exec on node", "node", nodeName, "cmd", cmd)
	runtimeBin := "docker"
	if m.runtime.Runtime == rtdetect.RuntimePodman {
		runtimeBin = "podman"
	}

	args := append([]string{"exec", nodeName}, cmd...)
	out, err := m.runner.Run(ctx, runtimeBin, args...)
	if err != nil {
		return string(out), fmt.Errorf("exec on node %q failed: %w\nOutput: %s", nodeName, err, string(out))
	}

	return string(out), nil
}

// GetClusterNodes returns node names for a Kind cluster.
func (m *Manager) GetClusterNodes(ctx context.Context, name string) ([]string, error) {
	args := append(m.kindArgs(), "get", "nodes", "--name", name)

	out, err := m.runner.Run(ctx, "kind", args...)
	if err != nil {
		return nil, fmt.Errorf("kind get nodes failed: %w", err)
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, nil
	}

	var nodes []string
	for _, n := range strings.Split(output, "\n") {
		n = strings.TrimSpace(n)
		if n != "" {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}
