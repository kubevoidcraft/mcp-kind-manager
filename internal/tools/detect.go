package tools

import (
	"context"
	"fmt"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/kind"
	"github.com/kubevoidcraft/mcp-kind-manager/internal/registry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (r *Registry) registerDetectTools(s *server.MCPServer) {
	detectTool := mcp.NewTool("detect_environment",
		mcp.WithDescription(
			"Detect the host operating system, container runtime (Docker/Podman), "+
				"runtime backend (Docker Desktop, Colima, WSL, Podman Machine, native), "+
				"and provide network configuration advice for exposing applications from Kind clusters."),
	)
	s.AddTool(detectTool, r.handleDetectEnvironment)
}

func (r *Registry) handleDetectEnvironment(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Debug("tool called: detect_environment")
	ri := r.runtimeInfo(ctx)
	networkAdvice := kind.DetectNetworkConfig(ri)

	result := map[string]any{
		"os":             ri.OS,
		"runtime":        ri.Runtime,
		"backend":        ri.Backend,
		"version":        ri.Version,
		"socket_path":    ri.SocketPath,
		"available":      ri.Available,
		"network_advice": networkAdvice,
	}
	if ri.Error != "" {
		result["error"] = ri.Error
	}

	return jsonResult(result)
}

func (r *Registry) registerConfigTools(s *server.MCPServer) {
	configTool := mcp.NewTool("generate_cluster_config",
		mcp.WithDescription(
			"Generate a Kind cluster configuration YAML. Returns the YAML for review before creation. "+
				"Supports multi-node clusters, port mappings, credential mounting, registry mirror overrides, "+
				"custom networking, and containerd configuration."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster"),
		),
		mcp.WithNumber("workers",
			mcp.Description("Number of worker nodes (default: 0, single control-plane only)"),
		),
		mcp.WithNumber("control_planes",
			mcp.Description("Number of control plane nodes (default: 1; >1 for HA)"),
		),
		mcp.WithString("kubernetes_version",
			mcp.Description("Kubernetes version for kindest/node image (e.g., '1.31.0'). Leave empty for Kind default."),
		),
		mcp.WithBoolean("mount_credentials",
			mcp.Description("Auto-detect and mount registry credentials to cluster nodes"),
		),
		mcp.WithString("pod_subnet",
			mcp.Description("Custom pod subnet CIDR (e.g., '10.244.0.0/16')"),
		),
		mcp.WithString("service_subnet",
			mcp.Description("Custom service subnet CIDR (e.g., '10.96.0.0/12')"),
		),
		mcp.WithBoolean("disable_default_cni",
			mcp.Description("Disable the default CNI (for installing a custom CNI like Cilium)"),
		),
		mcp.WithString("ip_family",
			mcp.Description("IP family: 'ipv4', 'ipv6', or 'dual'"),
		),
		mcp.WithString("kube_proxy_mode",
			mcp.Description("Kube-proxy mode: 'iptables', 'ipvs', 'nftables', or 'none'"),
		),
		mcp.WithNumber("api_server_port",
			mcp.Description("Pin the API server to a specific host port (e.g., 6443). Default: random."),
		),
	)
	s.AddTool(configTool, r.handleGenerateClusterConfig)
}

func (r *Registry) handleGenerateClusterConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Info("tool called: generate_cluster_config")
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'name' is required"), nil
	}

	ri := r.runtimeInfo(ctx)

	opts := kind.ConfigOptions{
		ClusterName:      name,
		NumControlPlanes: 1,
	}

	if workers, err := request.RequireFloat("workers"); err == nil {
		opts.NumWorkers = int(workers)
	}
	if cp, err := request.RequireFloat("control_planes"); err == nil && int(cp) > 0 {
		opts.NumControlPlanes = int(cp)
	}
	if version, err := request.RequireString("kubernetes_version"); err == nil {
		opts.KubernetesVersion = version
	}
	if subnet, err := request.RequireString("pod_subnet"); err == nil {
		opts.PodSubnet = subnet
	}
	if subnet, err := request.RequireString("service_subnet"); err == nil {
		opts.ServiceSubnet = subnet
	}
	if ipFamily, err := request.RequireString("ip_family"); err == nil {
		opts.IPFamily = ipFamily
	}
	if proxyMode, err := request.RequireString("kube_proxy_mode"); err == nil {
		opts.KubeProxyMode = proxyMode
	}
	if port, err := request.RequireFloat("api_server_port"); err == nil && int(port) > 0 {
		opts.APIServerPort = int(port)
	}
	if val, ok := request.GetArguments()["disable_default_cni"].(bool); ok {
		opts.DisableDefaultCNI = val
	}

	// Mount credentials if requested
	if val, ok := request.GetArguments()["mount_credentials"].(bool); ok && val {
		credInfo, err := registry.FindCredentials(ri)
		if err != nil {
			r.logger.Warn("credential discovery failed", "error", err)
		} else {
			opts.ExtraMounts = append(opts.ExtraMounts, kind.Mount{
				HostPath:      credInfo.FilePath,
				ContainerPath: credInfo.MountPath,
				ReadOnly:      true,
			})
		}
	}

	configYAML, err := kind.GenerateConfig(opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate config: %v", err)), nil
	}

	output := fmt.Sprintf("Generated Kind cluster config for %q:\n\n```yaml\n%s```\n\n"+
		"Review the configuration above, then use the 'create_cluster' tool with this YAML to create the cluster.",
		name, configYAML)

	return mcp.NewToolResultText(output), nil
}
