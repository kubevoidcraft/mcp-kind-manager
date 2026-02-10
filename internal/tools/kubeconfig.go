package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (r *Registry) registerKubeconfigTools(s *server.MCPServer) {
	tool := mcp.NewTool("get_kubeconfig",
		mcp.WithDescription(
			"Get the kubeconfig for a Kind cluster. "+
				"Returns the kubeconfig YAML that can be used with kubectl."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster"),
		),
		mcp.WithBoolean("internal",
			mcp.Description("Get internal kubeconfig (container IPs instead of localhost). Default: false."),
		),
	)
	s.AddTool(tool, r.handleGetKubeconfig)
}

func (r *Registry) handleGetKubeconfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Debug("tool called: get_kubeconfig")
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'name' is required"), nil
	}

	internal := false
	if val, ok := request.GetArguments()["internal"].(bool); ok {
		internal = val
	}

	mgr := r.kindManager(ctx)
	kubeconfig, err := mgr.GetKubeconfig(ctx, name, internal)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get kubeconfig: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Kubeconfig for cluster %q:\n\n```yaml\n%s```", name, kubeconfig)), nil
}
