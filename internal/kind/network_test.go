package kind

import (
	"testing"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

func TestDetectNetworkConfig_DockerDesktop(t *testing.T) {
	ri := rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimeDocker,
		Backend: rtdetect.BackendDockerDesktop,
	}
	advice := DetectNetworkConfig(ri)

	if advice.ListenAddress != "127.0.0.1" {
		t.Errorf("ListenAddress = %q, want 127.0.0.1", advice.ListenAddress)
	}
	if !advice.SupportsPortMapping {
		t.Error("expected SupportsPortMapping = true")
	}
	if advice.Notes == "" {
		t.Error("expected non-empty Notes")
	}
}

func TestDetectNetworkConfig_NativeLinux(t *testing.T) {
	ri := rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimeDocker,
		Backend: rtdetect.BackendNative,
		OS:      rtdetect.OSInfo{OS: "linux"},
	}
	advice := DetectNetworkConfig(ri)

	if advice.ListenAddress != "0.0.0.0" {
		t.Errorf("ListenAddress = %q, want 0.0.0.0", advice.ListenAddress)
	}
}

func TestDetectNetworkConfig_PodmanMachineMacOS(t *testing.T) {
	ri := rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimePodman,
		Backend: rtdetect.BackendPodmanMachine,
		OS:      rtdetect.OSInfo{OS: "darwin"},
	}
	advice := DetectNetworkConfig(ri)

	if !advice.RequiresExtraConfig {
		t.Error("expected RequiresExtraConfig = true for Podman Machine on macOS")
	}
}

func TestDetectNetworkConfig_WSL(t *testing.T) {
	ri := rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimeDocker,
		Backend: rtdetect.BackendWSL,
		OS:      rtdetect.OSInfo{OS: "linux"},
	}
	advice := DetectNetworkConfig(ri)

	if advice.ListenAddress != "127.0.0.1" {
		t.Errorf("ListenAddress = %q, want 127.0.0.1", advice.ListenAddress)
	}
	if advice.Notes == "" {
		t.Error("expected non-empty Notes")
	}
}

func TestDefaultPortMappings(t *testing.T) {
	mappings := DefaultPortMappings("")
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings[0].HostPort != 80 || mappings[0].ContainerPort != 80 {
		t.Errorf("first mapping = %+v, want 80:80", mappings[0])
	}
	if mappings[0].ListenAddress != "127.0.0.1" {
		t.Errorf("default listen address = %q, want 127.0.0.1", mappings[0].ListenAddress)
	}
	if mappings[1].HostPort != 443 {
		t.Errorf("second mapping host port = %d, want 443", mappings[1].HostPort)
	}
}

func TestDefaultPortMappings_CustomAddr(t *testing.T) {
	mappings := DefaultPortMappings("0.0.0.0")
	if mappings[0].ListenAddress != "0.0.0.0" {
		t.Errorf("listen address = %q, want 0.0.0.0", mappings[0].ListenAddress)
	}
}

func TestFormatNetworkAdvice(t *testing.T) {
	advice := NetworkAdvice{
		ListenAddress:        "127.0.0.1",
		SupportsPortMapping:  true,
		RequiresExtraConfig:  true,
		RecommendedPortRange: "30000-32767",
		Notes:                "test note",
	}
	out := FormatNetworkAdvice(advice)

	if out == "" {
		t.Error("expected non-empty output")
	}
}
