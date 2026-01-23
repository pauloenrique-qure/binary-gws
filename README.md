# Gateway Monitoring Agent

A cross-platform daemon that monitors gateway status and sends periodic heartbeats to a backend API. Built in Go for maximum portability and minimal resource usage.

## Overview

The Gateway Monitoring Agent is a push-only daemon (no inbound ports) that:
- Sends periodic JSON heartbeats to a backend API over HTTPS
- Collects system metrics (CPU, memory, disk) when available
- Supports token rotation for zero-downtime credential updates
- Runs as a system service on Linux and Windows
- Produces static binaries with no external runtime dependencies

## Features

- **Cross-platform**: Supports Linux (amd64/arm64) and Windows (amd64)
- **Push-only**: No listening ports, all communication is outbound HTTPS
- **Secure**: Token-based authentication with optional dual-token rotation
- **Resilient**: Automatic retry with exponential backoff on transient failures
- **Lightweight**: Single static binary, minimal resource usage
- **Observable**: Structured logging with configurable levels

## Supported Platforms

| Platform | Architecture | Target | Notes |
|----------|-------------|---------|-------|
| Ubuntu | amd64 | `linux_amd64` | Tested on Ubuntu 20.04+ |
| Raspberry Pi | arm64 | `linux_arm64` | Tested on Raspberry Pi 4 |
| Windows 11 | amd64 | `windows_amd64` | Requires Administrator |

## Build

### Prerequisites

- Go 1.21 or later
- Make (or run Go commands directly)

### Build for Current Platform

```bash
make build
```

Binary will be in `./dist/gw-agent` (or `gw-agent.exe` on Windows)

### Build for All Platforms

```bash
make build-all
```

Binaries will be in:
- `./dist/linux_amd64/gw-agent`
- `./dist/linux_arm64/gw-agent`
- `./dist/windows_amd64/gw-agent.exe`

### Other Make Targets

```bash
make test      # Run tests
make lint      # Run linters (requires golangci-lint or uses go vet)
make clean     # Remove build artifacts
make deps      # Download dependencies
```

## Configuration

Configuration is in YAML format. By default, the agent looks for:
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
  token_grace: "optional-old-token-for-rotation"  # Optional

# Platform detection (optional)
platform:
  platform_override: "ubuntu"  # Optional: raspberry_pi, ubuntu, windows, vm, linux

# Timing intervals (optional, defaults shown)
intervals:
  heartbeat_seconds: 60   # How often to send heartbeats
  compute_seconds: 120    # How often to refresh compute metrics

# TLS configuration (optional)
tls:
  ca_bundle_path: "/path/to/ca-bundle.pem"  # Optional custom CA bundle
  insecure_skip_verify: false               # Must be false by default
```

### Required Fields

The following fields are **required** and the agent will fail to start if missing:
- `uuid` - Unique identifier for this gateway
- `client_id` - Client name
- `site_id` - Site name
- `api_url` - Full HTTPS URL to the backend API endpoint
- `auth.token_current` - Current API authentication token

### Platform Override

If not specified, the agent auto-detects the platform:
- `raspberry_pi` - Linux ARM64 with Raspberry Pi board detected
- `ubuntu` - Linux with Ubuntu in `/etc/os-release`
- `windows` - Windows OS
- `vm` - Virtualization detected (VirtualBox, VMware, KVM, etc.)
- `linux` - Generic Linux (fallback)

## Payload Format

### Base Contract (Backward Compatible)

The agent sends JSON payloads matching this minimum structure:

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-unique-id",
  "client_id": "client_name",
  "site_id": "site_name",
  "stats": {
    "system_status": "online"
  },
  "additional_notes": {
    "metadata": {
      "platform": "raspberry_pi"
    }
  }
}
```

### Extended Payload (With Metrics)

