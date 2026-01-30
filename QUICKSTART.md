# Gateway Agent - Quick Start Guide

## Build the Binaries

```bash
# Build for all platforms
make build-all

# Binaries will be in:
# - dist/linux_amd64/gw-agent
# - dist/linux_arm64/gw-agent
# - dist/windows_amd64/gw-agent.exe
```

## Ubuntu Linux Installation

```bash
# 1. Build binaries
make build-all

# 2. Run installer (as root)
sudo ./scripts/install-linux.sh

# 3. Edit configuration
sudo nano /etc/gw-agent/config.yaml

# 4. Test manually
sudo -u gwagent /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --once

# 5. Enable and start service
sudo systemctl enable gw-agent
sudo systemctl start gw-agent

# 6. Check status
sudo systemctl status gw-agent
sudo journalctl -u gw-agent -f
```

## Raspberry Pi Installation

Same as Ubuntu - the installer auto-detects architecture:

```bash
make build-all
sudo ./scripts/install-linux.sh
# ... follow Ubuntu steps above
```

## Windows Installation

```powershell
# 1. Build binaries (on build machine)
make build-all

# 2. Copy binary to Windows machine
# Copy dist/windows_amd64/gw-agent.exe and scripts/*.ps1

# 3. Run PowerShell as Administrator and navigate to project folder

# 4. Install service
.\scripts\install-windows.ps1

# 5. Edit configuration
notepad C:\ProgramData\GWAgent\config.yaml

# 6. Test manually
& "C:\Program Files\GWAgent\gw-agent.exe" --config "C:\ProgramData\GWAgent\config.yaml" --once

# 7. Start service
.\scripts\start-windows.ps1
# Or: Start-Service -Name GWAgent

# 8. Check status
Get-Service -Name GWAgent
```

## Configuration Template

```yaml
uuid: "your-gateway-unique-id"
client_id: "your-client-name"
site_id: "your-site-name"
api_url: "https://api.example.com/v1/heartbeat"

auth:
  token_current: "your-api-token-here"
  # token_grace: "optional-for-rotation"

intervals:
  heartbeat_seconds: 60
  compute_seconds: 120

tls:
  insecure_skip_verify: false
```

## Testing Before Deployment

```bash
# Dry run (print payload without sending)
./gw-agent --config config.yaml --dry-run

# Send one heartbeat and exit
./gw-agent --config config.yaml --once

# Check version
./gw-agent --print-version

# Run with debug logging
./gw-agent --config config.yaml --log-level debug
```

## Common Commands

### Linux
```bash
# Start service
sudo systemctl start gw-agent

# Stop service
sudo systemctl stop gw-agent

# Restart service
sudo systemctl restart gw-agent

# Check status
sudo systemctl status gw-agent

# View logs
sudo journalctl -u gw-agent -f

# View recent errors
sudo journalctl -u gw-agent -p err -n 50
```

### Windows
```powershell
# Start service
Start-Service -Name GWAgent

# Stop service
Stop-Service -Name GWAgent

# Restart service
Restart-Service -Name GWAgent

# Check status
Get-Service -Name GWAgent

# View Event Log
Get-EventLog -LogName Application -Source GWAgent -Newest 50

# Uninstall
.\scripts\install-windows.ps1 -Uninstall
```

## Troubleshooting

### Agent won't start
1. Check config file exists and is valid YAML
2. Verify all required fields are present
3. Check logs for specific errors
4. Test with `--dry-run` first

### No heartbeats reaching backend
1. Test network connectivity: `curl -I https://api.example.com`
2. Verify token is valid
3. Check TLS settings
4. Review logs for HTTP errors

### Missing compute metrics
- Metrics require certain permissions
- Run as root/admin if needed (or accept gracefully omitted metrics)
- Check logs for specific permission errors
- Metrics are cached every 120s by default

## Payload Example

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-123",
  "client_id": "client",
  "site_id": "site",
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
      "build": "abc123 2024-01-01T12:00:00Z"
    }
  },
  "agent_timestamp_utc": "2024-01-01T12:00:00Z"
}
```

## Security Checklist

- [ ] Config file has restricted permissions (0600 on Linux)
- [ ] Token never logged or exposed
- [ ] TLS verification enabled (`insecure_skip_verify: false`)
- [ ] Service runs with minimal privileges
- [ ] API URL uses HTTPS only

## Support

See README.md for detailed documentation.
