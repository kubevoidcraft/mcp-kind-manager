# mcp-kind-manager

An MCP (Model Context Protocol) server for managing [Kind](https://kind.sigs.k8s.io/) Kubernetes clusters. Detects your OS, container runtime, and backend — then provides tools for full cluster lifecycle management including multi-node configs, registry credentials, and containerd mirror configuration.

## Features

- **Environment detection** — OS, container runtime (Docker/Podman), backend (Docker Desktop, Colima, WSL, Podman Machine, Lima, Rancher Desktop, native)
- **Cluster config generation** — multi-node, HA control planes, custom networking (pod/service subnets, CNI, kube-proxy mode, IP family), port mappings
- **Full lifecycle** — create, delete, list, status, kubeconfig
- **Registry credentials** — auto-discover Docker/Podman credential files, mount into cluster nodes
- **Registry mirrors** — configure containerd `hosts.toml` on cluster nodes with BYOP (Bring Your Own Proxy) support
- **Network advice** — per-backend guidance on port forwarding and exposure

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/getting-started/installation)

## Install

```bash
# From source
go install github.com/kubevoidcraft/mcp-kind-manager/cmd/mcp-kind-manager@latest

# Or build locally
make build
./bin/mcp-kind-manager
```

## MCP Client Configuration

### Claude Desktop / VS Code

Add to your MCP client config:

```json
{
  "mcpServers": {
    "kind-manager": {
      "command": "/path/to/mcp-kind-manager",
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `detect_environment` | Detect OS, container runtime, backend, and network advice |
| `generate_cluster_config` | Generate Kind cluster config YAML for review |
| `create_cluster` | Create a Kind cluster from config YAML |
| `delete_cluster` | Delete a Kind cluster by name |
| `list_clusters` | List all running Kind clusters |
| `get_cluster_status` | Get node names, roles, and container states |
| `get_kubeconfig` | Get kubeconfig for a cluster |
| `detect_credentials` | Discover registry credential files on the host |
| `configure_registry_mirrors` | Configure containerd mirrors on a running cluster |

## Workflow

The server uses a **two-step config flow**:

1. **Generate** — call `generate_cluster_config` to produce YAML, review it
2. **Create** — pass the YAML to `create_cluster`

For registry mirrors, configure them **after** cluster creation:

1. Create the cluster
2. Call `configure_registry_mirrors` with your proxy endpoints

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |

## Development

```bash
make test          # Run tests with race detector + coverage
make test-verbose  # Verbose test output
make lint          # Run go vet + golangci-lint
make fmt           # Format code
make cover         # Generate HTML coverage report
make build-all     # Cross-compile for Linux, macOS, Windows
```

## Project Structure

```
cmd/mcp-kind-manager/     Entry point (stdio MCP server)
internal/
  runtime/                OS + container runtime detection
  kind/                   Kind cluster config, lifecycle, networking
  registry/               Credential discovery + containerd mirror config
  tools/                  MCP tool definitions + handlers
```

## License

MIT
