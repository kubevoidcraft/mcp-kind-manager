package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/kind"
	rtdetect "github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Registry holds shared dependencies for tool handlers.
type Registry struct {
	logger   *slog.Logger
	runner   rtdetect.CommandRunner
	detector *rtdetect.Detector
}

// NewRegistry creates a new tool Registry.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	runner := &rtdetect.ExecCommandRunner{}
	return &Registry{
		logger:   logger,
		runner:   runner,
		detector: rtdetect.NewDetector(runner),
	}
}

// RegisterAll registers all tools on the given MCP server.
func (r *Registry) RegisterAll(s *server.MCPServer) {
	r.registerDetectTools(s)
	r.registerConfigTools(s)
	r.registerClusterTools(s)
	r.registerKubeconfigTools(s)
	r.registerRegistryTools(s)
}

func (r *Registry) runtimeInfo(ctx context.Context) rtdetect.RuntimeInfo {
	return r.detector.Detect(ctx)
}

func (r *Registry) kindManager(ctx context.Context) *kind.Manager {
	ri := r.runtimeInfo(ctx)
	return kind.NewManager(r.runner, ri, r.logger)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
