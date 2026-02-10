package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (r *Registry) registerClusterTools(s *server.MCPServer) {
	createTool := mcp.NewTool("create_cluster",
		mcp.WithDescription(
			"Create a Kind cluster from a configuration YAML. "+
				"Use 'generate_cluster_config' first to generate and review the config YAML."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster to create"),
		),
		mcp.WithString("config_yaml",
			mcp.Required(),
			mcp.Description("The Kind cluster configuration YAML (from generate_cluster_config)"),
		),
	)
	s.AddTool(createTool, r.handleCreateCluster)

	deleteTool := mcp.NewTool("delete_cluster",
		mcp.WithDescription("Delete a Kind cluster by name."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster to delete"),
		),
	)
	s.AddTool(deleteTool, r.handleDeleteCluster)

	listTool := mcp.NewTool("list_clusters",
		mcp.WithDescription("List all Kind clusters currently running."),
	)
	s.AddTool(listTool, r.handleListClusters)

	statusTool := mcp.NewTool("get_cluster_status",
		mcp.WithDescription(
			"Get the status of a Kind cluster, including node names, roles, and container states."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Kind cluster"),
		),
	)
	s.AddTool(statusTool, r.handleGetClusterStatus)
}

func (r *Registry) handleCreateCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Info("tool called: create_cluster")
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'name' is required"), nil
	}
	configYAML, err := request.RequireString("config_yaml")
	if err != nil {
		return mcp.NewToolResultError("parameter 'config_yaml' is required"), nil
	}

	mgr := r.kindManager(ctx)
	output, err := mgr.CreateCluster(ctx, name, configYAML)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create cluster: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Cluster %q created successfully.\n\n%s", name, output)), nil
}

func (r *Registry) handleDeleteCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Info("tool called: delete_cluster")
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'name' is required"), nil
	}

	mgr := r.kindManager(ctx)
	output, err := mgr.DeleteCluster(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete cluster: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Cluster %q deleted successfully.\n\n%s", name, output)), nil
}

func (r *Registry) handleListClusters(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Debug("tool called: list_clusters")
	mgr := r.kindManager(ctx)
	clusters, err := mgr.ListClusters(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list clusters: %v", err)), nil
	}

	if len(clusters) == 0 {
		return mcp.NewToolResultText("No Kind clusters found."), nil
	}

	result := map[string]any{
		"clusters": clusters,
		"count":    len(clusters),
	}
	return jsonResult(result)
}

func (r *Registry) handleGetClusterStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r.logger.Debug("tool called: get_cluster_status")
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("parameter 'name' is required"), nil
	}

	mgr := r.kindManager(ctx)
	status, err := mgr.GetClusterStatus(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get cluster status: %v", err)), nil
	}

	return jsonResult(status)
}
