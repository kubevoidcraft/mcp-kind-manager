package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/registry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (r *Registry) registerRegistryTools(s *server.MCPServer) {
	credTool := mcp.NewTool("detect_credentials",
		mcp.WithDescription(
			"Discover registry credential files on the host. "+
				"Searches Docker and Podman credential stores based on the detected runtime and OS. "+
				"Returns the credential file path, registries with stored credentials, "+
				"and whether credentials are inline or managed by a credential helper."),
	)
	s.AddTool(credTool, r.handleDetectCredentials)

	mirrorTool := mcp.NewTool("configure_registry_mirrors",
		mcp.WithDescription(
			"Configure containerd registry mirrors on a running Kind cluster. "+
				"Writes hosts.toml files to each node to redirect image pulls "+
				"from specified registries to local mirror/proxy endpoints. "+
				"Restarts containerd on all nodes after applying configuration."),
		mcp.WithString("cluster_name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster to configure"),
		),
		mcp.WithString("overrides",
			mcp.Required(),
			mcp.Description(
				"JSON array of registry overrides. Each object has 'original' (source registry, e.g. 'docker.io') "+
					"and 'mirror' (mirror URL, e.g. 'http://my-proxy:5000'). "+
					"Example: [{\"original\":\"docker.io\",\"mirror\":\"http://localhost:5000\"}]"),
		),
		mcp.WithBoolean("include_credentials",
			mcp.Description("Also mount discovered host credentials into the cluster nodes. Default: false."),
		),
	)
	s.AddTool(mirrorTool, r.handleConfigureRegistryMirrors)
}

func (r *Registry) handleDetectCredentials(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Debug("tool called: detect_credentials")
	ri := r.runtimeInfo(ctx)
	credInfo, err := registry.FindCredentials(ri)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("credential discovery failed: %v", err)), nil
	}
	return jsonResult(credInfo)
}

func (r *Registry) handleConfigureRegistryMirrors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Info("tool called: configure_registry_mirrors")
	clusterName, err := request.RequireString("cluster_name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'cluster_name' is required"), nil
	}

	overridesJSON, err := request.RequireString("overrides")
	if err != nil {
		return mcp.NewToolResultError("parameter 'overrides' is required"), nil
	}

	var overrides []registry.RegistryOverride
	if err := json.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"invalid 'overrides' JSON: %v. Expected: [{\"original\":\"docker.io\",\"mirror\":\"http://localhost:5000\"}]",
			err)), nil
	}
	if len(overrides) == 0 {
		return mcp.NewToolResultError("at least one registry override is required"), nil
	}

	var credInfo *registry.CredentialInfo
	if val, ok := request.GetArguments()["include_credentials"].(bool); ok && val {
		ri := r.runtimeInfo(ctx)
		credInfo, _ = registry.FindCredentials(ri)
	}

	mirrorCfg, err := registry.GenerateMirrorConfig(overrides, credInfo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate mirror config: %v", err)), nil
	}

	mgr := r.kindManager(ctx)
	results, err := registry.ApplyMirrorConfig(ctx, mgr, clusterName, mirrorCfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to apply mirror config: %v", err)), nil
	}

	output := fmt.Sprintf("Registry mirror configuration applied to cluster %q.\n\nResults:\n%s",
		clusterName, strings.Join(results, "\n"))

	return mcp.NewToolResultText(output), nil
}
