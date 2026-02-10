# AGENT.md — Developer Guide for AI Agents

## Project Overview

**mcp-kind-manager** is an MCP (Model Context Protocol) server written in Go that manages Kind Kubernetes clusters. It communicates over stdio using the `mcp-go` SDK and shells out to the `kind` CLI and container runtime (`docker`/`podman`) for all operations.

## Build & Test

```bash
make build           # Build binary to bin/mcp-kind-manager
make test            # Run tests with race detector + coverage
make test-verbose    # Verbose test output
make lint            # go vet + golangci-lint
make fmt             # gofmt + gofumpt
```

## Architecture

```
cmd/mcp-kind-manager/main.go    Entrypoint — creates MCP server, registers tools, serves stdio
internal/
  runtime/                       OS + container runtime detection (Docker/Podman, backend identification)
  kind/                          Kind cluster config generation, lifecycle management, networking advice
  registry/                      Credential discovery + containerd mirror configuration
  tools/                         MCP tool definitions, parameter parsing, handler wiring
```

### Key Design Patterns

- **CLI wrapping**: `kind.Manager` wraps the `kind` CLI via `runtime.CommandRunner` interface. No SDK dependency — uses `os/exec` under the hood.
- **Runtime detection**: `runtime.Detector` probes Docker/Podman and identifies the backend (Docker Desktop, Colima, WSL, etc.) for environment-specific advice.
- **Two-step config**: Config generation (`GenerateConfig`) and cluster creation (`CreateCluster`) are separate — the config YAML is returned for review before being passed to create.
- **Testability**: All external commands go through `CommandRunner` interface. Tests use `mockRunner` to simulate CLI output without real clusters.

### Dependency Graph

```
tools → kind, registry, runtime
registry → kind (for Mount type), runtime (for credential paths)
kind → runtime (for CommandRunner, RuntimeInfo)
runtime → (no internal deps)
```

## Key Interfaces

### `runtime.CommandRunner`
Abstracts `os/exec` for testability. Has `Run(ctx, name, args...) ([]byte, error)` and `LookPath(name) (string, error)`.

### `kind.Manager`
Created with `NewManager(runner, runtimeInfo, logger)`. Methods: `CreateCluster`, `DeleteCluster`, `ListClusters`, `GetKubeconfig`, `GetClusterStatus`, `ExecOnNode`, `GetClusterNodes`.

### `tools.Registry`
Holds shared deps (logger, runner, detector). `RegisterAll(s)` wires all 9 MCP tools onto the server.

## MCP Tools (9 total)

| Tool | Handler | Package |
|------|---------|---------|
| `detect_environment` | `handleDetectEnvironment` | tools/detect.go |
| `generate_cluster_config` | `handleGenerateClusterConfig` | tools/detect.go |
| `create_cluster` | `handleCreateCluster` | tools/cluster.go |
| `delete_cluster` | `handleDeleteCluster` | tools/cluster.go |
| `list_clusters` | `handleListClusters` | tools/cluster.go |
| `get_cluster_status` | `handleGetClusterStatus` | tools/cluster.go |
| `get_kubeconfig` | `handleGetKubeconfig` | tools/kubeconfig.go |
| `detect_credentials` | `handleDetectCredentials` | tools/registry_tools.go |
| `configure_registry_mirrors` | `handleConfigureRegistryMirrors` | tools/registry_tools.go |

## Testing Conventions

- Tests live alongside source: `foo.go` → `foo_test.go`
- Use `mockRunner` for CLI simulation (defined in `manager_test.go`)
- `matchArgs` does prefix-matching on command args for flexibility
- No real clusters in tests — everything is mocked

## Common Tasks

### Adding a new MCP tool
1. Add handler method on `*Registry` in the appropriate `tools/*.go` file
2. Register it in the corresponding `register*Tools` method
3. Add tests for parameter validation and happy path

### Modifying cluster config generation
Edit `internal/kind/config.go` — `ConfigOptions` struct and `GenerateConfig()` function. Update `config_test.go` and the `handleGenerateClusterConfig` handler in `tools/detect.go`.

### Adding a new runtime backend
Add a constant to `internal/runtime/detect.go`, update `detectDockerBackend` or `detectPodmanBackend`, and add a case in `kind/network.go` `DetectNetworkConfig`.

## Environment

- Go 1.24+
- Requires `kind` CLI in PATH
- Requires `docker` or `podman` in PATH
- Env var `LOG_LEVEL` controls log verbosity (debug/info/warn/error)

## Known Constraints

- **Privileged ports**: On macOS with Docker Desktop, binding to ports 80/443 requires the `vmnetd` helper socket (`/var/run/com.docker.vmnetd.sock`). If missing, use ports ≥1024 in `extraPortMappings`.
- **Docker socket**: Docker Desktop on macOS may use `~/.docker/run/docker.sock` instead of `/var/run/docker.sock`. The runtime detector handles this.
- **Podman**: When using Podman, `--runtime podman` is automatically appended to `kind` CLI calls.
