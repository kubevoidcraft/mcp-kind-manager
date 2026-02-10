package main

import (
	"log/slog"
	"os"

	"github.com/kubevoidcraft/mcp-kind-manager/internal/runtime"
	"github.com/kubevoidcraft/mcp-kind-manager/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel(),
	}))

	slog.SetDefault(logger)

	logger.Info("starting mcp-kind-manager",
		"version", Version,
		"os", runtime.DetectOS().OS,
		"arch", runtime.DetectOS().Arch,
	)

	s := server.NewMCPServer(
		"mcp-kind-manager",
		Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	reg := tools.NewRegistry(logger)
	reg.RegisterAll(s)

	logger.Info("serving over stdio")
	if err := server.ServeStdio(s); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func logLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
