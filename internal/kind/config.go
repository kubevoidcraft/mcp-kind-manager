package kind

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ClusterConfig represents a Kind cluster configuration.
type ClusterConfig struct {
	Kind                    string          `yaml:"kind"`
	APIVersion              string          `yaml:"apiVersion"`
	Name                    string          `yaml:"name,omitempty"`
	Nodes                   []NodeConfig    `yaml:"nodes,omitempty"`
	Networking              *NetworkConfig  `yaml:"networking,omitempty"`
	FeatureGates            map[string]bool `yaml:"featureGates,omitempty"`
	ContainerdConfigPatches []string        `yaml:"containerdConfigPatches,omitempty"`
}

// NodeConfig represents a Kind node configuration.
type NodeConfig struct {
	Role                 string            `yaml:"role"`
	Image                string            `yaml:"image,omitempty"`
	ExtraPortMappings    []PortMapping     `yaml:"extraPortMappings,omitempty"`
	ExtraMounts          []Mount           `yaml:"extraMounts,omitempty"`
	Labels               map[string]string `yaml:"labels,omitempty"`
	KubeadmConfigPatches []string          `yaml:"kubeadmConfigPatches,omitempty"`
}

// Mount represents a host-to-container mount in Kind.
type Mount struct {
	HostPath      string `yaml:"hostPath" json:"host_path"`
	ContainerPath string `yaml:"containerPath" json:"container_path"`
	ReadOnly      bool   `yaml:"readOnly,omitempty" json:"read_only,omitempty"`
	Propagation   string `yaml:"propagation,omitempty" json:"propagation,omitempty"`
}

// NetworkConfig represents Kind cluster networking options.
type NetworkConfig struct {
	IPFamily          string `yaml:"ipFamily,omitempty"`
	APIServerAddress  string `yaml:"apiServerAddress,omitempty"`
	APIServerPort     int    `yaml:"apiServerPort,omitempty"`
	PodSubnet         string `yaml:"podSubnet,omitempty"`
	ServiceSubnet     string `yaml:"serviceSubnet,omitempty"`
	DisableDefaultCNI bool   `yaml:"disableDefaultCNI,omitempty"`
	KubeProxyMode     string `yaml:"kubeProxyMode,omitempty"`
}

// ConfigOptions holds the parameters for generating a Kind cluster config.
type ConfigOptions struct {
	ClusterName       string
	NumWorkers        int
	NumControlPlanes  int
	KubernetesVersion string
	PortMappings      []PortMapping
	ExtraMounts       []Mount
	ContainerdPatches []string
	PodSubnet         string
	ServiceSubnet     string
	DisableDefaultCNI bool
	Labels            map[string]string
	IPFamily          string
	KubeProxyMode     string
	APIServerPort     int
}

// GenerateConfig generates a Kind cluster configuration YAML from the given options.
func GenerateConfig(opts ConfigOptions) (string, error) {
	if opts.ClusterName == "" {
		return "", fmt.Errorf("cluster name is required")
	}
	if opts.NumControlPlanes <= 0 {
		opts.NumControlPlanes = 1
	}

	cfg := ClusterConfig{
		Kind:       "Cluster",
		APIVersion: "kind.x-k8s.io/v1alpha4",
		Name:       opts.ClusterName,
	}

	// Build control plane nodes
	for i := 0; i < opts.NumControlPlanes; i++ {
		node := NodeConfig{
			Role: "control-plane",
		}
		if opts.KubernetesVersion != "" {
			node.Image = kindNodeImage(opts.KubernetesVersion)
		}
		// Port mappings only on the first control plane
		if i == 0 && len(opts.PortMappings) > 0 {
			node.ExtraPortMappings = opts.PortMappings
		}
		if len(opts.ExtraMounts) > 0 {
			node.ExtraMounts = opts.ExtraMounts
		}
		if len(opts.Labels) > 0 {
			node.Labels = opts.Labels
		}
		cfg.Nodes = append(cfg.Nodes, node)
	}

	// Build worker nodes
	for i := 0; i < opts.NumWorkers; i++ {
		node := NodeConfig{
			Role: "worker",
		}
		if opts.KubernetesVersion != "" {
			node.Image = kindNodeImage(opts.KubernetesVersion)
		}
		if len(opts.ExtraMounts) > 0 {
			node.ExtraMounts = opts.ExtraMounts
		}
		if len(opts.Labels) > 0 {
			node.Labels = opts.Labels
		}
		cfg.Nodes = append(cfg.Nodes, node)
	}

	// Networking
	if opts.PodSubnet != "" || opts.ServiceSubnet != "" || opts.DisableDefaultCNI ||
		opts.IPFamily != "" || opts.KubeProxyMode != "" || opts.APIServerPort != 0 {
		cfg.Networking = &NetworkConfig{
			PodSubnet:         opts.PodSubnet,
			ServiceSubnet:     opts.ServiceSubnet,
			DisableDefaultCNI: opts.DisableDefaultCNI,
			IPFamily:          opts.IPFamily,
			KubeProxyMode:     opts.KubeProxyMode,
			APIServerPort:     opts.APIServerPort,
		}
	}

	// Containerd patches
	if len(opts.ContainerdPatches) > 0 {
		cfg.ContainerdConfigPatches = opts.ContainerdPatches
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshaling config to YAML: %w", err)
	}

	return string(data), nil
}

// kindNodeImage returns the kindest/node image for a given Kubernetes version.
func kindNodeImage(version string) string {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return fmt.Sprintf("kindest/node:%s", version)
}

// ValidateConfig performs basic validation on a Kind cluster config YAML.
func ValidateConfig(configYAML string) error {
	var cfg ClusterConfig
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if cfg.Kind != "Cluster" {
		return fmt.Errorf("expected kind 'Cluster', got %q", cfg.Kind)
	}
	if cfg.APIVersion != "kind.x-k8s.io/v1alpha4" {
		return fmt.Errorf("expected apiVersion 'kind.x-k8s.io/v1alpha4', got %q", cfg.APIVersion)
	}

	hasControlPlane := false
	for _, node := range cfg.Nodes {
		if node.Role == "control-plane" {
			hasControlPlane = true
		}
		if node.Role != "control-plane" && node.Role != "worker" {
			return fmt.Errorf("invalid node role %q; must be 'control-plane' or 'worker'", node.Role)
		}
	}
	if len(cfg.Nodes) > 0 && !hasControlPlane {
		return fmt.Errorf("at least one control-plane node is required")
	}

	return nil
}
