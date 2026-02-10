package kind

import (
	"fmt"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

// PortMapping describes a host-to-container port mapping for Kind extraPortMappings.
type PortMapping struct {
	HostPort      int    `yaml:"hostPort" json:"host_port"`
	ContainerPort int    `yaml:"containerPort" json:"container_port"`
	ListenAddress string `yaml:"listenAddress,omitempty" json:"listen_address,omitempty"`
	Protocol      string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// NetworkAdvice provides guidance on exposing applications from Kind based on the runtime.
type NetworkAdvice struct {
	ListenAddress        string `json:"listen_address"`
	SupportsPortMapping  bool   `json:"supports_port_mapping"`
	RequiresExtraConfig  bool   `json:"requires_extra_config"`
	Notes                string `json:"notes"`
	RecommendedPortRange string `json:"recommended_port_range"`
}

// DetectNetworkConfig returns network exposure advice based on the runtime info.
func DetectNetworkConfig(ri rtdetect.RuntimeInfo) NetworkAdvice {
	advice := NetworkAdvice{
		ListenAddress:        "127.0.0.1",
		SupportsPortMapping:  true,
		RequiresExtraConfig:  false,
		RecommendedPortRange: "30000-32767",
	}

	switch ri.Backend {
	case rtdetect.BackendDockerDesktop:
		advice.Notes = "Docker Desktop forwards container ports to the host automatically. " +
			"Use extraPortMappings in the Kind config to map NodePorts to host ports. " +
			"Bind to 127.0.0.1 for local-only access or 0.0.0.0 for LAN access."

	case rtdetect.BackendColima:
		advice.Notes = "Colima forwards ports from the VM to the macOS host. " +
			"extraPortMappings work seamlessly. Bind to 127.0.0.1 for local access. " +
			"For LAN access, use 0.0.0.0 and ensure Colima network settings allow it."

	case rtdetect.BackendWSL:
		advice.Notes = "WSL2 automatically forwards localhost ports from the Linux VM to Windows. " +
			"extraPortMappings on 127.0.0.1 are reachable from Windows host. " +
			"For LAN access, Windows firewall rules may need adjustment."

	case rtdetect.BackendPodmanMachine:
		advice.Notes = "Podman Machine forwards ports from the VM to the host. " +
			"extraPortMappings work with rootful Podman Machine. " +
			"For rootless mode, additional port forwarding may be required."
		if ri.OS.OS == "darwin" || ri.OS.OS == "windows" {
			advice.RequiresExtraConfig = true
		}

	case rtdetect.BackendNative:
		if ri.OS.OS == "linux" {
			advice.ListenAddress = "0.0.0.0"
			advice.Notes = "Native Linux: containers are directly accessible. " +
				"extraPortMappings bind to the host network interface. " +
				"Container IPs are also directly reachable from the host."
		}

	case rtdetect.BackendRancherDesktop:
		advice.Notes = "Rancher Desktop forwards container ports similar to Docker Desktop. " +
			"extraPortMappings work as expected."

	case rtdetect.BackendLima:
		advice.Notes = "Lima VMs forward ports to the macOS host. " +
			"extraPortMappings should work. Check Lima port forwarding configuration if issues arise."

	default:
		advice.Notes = "Unknown backend. extraPortMappings with 127.0.0.1 is a safe default."
	}

	return advice
}

// DefaultPortMappings returns commonly useful port mappings for Kind clusters.
func DefaultPortMappings(listenAddr string) []PortMapping {
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	return []PortMapping{
		{
			HostPort:      80,
			ContainerPort: 80,
			ListenAddress: listenAddr,
			Protocol:      "TCP",
		},
		{
			HostPort:      443,
			ContainerPort: 443,
			ListenAddress: listenAddr,
			Protocol:      "TCP",
		},
	}
}

// FormatNetworkAdvice returns a human-readable summary of the network advice.
func FormatNetworkAdvice(advice NetworkAdvice) string {
	result := fmt.Sprintf("Listen Address: %s\nPort Mapping Supported: %t\n",
		advice.ListenAddress, advice.SupportsPortMapping)
	if advice.RequiresExtraConfig {
		result += "Warning: Extra configuration may be required for port forwarding.\n"
	}
	result += fmt.Sprintf("Recommended NodePort Range: %s\n", advice.RecommendedPortRange)
	result += fmt.Sprintf("Notes: %s", advice.Notes)
	return result
}
