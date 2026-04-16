
# Clean Installation Procedure for GW Agent

The binary is already located at `/tmp/gw-agent` on the server.  
Run the following commands on the server as `root` or with `sudo`.

---

## 0. Generate / Validate the Gateway UUID (required)

The `uuid` in the agent configuration **must** be set to the gateway UUID.

### 0.1 Get the UUID from the kernel interface
```bash
cat /proc/sys/kernel/random/uuid
```

Example output:
```text
6000a75f-e622-4d88-b020-5b4244df4ff1
```

### 0.2 If the UUID is not available, create one
On a standard Linux system, `/proc/sys/kernel/random/uuid` should exist. If it does not (or returns nothing), generate a UUID manually and use it in the config:

```bash
uuidgen
```

If `uuidgen` is not installed, install it and try again:
- Debian/Ubuntu:
```bash
sudo apt-get update
sudo apt-get install -y uuid-runtime
uuidgen
```
- RHEL/CentOS/Rocky/Alma:
```bash
sudo yum install -y util-linux
uuidgen
```

Copy the resulting UUID and use it as the value of `uuid:` in `/etc/gw-agent/config.yaml` (Step 3).

---

## 1. Clean Previous Installation (if it exists)
```bash
sudo systemctl stop gw-agent 2>/dev/null || true
sudo systemctl disable gw-agent 2>/dev/null || true
```

## 2. Create User, Directories, and Install Binary
```bash
sudo useradd --system --no-create-home --shell /bin/false gwagent 2>/dev/null || true
sudo mkdir -p /opt/gw-agent /etc/gw-agent /var/log/gw-agent
sudo cp /tmp/gw-agent /opt/gw-agent/gw-agent
sudo chmod 755 /opt/gw-agent/gw-agent
sudo chown root:root /opt/gw-agent/gw-agent
sudo chown gwagent:gwagent /var/log/gw-agent
```

## 3. Create `config.yaml`
Replace the `uuid:` value with the UUID obtained/generated in Step 0.

```bash
sudo tee /etc/gw-agent/config.yaml > /dev/null <<'EOF'
uuid: "6000a75f-e622-4d88-b020-5b4244df4ff1"
client_id: "azelsalvador5"
site_id: "azelsalvador5"

api_url: "https://pulse-staging-api.qure.ai/api/v1/service-stats-data/"

auth:
  token_current: "4d06fd1962427e1c0ec890005b7de8ceed3fee04fd80b8cb52937a5d2e162801"

intervals:
  heartbeat_seconds: 15
  compute_seconds: 120

dcmio:
  container_name: "postgres_dcmio"
  postgres_user: "postgres"
  postgres_pass: ""
  postgres_db: "dcmio"
  interval_seconds: 60
  window_hours: 12

monitored_processes:
  - rundicomserver
  - dockerd
  - startworker
  - postgres

tls:
  insecure_skip_verify: false
EOF

sudo chmod 600 /etc/gw-agent/config.yaml
sudo chown gwagent:gwagent /etc/gw-agent/config.yaml
```

## 4. Install the Systemd Service
```bash
sudo tee /etc/systemd/system/gw-agent.service > /dev/null <<'EOF'
[Unit]
Description=Gateway Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=gwagent
Group=gwagent
ExecStart=/opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gw-agent

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/gw-agent
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
```

## 5. Add `gwagent` to the Docker Group
(Required to execute `docker exec` on the `postgres_dcmio` container)
```bash
sudo usermod -aG docker gwagent
```

## 6. Test Before Enabling the Service
```bash
sudo -u gwagent /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --once
```

If the output shows:
```text
Heartbeat sent successfully
```
The installation is correct.

## 7. Enable and Start the Service
```bash
sudo systemctl enable gw-agent
sudo systemctl start gw-agent
sudo systemctl status gw-agent
```

## 8. Verify Live Logs
```bash
sudo journalctl -u gw-agent -f
```
