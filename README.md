# Gateway Monitoring Agent

A cross-platform daemon that monitors gateway status and sends periodic heartbeats to a backend API. Built in Go for maximum portability and minimal resource usage.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Build](#build)
- [Configuration](#configuration)
- [Running the Agent](#running-the-agent)
- [Installation as Service](#installation-as-service)
- [Payload Format](#payload-format)
- [Retry & Token Rotation](#retry--token-rotation)
- [Monitoring & Metrics](#monitoring--metrics)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Overview

The Gateway Monitoring Agent is a push-only daemon (no inbound ports) that:
- Sends periodic JSON heartbeats to a backend API over HTTPS
- Collects system metrics (CPU, memory, disk) when available
- Supports token rotation for zero-downtime credential updates
- Runs as a system service on Linux and Windows
- Produces static binaries with no external runtime dependencies

### Features

- **Cross-platform**: Linux (amd64/arm64) and Windows (amd64)
- **Push-only**: No listening ports, all communication is outbound HTTPS
- **Secure**: Token-based authentication with dual-token rotation
- **Resilient**: Automatic retry with exponential backoff on transient failures
- **Lightweight**: Single static binary (~6-7MB), minimal resource usage
- **Observable**: Structured JSON logging with configurable levels

### Supported Platforms

| Platform | Architecture | Target | Notes |
|----------|-------------|---------|-------|
| Ubuntu | amd64 | `linux_amd64` | Tested on Ubuntu 20.04+ |
| Raspberry Pi | arm64 | `linux_arm64` | Tested on Raspberry Pi 4 |
| Windows 11 | amd64 | `windows_amd64` | Requires Administrator |

## Quick Start

```bash
# Build for all platforms
make build-all

# Test with dry run (no network call)
./dist/gw-agent --config config.yaml --dry-run

# Send single heartbeat
./dist/gw-agent --config config.yaml --once

# Run continuously
./dist/gw-agent --config config.yaml

# Check version
./dist/gw-agent --print-version
```

See [QUICKSTART.md](QUICKSTART.md) for installation guide and [DEMO-GUIDE.md](DEMO-GUIDE.md) for demonstrations.

## Build

### Prerequisites

- Go 1.21 or later
- Make (or run Go commands directly)

### Basic Build

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Other targets
make test      # Run tests with race detector
make lint      # Run linters
make clean     # Remove build artifacts
make deps      # Download dependencies
```

Output binaries:
- `./dist/linux_amd64/gw-agent` - Ubuntu/Linux x86_64
- `./dist/linux_arm64/gw-agent` - Raspberry Pi ARM64
- `./dist/windows_amd64/gw-agent.exe` - Windows x86_64

### Advanced Build

<details>
<summary>Version Embedding, CI/CD, and Distribution</summary>

#### Version Embedding

```bash
# From git tags
VERSION=$(git describe --tags --always) make build-all

# Manual version
VERSION=1.0.0 COMMIT=$(git rev-parse --short HEAD) make build-all
```

Check version: `./gw-agent --print-version`

#### Binary Details

- **Static binaries**: CGO disabled, no external dependencies
- **Size**: ~6-7 MB (stripped with -s -w)
- **Cross-compilation**: Build from any platform for all targets

#### CI/CD Integration

```yaml
name: Build
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make build-all
      - uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: dist/
```

#### Distribution Methods

1. **Direct binary** - Upload to S3/CDN
2. **Package managers** - .deb, .rpm, .msi wrappers
3. **Containers** - Use scratch or distroless base
4. **Tar archives** - Bundle binary + scripts + config

```bash
tar czf gw-agent-linux-amd64.tar.gz \
  -C dist/linux_amd64 gw-agent \
  -C ../../scripts gw-agent.service install-linux.sh \
  -C .. config.example.yaml README.md
```

#### Dependencies

```
github.com/shirou/gopsutil/v3  # System metrics
gopkg.in/yaml.v3               # YAML parsing
```

</details>

## Configuration

Default config locations:
- Linux: `/etc/gw-agent/config.yaml`
- Windows: `C:\ProgramData\GWAgent\config.yaml`

### Example Configuration

```yaml
# Required fields
uuid: "gateway-unique-id-123"
client_id: "client_name"
site_id: "site_name"
api_url: "https://api.example.com/v1/heartbeat"

# Authentication (required)
auth:
  token_current: "your-current-api-token"
  token_grace: "optional-old-token-for-rotation"  # For zero-downtime rotation

# Platform detection (optional)
platform:
  platform_override: "ubuntu"  # raspberry_pi, ubuntu, windows, vm, linux

# Timing intervals (optional, defaults shown)
intervals:
  heartbeat_seconds: 60   # How often to send heartbeats
  compute_seconds: 120    # How often to refresh compute metrics

# TLS configuration (optional)
tls:
  ca_bundle_path: "/path/to/ca-bundle.pem"  # Custom CA bundle
  insecure_skip_verify: false               # Keep false in production
```

**Required fields**: `uuid`, `client_id`, `site_id`, `api_url`, `auth.token_current`

**Platform auto-detection**: `raspberry_pi`, `ubuntu`, `windows`, `vm`, or `linux` (fallback)

## Running the Agent

### Command-Line Flags

```bash
gw-agent [flags]

Flags:
  --config string       Config file path (default: /etc/gw-agent/config.yaml)
  --once                Send one heartbeat and exit
  --log-level string    Log level: debug, info, warn, error (default: info)
  --dry-run             Print payload without sending
  --print-version       Print version and exit
```

### Examples

```bash
# Linux
./gw-agent --config /path/to/config.yaml --dry-run
./gw-agent --config /path/to/config.yaml --once
./gw-agent --config /path/to/config.yaml

# Windows
.\gw-agent.exe --config C:\path\to\config.yaml --dry-run
.\gw-agent.exe --config C:\path\to\config.yaml --once
.\gw-agent.exe --config C:\path\to\config.yaml
```

## Installation as Service

### Linux (systemd)

```bash
# Install (creates gwagent user, systemd service)
sudo ./scripts/install-linux.sh

# Configure
sudo nano /etc/gw-agent/config.yaml

# Start service
sudo systemctl enable gw-agent
sudo systemctl start gw-agent
sudo systemctl status gw-agent

# View logs
sudo journalctl -u gw-agent -f
```

**Locations**:
- Binary: `/opt/gw-agent/gw-agent`
- Config: `/etc/gw-agent/config.yaml`
- Logs: journald

**Note**: Service runs as non-privileged `gwagent` user. Some metrics may require elevated permissions.

### Windows

```powershell
# Install (PowerShell as Administrator)
.\scripts\install-windows.ps1

# Configure
notepad C:\ProgramData\GWAgent\config.yaml

# Start service
.\scripts\start-windows.ps1
Get-Service -Name GWAgent

# Uninstall
.\scripts\install-windows.ps1 -Uninstall
```

**Locations**:
- Binary: `C:\Program Files\GWAgent\gw-agent.exe`
- Config: `C:\ProgramData\GWAgent\config.yaml`
- Logs: Windows Event Log

## Payload Format

### Minimum Payload

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-unique-id",
  "client_id": "client_name",
  "site_id": "site_name",
  "stats": {
    "system_status": "online"
  },
  "additional": {
    "metadata": {
      "platform": "ubuntu"
    }
  }
}
```

### Full Payload (with metrics)

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-unique-id",
  "stats": {
    "system_status": "online",
    "compute": {
      "cpu": {"usage_percent": 25.5},
      "memory": {
        "total_bytes": 8589934592,
        "used_bytes": 4294967296,
        "usage_percent": 50.0
      },
      "disk": {
        "total_bytes": 107374182400,
        "used_bytes": 53687091200,
        "usage_percent": 50.0
      }
    }
  },
  "additional": {
    "metadata": {
      "platform": "ubuntu",
      "agent_version": "1.0.0",
      "build": "abc123def 2024-01-01T12:00:00Z"
    }
  },
  "agent_timestamp_utc": "2024-01-01T12:00:00Z"
}
```

**Important**:
- Missing metrics are omitted entirely (not null/0/"unknown")
- Backend should use its own timestamp for availability tracking
- Metrics cached every `compute_seconds` (default: 120s)
- Platform field always present

## Retry & Token Rotation

### Retry Policy

- **5xx/network errors**: Retry 3x with backoff (5s, 15s, 30s)
- **4xx (except 401/403)**: No retry, wait for next cycle
- **401/403**: Try `token_grace` if configured, otherwise no retry
- **Timeout**: 10s per request

**Example timeline** (500 errors):
```
0s:  Initial attempt → 500
5s:  Retry 1 → 500
20s: Retry 2 → 500 (5s + 15s)
50s: Retry 3 → 200 ✓ (5s + 15s + 30s)
```

### Token Rotation (Zero-Downtime)

1. Add new token as `token_grace` in config
2. Agent tries `token_current` first, falls back to `token_grace` on 401/403
3. Verify new token works in backend
4. Promote new token to `token_current`
5. Remove old token

**Security**: Tokens never logged. Config file should be 0600 on Linux.

### Offline Detection

Backend determines availability:
- **Recommended**: Gateway offline if no heartbeat for >180s (3 missed intervals)
- Agent always sends `system_status: "online"` when heartbeat succeeds
- `agent_timestamp_utc` is for diagnostics only

## Monitoring & Metrics

### Structured Logging

JSON logs include:
- `timestamp` - ISO 8601 UTC
- `level` - DEBUG, INFO, WARN, ERROR
- `msg` - Human-readable message
- `gateway_uuid` - Partially redacted (first 4 + last 4 chars)
- `consecutive_failures`, `last_success_at` - Contextual fields

**Security**: Tokens and authorization headers never logged.

### Collected Metrics

| Metric | Description | Unit |
|--------|-------------|------|
| `cpu.usage_percent` | System-wide CPU | % (0-100) |
| `memory.total_bytes` | Total RAM | Bytes |
| `memory.used_bytes` | Used RAM | Bytes |
| `memory.usage_percent` | RAM usage | % (0-100) |
| `disk.total_bytes` | Total disk | Bytes |
| `disk.used_bytes` | Used disk | Bytes |
| `disk.usage_percent` | Disk usage | % (0-100) |

**Behavior**:
- Refreshed every `compute_seconds` (default: 120s)
- Cached and included in every heartbeat
- Omitted if collection fails (permissions, errors)
- CPU measurement takes ~1s

### View Logs

**Linux**:
```bash
sudo journalctl -u gw-agent -f        # Follow
sudo journalctl -u gw-agent -n 100    # Last 100 lines
sudo journalctl -u gw-agent -p err    # Errors only
```

**Windows**:
```powershell
Get-EventLog -LogName Application -Source GWAgent -Newest 50
Get-Content C:\ProgramData\GWAgent\logs\gw-agent.log -Tail 50 -Wait
```

## Troubleshooting

### Agent fails to start

1. Verify config is valid YAML with all required fields
2. Check logs for validation errors
3. Test manually: `./gw-agent --config config.yaml --dry-run`

### No metrics in payload

- Metrics require permissions (service may need privileges)
- Wait for one refresh cycle (default: 120s)
- Check logs for collection errors
- Metrics gracefully omitted on permission errors

### Heartbeats failing

1. Check network connectivity to `api_url`
2. Verify token is valid
3. Check TLS validation (`insecure_skip_verify: true` for testing only)
4. Review logs for HTTP error codes
5. Verify backend returns 2xx

### Service won't start

**Linux**:
```bash
sudo systemctl status gw-agent
sudo journalctl -u gw-agent -n 50
ls -l /opt/gw-agent/gw-agent
sudo -u gwagent /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --once
```

**Windows**:
```powershell
Get-Service -Name GWAgent
Get-EventLog -LogName Application -Source GWAgent -Newest 10
& "C:\Program Files\GWAgent\gw-agent.exe" --config "C:\ProgramData\GWAgent\config.yaml" --once
```

## Development

### Project Structure

```
binary-gws/
├── cmd/agent/              # Main entrypoint
├── internal/
│   ├── config/            # Configuration validation
│   ├── platform/          # Platform detection
│   ├── collector/         # System metrics collection
│   ├── scheduler/         # Heartbeat scheduling
│   ├── transport/         # HTTP client with retry
│   └── logging/           # Structured logging
├── scripts/               # Installation scripts
├── dist/                  # Build output
├── Makefile              # Build system
└── go.mod                # Go modules
```

### Testing

```bash
# Unit tests with race detector
make test

# Dry-run (verify payload without sending)
./gw-agent --config test-config.yaml --dry-run

# Single heartbeat test
./gw-agent --config test-config.yaml --once
```

Tests cover:
- Configuration validation
- Payload building with metric omission
- HTTP retry logic with mock server
- Token fallback behavior
- Backoff timing

### Adding Features

When extending:
1. **Maintain backward compatibility** - Never break existing payload fields
2. **Omit missing data** - Don't send null/0/"unknown" for unavailable metrics
3. **Add tests** - Include unit tests
4. **Update docs** - Document new config options

### Security Considerations

1. **Config secrets** - Use 0600 permissions (Linux) or restricted ACLs (Windows)
2. **Token rotation** - Use dual-token for zero-downtime updates
3. **TLS verification** - Keep `insecure_skip_verify: false` in production
4. **Service user** - Linux runs as non-privileged `gwagent` user
5. **No self-update** - Manual updates only (prevents supply-chain attacks)

### Version Information

```bash
./gw-agent --print-version
```

Version embedded at build time via ldflags and included in payload under `additional.metadata.build`.

## License

Proprietary - All rights reserved

## Support

For issues or questions, contact your system administrator or backend API provider.
