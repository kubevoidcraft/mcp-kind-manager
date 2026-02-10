package kind

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateConfig_Simple(t *testing.T) {
	opts := ConfigOptions{
		ClusterName:      "test-cluster",
		NumControlPlanes: 1,
	}

	out, err := GenerateConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "kind: Cluster") {
		t.Error("missing kind: Cluster")
	}
	if !strings.Contains(out, "apiVersion: kind.x-k8s.io/v1alpha4") {
		t.Error("missing apiVersion")
	}
	if !strings.Contains(out, "name: test-cluster") {
		t.Error("missing cluster name")
	}
	if !strings.Contains(out, "role: control-plane") {
		t.Error("missing control-plane node")
	}
}

func TestGenerateConfig_MultiNode(t *testing.T) {
	opts := ConfigOptions{
		ClusterName:      "multi",
		NumControlPlanes: 3,
		NumWorkers:       2,
	}

	out, err := GenerateConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	cpCount := strings.Count(out, "role: control-plane")
	if cpCount != 3 {
		t.Errorf("control-plane count = %d, want 3", cpCount)
	}
	workerCount := strings.Count(out, "role: worker")
	if workerCount != 2 {
		t.Errorf("worker count = %d, want 2", workerCount)
	}
}

func TestGenerateConfig_KubernetesVersion(t *testing.T) {
	opts := ConfigOptions{
		ClusterName:       "versioned",
		NumControlPlanes:  1,
		KubernetesVersion: "1.31.0",
	}

	out, err := GenerateConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "kindest/node:v1.31.0") {
		t.Errorf("expected image kindest/node:v1.31.0 in output:\n%s", out)
	}
}

func TestGenerateConfig_Networking(t *testing.T) {
	opts := ConfigOptions{
		ClusterName:       "net",
		NumControlPlanes:  1,
		PodSubnet:         "10.244.0.0/16",
		ServiceSubnet:     "10.96.0.0/12",
		DisableDefaultCNI: true,
		IPFamily:          "dual",
		KubeProxyMode:     "ipvs",
		APIServerPort:     6443,
	}

	out, err := GenerateConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	var cfg ClusterConfig
	if err := yaml.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.Networking == nil {
		t.Fatal("Networking should not be nil")
	}
	if cfg.Networking.PodSubnet != "10.244.0.0/16" {
		t.Errorf("PodSubnet = %q", cfg.Networking.PodSubnet)
	}
	if cfg.Networking.ServiceSubnet != "10.96.0.0/12" {
		t.Errorf("ServiceSubnet = %q", cfg.Networking.ServiceSubnet)
	}
	if !cfg.Networking.DisableDefaultCNI {
		t.Error("DisableDefaultCNI should be true")
	}
	if cfg.Networking.IPFamily != "dual" {
		t.Errorf("IPFamily = %q", cfg.Networking.IPFamily)
	}
	if cfg.Networking.KubeProxyMode != "ipvs" {
		t.Errorf("KubeProxyMode = %q", cfg.Networking.KubeProxyMode)
	}
	if cfg.Networking.APIServerPort != 6443 {
		t.Errorf("APIServerPort = %d", cfg.Networking.APIServerPort)
	}
}

func TestGenerateConfig_Mounts(t *testing.T) {
	opts := ConfigOptions{
		ClusterName:      "mounts",
		NumControlPlanes: 1,
		NumWorkers:       1,
		ExtraMounts: []Mount{
			{HostPath: "/home/user/.docker/config.json", ContainerPath: "/var/lib/kubelet/config.json", ReadOnly: true},
		},
	}

	out, err := GenerateConfig(opts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "/home/user/.docker/config.json") {
		t.Error("missing hostPath in mount")
	}
	if !strings.Contains(out, "/var/lib/kubelet/config.json") {
		t.Error("missing containerPath in mount")
	}
}

func TestGenerateConfig_EmptyName(t *testing.T) {
	_, err := GenerateConfig(ConfigOptions{})
	if err == nil {
		t.Error("expected error for empty cluster name")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	out, _ := GenerateConfig(ConfigOptions{
		ClusterName:      "valid",
		NumControlPlanes: 1,
		NumWorkers:       2,
	})
	if err := ValidateConfig(out); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfig_InvalidYAML(t *testing.T) {
	if err := ValidateConfig("not: [valid: yaml"); err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateConfig_WrongKind(t *testing.T) {
	cfg := "kind: Pod\napiVersion: v1\n"
	if err := ValidateConfig(cfg); err == nil {
		t.Error("expected error for wrong kind")
	}
}

func TestValidateConfig_InvalidRole(t *testing.T) {
	cfg := "kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n  - role: bogus\n"
	if err := ValidateConfig(cfg); err == nil {
		t.Error("expected error for invalid role")
	}
}

func TestKindNodeImage(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"1.31.0", "kindest/node:v1.31.0"},
		{"v1.30.0", "kindest/node:v1.30.0"},
	}
	for _, tt := range tests {
		got := kindNodeImage(tt.version)
		if got != tt.want {
			t.Errorf("kindNodeImage(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}
