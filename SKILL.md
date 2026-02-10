# SKILL.md — What This MCP Server Can Do

## Identity

**mcp-kind-manager** — an MCP server that manages Kind (Kubernetes in Docker) clusters. It detects your local container runtime environment and provides tools for the full cluster lifecycle, registry credential discovery, and containerd mirror configuration.

## Capabilities

### Environment Detection
- Detects host OS (Linux, macOS, Windows) and architecture
- Identifies container runtime: Docker or Podman
- Identifies runtime backend: Docker Desktop, Colima, WSL, Podman Machine, Lima, Rancher Desktop, or native Linux
- Provides network advice specific to the detected backend (listen addresses, port mapping support, extra config requirements)

### Cluster Configuration
- Generates Kind cluster config YAML with full control over:
  - Number of control-plane and worker nodes (multi-node, HA)
  - Kubernetes version selection (kindest/node image)
  - Custom networking: pod/service subnets, IP family (IPv4/IPv6/dual), kube-proxy mode, API server port pinning
  - Disable default CNI (for custom CNI like Cilium)
  - Extra port mappings and host mounts
  - Containerd config patches
- Returns YAML for human review before cluster creation

### Cluster Lifecycle
- **Create** clusters from config YAML
- **Delete** clusters by name
- **List** all running Kind clusters
- **Get status** of a cluster — node names, roles (control-plane/worker), container states (running/stopped/etc.)
- **Get kubeconfig** — external (localhost) or internal (container IP) variants

### Registry Credentials
- Auto-discovers Docker and Podman credential files across platform-specific paths
- Detects whether credentials are inline or managed by a credential helper (e.g., `desktop`, `osxkeychain`)
- Reports which registries have stored credentials
- Can mount credential files into cluster nodes

### Registry Mirrors
- Configures containerd `hosts.toml` on all cluster nodes to redirect image pulls through a local mirror/proxy
- Supports multiple registry overrides (e.g., docker.io → local-proxy:5000, ghcr.io → local-proxy:5001)
- Handles HTTP mirrors with automatic `skip_verify` for plain HTTP endpoints
- Restarts containerd on all nodes after configuration

## Workflow

1. Call `detect_environment` to understand the runtime context
2. Call `generate_cluster_config` with desired parameters — review the YAML
3. Call `create_cluster` with the reviewed YAML
4. Optionally call `configure_registry_mirrors` to set up image pull proxies
5. Call `get_kubeconfig` to interact with the cluster via kubectl

## Limitations

- Requires `kind` CLI installed and in PATH — this server wraps the CLI, it does not embed the Kind library
- Requires Docker or Podman running
- On macOS with Docker Desktop: binding to privileged ports (80, 443) may fail if the `vmnetd` helper socket is not present — use ports ≥ 1024 instead
- Registry mirror configuration applies to a running cluster — if the cluster is recreated, mirrors must be reconfigured
- Credential helper-managed credentials (e.g., macOS Keychain) cannot be directly mounted; only inline auth configs are mountable
