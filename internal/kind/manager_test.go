package kind

import (
	"context"
	"fmt"
	"strings"
	"testing"

	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
)

type mockRunner struct {
	runs []runCall
}

type runCall struct {
	name string
	args []string
	out  []byte
	err  error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	for _, r := range m.runs {
		if r.name == name && matchArgs(r.args, args) {
			return r.out, r.err
		}
	}
	return nil, fmt.Errorf("no mock for %s %v", name, args)
}

func (m *mockRunner) LookPath(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func matchArgs(want, got []string) bool {
	if len(want) == 0 {
		return true
	}
	if len(want) > len(got) {
		return false
	}
	for i, w := range want {
		if w != got[i] {
			return false
		}
	}
	return true
}

func newDockerManager(runner *mockRunner) *Manager {
	return NewManager(runner, rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimeDocker,
	}, nil)
}

func newPodmanManager(runner *mockRunner) *Manager {
	return NewManager(runner, rtdetect.RuntimeInfo{
		Runtime: rtdetect.RuntimePodman,
	}, nil)
}

func TestCreateCluster(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"create", "cluster"}, out: []byte("Creating cluster\n")},
		},
	}

	cfg, _ := GenerateConfig(ConfigOptions{
		ClusterName:      "test",
		NumControlPlanes: 1,
	})

	mgr := newDockerManager(runner)
	out, err := mgr.CreateCluster(context.Background(), "test", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Creating cluster") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestCreateCluster_EmptyName(t *testing.T) {
	mgr := newDockerManager(&mockRunner{})
	_, err := mgr.CreateCluster(context.Background(), "", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateCluster_InvalidConfig(t *testing.T) {
	mgr := newDockerManager(&mockRunner{})
	_, err := mgr.CreateCluster(context.Background(), "test", "not valid yaml [[[")
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestDeleteCluster(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"delete", "cluster"}, out: []byte("Deleting cluster\n")},
		},
	}

	mgr := newDockerManager(runner)
	out, err := mgr.DeleteCluster(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Deleting") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestDeleteCluster_EmptyName(t *testing.T) {
	mgr := newDockerManager(&mockRunner{})
	_, err := mgr.DeleteCluster(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestListClusters(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"get", "clusters"}, out: []byte("cluster1\ncluster2\n")},
		},
	}

	mgr := newDockerManager(runner)
	clusters, err := mgr.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	if clusters[0] != "cluster1" || clusters[1] != "cluster2" {
		t.Errorf("clusters = %v", clusters)
	}
}

func TestListClusters_Empty(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"get", "clusters"}, out: []byte("No kind clusters found.\n")},
		},
	}

	mgr := newDockerManager(runner)
	clusters, err := mgr.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestGetKubeconfig(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"get", "kubeconfig"}, out: []byte("apiVersion: v1\nclusters: []\n")},
		},
	}

	mgr := newDockerManager(runner)
	kc, err := mgr.GetKubeconfig(context.Background(), "test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(kc, "apiVersion") {
		t.Errorf("unexpected kubeconfig: %s", kc)
	}
}

func TestGetKubeconfig_EmptyName(t *testing.T) {
	mgr := newDockerManager(&mockRunner{})
	_, err := mgr.GetKubeconfig(context.Background(), "", false)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestGetClusterStatus(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"get", "nodes"}, out: []byte("test-control-plane\ntest-worker\n")},
			{name: "docker", args: []string{"inspect"}, out: []byte("running\n")},
		},
	}

	mgr := newDockerManager(runner)
	status, err := mgr.GetClusterStatus(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Name != "test" {
		t.Errorf("Name = %q", status.Name)
	}
	if len(status.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(status.Nodes))
	}
	if status.Nodes[0].Role != "control-plane" {
		t.Errorf("first node role = %q, want control-plane", status.Nodes[0].Role)
	}
	if status.Nodes[1].Role != "worker" {
		t.Errorf("second node role = %q, want worker", status.Nodes[1].Role)
	}
}

func TestPodmanManager_KindArgs(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"--runtime", "podman", "get", "clusters"}, out: []byte("")},
		},
	}

	mgr := newPodmanManager(runner)
	_, err := mgr.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecOnNode(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "docker", args: []string{"exec", "test-control-plane"}, out: []byte("ok\n")},
		},
	}

	mgr := newDockerManager(runner)
	out, err := mgr.ExecOnNode(context.Background(), "test-control-plane", []string{"echo", "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "ok" {
		t.Errorf("output = %q", out)
	}
}

func TestGetClusterNodes(t *testing.T) {
	runner := &mockRunner{
		runs: []runCall{
			{name: "kind", args: []string{"get", "nodes"}, out: []byte("node1\nnode2\nnode3\n")},
		},
	}

	mgr := newDockerManager(runner)
	nodes, err := mgr.GetClusterNodes(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}
}