When compute metrics are available (collected every 120s by default):

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-unique-id",
  "client_id": "client_name",
  "site_id": "site_name",
  "stats": {
    "system_status": "online",
    "compute": {
      "cpu": {
        "usage_percent": 25.5
      },
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
  "additional_notes": {
    "metadata": {
      "platform": "ubuntu",
      "agent_version": "1.0.0",
      "build": "abc123def 2024-01-01T12:00:00Z"
    }
  },
  "agent_timestamp_utc": "2024-01-01T12:00:00Z"
}
```

### Important Notes on Payload

- **Missing metrics are omitted** - If CPU/memory/disk metrics cannot be collected (permissions, errors), those fields are omitted entirely (not null, not 0, not "unknown")
- **Backend uses backend time** - The `agent_timestamp_utc` field is for diagnostics only; the backend should use its own timestamp for availability tracking
- **Compute metrics are cached** - Metrics are refreshed based on `compute_seconds` interval but included in every heartbeat when available
- **Platform is always present** - The `metadata.platform` field is always included

## Running the Agent

### Command-Line Flags

```bash
gw-agent [flags]

Flags:
  --config string       Path to configuration file (default "/etc/gw-agent/config.yaml")
  --once                Send one heartbeat and exit
  --log-level string    Log level: debug, info, warn, error (default "info")
  --dry-run             Build payload and print to stdout without sending
  --print-version       Print version information and exit
```

### Manual Execution

#### Linux

```bash
# Test with dry run
./gw-agent --config /path/to/config.yaml --dry-run

# Send one heartbeat
./gw-agent --config /path/to/config.yaml --once

# Run continuously
./gw-agent --config /path/to/config.yaml
```

#### Windows

```powershell
# Test with dry run
.\gw-agent.exe --config C:\path\to\config.yaml --dry-run

# Send one heartbeat
.\gw-agent.exe --config C:\path\to\config.yaml --once

# Run continuously
.\gw-agent.exe --config C:\path\to\config.yaml
```

## Installation as Service

### Linux (Ubuntu/Raspberry Pi)

The installer creates a systemd service running as a dedicated `gwagent` user.

```bash
# Run the installation script (requires root)
sudo ./scripts/install-linux.sh

# Edit the configuration
sudo nano /etc/gw-agent/config.yaml

# Enable and start the service
sudo systemctl enable gw-agent
sudo systemctl start gw-agent

# Check status
sudo systemctl status gw-agent

# View logs
sudo journalctl -u gw-agent -f
```

**Installed locations:**
- Binary: `/opt/gw-agent/gw-agent`
- Config: `/etc/gw-agent/config.yaml`
- Logs: journald (use `journalctl -u gw-agent`)

**Note on permissions:** The service runs as the `gwagent` user with minimal privileges. Some system metrics may require elevated permissions. If all metrics fail to collect, you may need to adjust the service user or permissions.

### Windows

The installer creates a Windows service using `sc.exe`.

```powershell
# Run PowerShell as Administrator
cd binary-gws

# Install the service
.\scripts\install-windows.ps1

# Edit the configuration
notepad C:\ProgramData\GWAgent\config.yaml

# Start the service
.\scripts\start-windows.ps1
# Or: Start-Service -Name GWAgent

# Check status
Get-Service -Name GWAgent

# Stop the service
.\scripts\stop-windows.ps1
# Or: Stop-Service -Name GWAgent

# Uninstall
.\scripts\install-windows.ps1 -Uninstall
```

**Installed locations:**
- Binary: `C:\Program Files\GWAgent\gw-agent.exe`
- Config: `C:\ProgramData\GWAgent\config.yaml`
- Logs: Windows Event Log or `C:\ProgramData\GWAgent\logs\`

## Retry and Backoff Policy

The agent implements a deterministic retry policy for resilience:

### Retry Behavior

- **Network errors or HTTP 5xx** - Retry up to 3 times with backoff delays: 5s, 15s, 30s
- **HTTP 4xx (except 401/403)** - No retry, log error and wait for next cycle
- **HTTP 401/403** - If `token_grace` is configured, fallback to grace token once; otherwise no retry

### Timeout Policy

- **Request timeout**: 10 seconds per HTTP request
- **Cycle behavior**: Retries happen within a single heartbeat cycle; the cycle never blocks indefinitely
- After exhausting retries, the agent waits until the next heartbeat interval

### Example Timeline

For a 500 Internal Server Error:
```
0s:  Initial request fails (HTTP 500)
5s:  First retry fails (HTTP 500)
20s: Second retry fails (HTTP 500, cumulative: 5s + 15s delay)
50s: Third retry succeeds (cumulative: 5s + 15s + 30s delay)
```

## Token Rotation

The agent supports dual-token rotation for zero-downtime credential updates:

1. Configure both `token_current` and `token_grace` in config
2. Agent tries `token_current` first
3. If it receives 401 or 403, agent tries `token_grace` once (single fallback per cycle)
4. Once the new token is confirmed working, remove `token_grace` from config

This allows you to:
- Add new token as `token_grace`
- Verify it works in backend
- Promote new token to `token_current`
- Remove old token

**Security note:** Tokens are never logged. The configuration file should have restricted permissions (0600 on Linux).

## Offline Detection

The agent itself does not track "offline" status. The **backend** is responsible for determining availability:

- **Recommended policy**: If no heartbeat received for **>180 seconds** (3 missed 60s intervals), consider gateway offline
- The agent sends `system_status: "online"` in every successful heartbeat
- The agent includes `agent_timestamp_utc` for diagnostics, but the backend should use its own receive timestamp

## Metrics and Observability

### Structured Logging

All logs are JSON-formatted with:
- `timestamp` - ISO 8601 UTC timestamp
- `level` - DEBUG, INFO, WARN, ERROR
- `msg` - Human-readable message
- `gateway_uuid` - Partially redacted UUID (first 4 and last 4 chars)
- Additional contextual fields (e.g., `consecutive_failures`, `last_success_at`)

**Security:** Tokens and authorization headers are never logged.

### Key Metrics

The agent logs:
- `consecutive_failures` - Count of failed heartbeat attempts
- `last_success_at` - Timestamp of last successful heartbeat
- Platform detection results on startup
- Retry attempts and backoff delays

### Monitoring the Agent

On Linux:
```bash
# View recent logs
sudo journalctl -u gw-agent -n 100

# Follow logs in real-time
sudo journalctl -u gw-agent -f

# Check for errors
sudo journalctl -u gw-agent -p err
```

On Windows:
```powershell
# View Windows Event Log
Get-EventLog -LogName Application -Source GWAgent -Newest 50

# Or check log file if configured for file logging
Get-Content C:\ProgramData\GWAgent\logs\gw-agent.log -Tail 50 -Wait
```

## Compute Metrics

The agent collects the following system metrics when available:

| Metric | Description | Unit |
|--------|-------------|------|
| `cpu.usage_percent` | System-wide CPU usage | Percentage (0-100) |
| `memory.total_bytes` | Total physical memory | Bytes |
| `memory.used_bytes` | Used physical memory | Bytes |
| `memory.usage_percent` | Memory usage percentage | Percentage (0-100) |
| `disk.total_bytes` | Total disk space (root/primary volume) | Bytes |
| `disk.used_bytes` | Used disk space | Bytes |
| `disk.usage_percent` | Disk usage percentage | Percentage (0-100) |

**Behavior:**
- Metrics are refreshed every `compute_seconds` interval (default: 120s)
- Cached metrics are included in every heartbeat when available
- If metric collection fails (permissions, errors), that metric group is omitted entirely
- CPU measurement takes ~1 second to calculate average usage

## Testing

### Unit Tests

```bash
make test
```

Tests cover:
- Configuration validation
- Payload building (ensures missing metrics are omitted)
- HTTP retry logic with mock server
- Token fallback behavior
- Backoff timing (using injectable sleeper interface)

### Integration Testing

Use the `--dry-run` flag to verify payload format without sending:

```bash
./gw-agent --config ./test-config.yaml --dry-run
```

Use the `--once` flag to send a single heartbeat and exit:

```bash
./gw-agent --config ./config.yaml --once
```

## Troubleshooting

### Agent fails to start

1. Check configuration file exists and is valid YAML
2. Ensure all required fields are present (`uuid`, `client_id`, `site_id`, `api_url`, `auth.token_current`)
3. Check logs for specific validation errors

### No metrics in payload

- Metrics require certain permissions; the service may need elevated privileges
- Check logs for metric collection errors
- Metrics are cached every 120s by default; wait for one refresh cycle
- If permission errors occur, metrics are gracefully omitted

### Heartbeats failing

1. Check network connectivity to `api_url`
2. Verify token is valid
3. Check TLS certificate validation (use `insecure_skip_verify: true` only for testing)
4. Review logs for specific HTTP error codes
5. Check backend API is returning 2xx status codes

### Service won't start on Linux

```bash
# Check service status
sudo systemctl status gw-agent

# Check for errors
sudo journalctl -u gw-agent -n 50

# Verify binary permissions
ls -l /opt/gw-agent/gw-agent

# Test running manually as service user
sudo -u gwagent /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --once
```

### Service won't start on Windows

```powershell
# Check service status
Get-Service -Name GWAgent

# Check Event Viewer
Get-EventLog -LogName Application -Source GWAgent -Newest 10

# Test running manually
& "C:\Program Files\GWAgent\gw-agent.exe" --config "C:\ProgramData\GWAgent\config.yaml" --once
```

## Security Considerations

1. **Configuration files contain secrets** - Ensure config files have restrictive permissions (0600 on Linux, restricted ACLs on Windows)
2. **Token rotation** - Use dual-token rotation to update credentials without downtime
3. **TLS verification** - Keep `insecure_skip_verify: false` in production
4. **Service user** - Linux service runs as dedicated `gwagent` user with minimal privileges
5. **No self-update** - The agent does NOT self-update; updates must be performed manually to prevent supply-chain attacks

## Development

### Project Structure

```
binary-gws/
├── cmd/agent/              # Main entrypoint
│   └── main.go
├── internal/
│   ├── config/            # Configuration loading and validation
│   ├── platform/          # Platform detection
│   ├── collector/         # System metrics collection
│   ├── scheduler/         # Heartbeat scheduling and payload building
│   ├── transport/         # HTTP client with retry logic
│   └── logging/           # Structured logging
├── scripts/               # Installation and service scripts
│   ├── gw-agent.service   # systemd unit file
│   ├── install-linux.sh   # Linux installer
│   ├── install-windows.ps1 # Windows installer
│   ├── start-windows.ps1  # Windows service start script
│   └── stop-windows.ps1   # Windows service stop script
├── dist/                  # Build output (created by make)
├── Makefile              # Build system
├── go.mod                # Go module definition
└── README.md             # This file
```

### Adding Features

When extending the agent:
1. **Maintain backward compatibility** - Never break existing payload fields
2. **Omit missing data** - Don't send null/0/"unknown" for unavailable metrics
3. **Add tests** - Include unit tests for new functionality
4. **Update README** - Document new configuration options and behavior

## Version Information

To check the agent version:

```bash
./gw-agent --print-version
```

Version information is embedded at build time using ldflags and included in the payload under `additional_notes.metadata.build`.

## License

Proprietary - All rights reserved

## Support

For issues or questions, contact your system administrator or backend API provider.
