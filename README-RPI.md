# Raspberry Pi (ARM64) - Quick Install

Short instructions for teams that want the essentials only.

## 1) Copy binary to the RPi

```bash
scp dist/linux_arm64/gw-agent user@rpi-host:/tmp/
```

## 2) Install binary

```bash
ssh user@rpi-host
sudo mkdir -p /opt/gw-agent
sudo mv /tmp/gw-agent /opt/gw-agent/gw-agent
sudo chmod +x /opt/gw-agent/gw-agent
```

## 3) Create config

```bash
sudo mkdir -p /etc/gw-agent
sudo nano /etc/gw-agent/config.yaml
```

Paste and replace values:

```yaml
uuid: "YOUR_GATEWAY_UUID"
client_id: "YOUR_CLIENT_ID"
site_id: "YOUR_SITE_ID"
api_url: "http://pulse.qure.ai:8001/api/v1/service-stats-data/"
api_url_fallbacks:
  - "http://172.17.4.26:8001/api/v1/service-stats-data/"

auth:
  token_current: "YOUR_API_TOKEN"

intervals:
  heartbeat_seconds: 60
  compute_seconds: 120

tls:
  insecure_skip_verify: false
```

## 4) Test once

```bash
sudo /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --dry-run
```

## 5) systemd service

```bash
sudo nano /etc/systemd/system/gw-agent.service
```

```ini
[Unit]
Description=Gateway Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable gw-agent
sudo systemctl start gw-agent
```

## 6) Verify

```bash
sudo systemctl status gw-agent
sudo journalctl -u gw-agent -f
```

## Troubleshooting (fast)

- Logs: `sudo journalctl -u gw-agent -n 50`
- Config: `cat /etc/gw-agent/config.yaml`
- Restart: `sudo systemctl restart gw-agent`
