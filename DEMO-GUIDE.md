# Demonstration Guide - Gateway Agent

This guide contains all the tests you can run locally to demonstrate the agent's functionality without needing to connect to a real backend.

## Demonstration Index

1. [Automated Quick Demo](#1-automated-quick-demo)
2. [Build and Binary Verification](#2-build-and-binary-verification)
3. [Payload Test (Dry-run)](#3-payload-test-dry-run)
4. [Heartbeat with Local Server](#4-heartbeat-with-local-server)
5. [Retry Logic Test](#5-retry-logic-test)
6. [Configuration Validation](#6-configuration-validation)
7. [Docker Container Test](#7-docker-container-test)
8. [Complete Demo with Docker Compose](#8-complete-demo-with-docker-compose)
9. [Unit Tests](#9-unit-tests)

---

## 1. Automated Quick Demo

**Description**: Script that runs all basic demonstrations sequentially.

**Commands**:
```bash
# Grant execution permissions
chmod +x demo.sh

# Run complete demo
./demo.sh
```

**What it demonstrates**:
- ✅ Multi-platform build
- ✅ Binary verification
- ✅ Version information
- ✅ Payload dry-run
- ✅ End-to-end heartbeat
- ✅ Config validation
- ✅ Logging levels
- ✅ Unit tests

**Duration**: ~2-3 minutes

---

## 2. Build and Binary Verification

**Description**: Compile binaries for all supported platforms.

**Commands**:
```bash
# Clean previous builds
make clean

# Build for all platforms
make build-all

# Verify binary format
file dist/linux_amd64/gw-agent
file dist/linux_arm64/gw-agent
file dist/windows_amd64/gw-agent.exe

# Check sizes
ls -lh dist/*/gw-agent*
```

**Expected results**:
```
dist/linux_amd64/gw-agent:       ELF 64-bit LSB executable, x86-64, statically linked
dist/linux_arm64/gw-agent:       ELF 64-bit LSB executable, ARM aarch64, statically linked
dist/windows_amd64/gw-agent.exe: PE32+ executable (console) x86-64, for MS Windows
```

**What it demonstrates**:
- Static binaries (no external libraries required)
- Functional cross-compilation
- Reasonable sizes (~9MB)

---

## 3. Payload Test (Dry-run)

**Description**: View the JSON payload that would be sent to the backend WITHOUT making network calls.

**Commands**:
```bash
# Basic dry-run
./dist/gw-agent --config test-local-config.yaml --dry-run

# With DEBUG logging
./dist/gw-agent --config test-local-config.yaml --dry-run --log-level debug
```

**Expected result**:
```json
{
  "payload_version": "1.0",
  "uuid": "local-test-001",
  "client_id": "test_client",
  "site_id": "test_site",
  "stats": {
    "system_status": "online",
    "compute": {
      "cpu": {"usage_percent": 25.5},
      "memory": {
        "total_bytes": 17179869184,
        "used_bytes": 12552273920,
        "usage_percent": 73.06
      },
      "disk": {
        "total_bytes": 494384795648,
        "used_bytes": 100244705280,
        "usage_percent": 20.27
      }
    }
  },
  "additional_notes": {
    "metadata": {
      "platform": "linux",
      "agent_version": "dev",
      "build": "none unknown"
    }
  },
  "agent_timestamp_utc": "2026-01-24T14:28:23Z"
}
```

**What it demonstrates**:
- ✅ Payload structure conforms to v1.0 contract
- ✅ System metrics (CPU, memory, disk) collected locally
- ✅ Platform detection
- ✅ Version metadata

---

## 4. Heartbeat with Local Server

**Description**: Simple HTTP server that receives heartbeats and displays the payload.

### Terminal 1: Server
```bash
go run test-server.go
```

Expected output:
```
🚀 Test server starting on :8080
📡 Endpoint: http://localhost:8080/heartbeat
Waiting for heartbeats...
```

### Terminal 2: Agent
```bash
# Single send
./dist/gw-agent --config test-local-config.yaml --once

# Continuous sending (Ctrl+C to stop)
./dist/gw-agent --config test-local-config.yaml
```

### Server Logs
```
[Request #1] Method: POST, Path: /heartbeat
[Request #1] Authorization: Bearer test-token
[Request #1] Payload received:
{
  "uuid": "local-test-001",
  "client_id": "test_client",
  ...
}
[Request #1] ✅ System Status: online
[Request #1] ✅ Compute metrics present: true
[Request #1] ✅ Response sent successfully
```

**What it demonstrates**:
- ✅ Functional HTTP POST communication
- ✅ Bearer token authentication
- ✅ Well-formed JSON payload
- ✅ Metrics included in payload
- ✅ Configurable heartbeat interval (60s by default)

---

## 5. Retry Logic Test

**Description**: Server that intentionally fails the first 2 requests to test retry logic with backoff.

### Terminal 1: Server simulating failures
```bash
go run test-retry-server.go
```

Expected output:
```
🚀 Retry Test Server starting on :8080
⚠️  First 2 requests will return HTTP 500 to test retry logic
Waiting for heartbeats...
```

### Terminal 2: Agent
```bash
./dist/gw-agent --config test-local-config.yaml --once --log-level debug
```

### Expected behavior

**Request #1** - Immediate failure
```
[Request #1] 🔴 SIMULATING SERVER ERROR (HTTP 500)
```

**Request #2** - Retry after 5 seconds
```
[Request #2] 🔴 SIMULATING SERVER ERROR (HTTP 500)
```

**Request #3** - Retry after 15 seconds (cumulative: 20s)
```
[Request #3] 🟢 SUCCESS - Payload received
[Request #3] ✅ Response sent successfully (after 2 retries)
```

**Timeline**:
```
0s:   Attempt 1 → HTTP 500
5s:   Attempt 2 → HTTP 500
20s:  Attempt 3 → HTTP 200 ✓
```

**What it demonstrates**:
- ✅ Automatic retries on 5xx errors
- ✅ Exponential backoff (delays: 5s, 15s, 30s)
- ✅ Persistence until success
- ✅ Does not block indefinitely

---

## 6. Configuration Validation

**Description**: Verify that the agent detects invalid configurations.

### Test 1: Missing required field
```bash
cat > /tmp/invalid1.yaml <<EOF
uuid: "test"
# client_id missing (REQUIRED)
site_id: "test"
api_url: "http://localhost:8080/heartbeat"
auth:
  token_current: "test"
EOF

./dist/gw-agent --config /tmp/invalid1.yaml --dry-run
```

**Expected result**:
```
Failed to load configuration: config validation failed: client_id is required
```
✅ **PASS** - Detects missing field

### Test 2: Invalid URL
```bash
cat > /tmp/invalid2.yaml <<EOF
uuid: "test"
client_id: "test"
site_id: "test"
api_url: "not-a-valid-url"
auth:
  token_current: "test"
EOF

./dist/gw-agent --config /tmp/invalid2.yaml --dry-run
```

**Expected result**:
```
Failed to load configuration: config validation failed: api_url must be an HTTP(S) URL
```
✅ **PASS** - Detects malformed URL

### Test 3: Mutually exclusive configuration
```bash
cat > /tmp/invalid3.yaml <<EOF
uuid: "test"
client_id: "test"
site_id: "test"
api_url: "http://localhost:8080/heartbeat"
auth:
  token_current: "test"
tls:
  insecure_skip_verify: true
  ca_bundle_path: "/some/path.pem"  # CANNOT have both
EOF

./dist/gw-agent --config /tmp/invalid3.yaml --dry-run
```

**Expected result**:
```
Failed to load configuration: config validation failed: tls.insecure_skip_verify and tls.ca_bundle_path are mutually exclusive
```
✅ **PASS** - Detects configuration conflict

**What it demonstrates**:
- ✅ Robust configuration validation
- ✅ Clear error messages
- ✅ Fails fast before starting

---

## 7. Docker Container Test

**Description**: Run the agent inside a Linux Ubuntu container to simulate real deployment.

### Image build
```bash
# Build Linux binary
make build-all

# Build Docker image
docker build -f Dockerfile.demo -t gw-agent:demo .
```

### Basic verification
```bash
# Verify version inside container
docker run --rm gw-agent:demo /usr/local/bin/gw-agent --print-version
```

### Dry-run in container
```bash
docker run --rm \
  -v $(pwd)/test-local-config.yaml:/etc/gw-agent/config.yaml:ro \
  gw-agent:demo \
  /usr/local/bin/gw-agent --config /etc/gw-agent/config.yaml --dry-run
```

**What it demonstrates**:
- ✅ Binary works on Ubuntu 22.04
- ✅ Non-privileged user (gwagent)
- ✅ Same functionality as on host

---

## 8. Complete Demo with Docker Compose

**Description**: Launch server + agent in Docker network to simulate complete environment.

### Commands
```bash
# Launch services
docker-compose -f docker-compose.demo.yml up --build

# View logs in real-time
docker-compose -f docker-compose.demo.yml logs -f

# Stop
docker-compose -f docker-compose.demo.yml down
```

### Expected output

**test-server**:
```
test-server_1       | 🚀 Test server starting on :8080
test-server_1       | Waiting for heartbeats...
test-server_1       | [Request #1] Method: POST, Path: /heartbeat
test-server_1       | [Request #1] ✅ System Status: online
```

**gw-agent-ubuntu**:
```
gw-agent-ubuntu_1   | {"timestamp":"2026-01-24T15:00:00Z","level":"INFO","msg":"Starting Gateway Agent",...}
gw-agent-ubuntu_1   | {"timestamp":"2026-01-24T15:00:00Z","level":"INFO","msg":"Heartbeat sent successfully",...}
```

**What it demonstrates**:
- ✅ Communication between containers
- ✅ DNS resolution (test-server hostname)
- ✅ Continuous heartbeats every 10 seconds
- ✅ Reproducible isolated environment

---

## 9. Unit Tests

**Description**: Run automated test suite.

### Commands
```bash
# Basic tests
make test

# Tests with verbose output
go test -v ./...

# Tests with coverage
go test -cover ./...

# Tests with race detector
go test -race ./...
```

### Expected results
```
ok      github.com/binary-gws/agent/internal/config      0.123s
ok      github.com/binary-gws/agent/internal/scheduler   0.456s
ok      github.com/binary-gws/agent/internal/transport   0.789s
```

**What it demonstrates**:
- ✅ Config validation tests
- ✅ Payload building tests
- ✅ HTTP retry logic tests
- ✅ Token fallback tests
- ✅ No race conditions

---

## 10. Binary Verification on Real Platform

### Linux (if you have access to a Linux VM)

```bash
# On your Mac, copy binary to VM
scp dist/linux_amd64/gw-agent user@linux-vm:~/

# On the Linux VM
chmod +x gw-agent
./gw-agent --print-version
./gw-agent --config config.yaml --dry-run
```

### Windows (if you have access to Windows VM)

```powershell
# Copy gw-agent.exe to Windows
# From PowerShell:
.\gw-agent.exe --print-version
.\gw-agent.exe --config config.yaml --dry-run
```

---

## Summary of Demonstrated Capabilities

| Functionality | Demo | Time |
|---------------|------|--------|
| Multi-platform build | #2 | 1 min |
| Static binaries | #2 | 1 min |
| Payload preview (dry-run) | #3 | 30 sec |
| Local HTTP heartbeat | #4 | 2 min |
| Retry logic | #5 | 1 min |
| Config validation | #6 | 2 min |
| Docker execution | #7 | 2 min |
| Complete Docker stack | #8 | 3 min |
| Unit tests | #9 | 1 min |

**Total**: 13-15 minutes for complete demo

---

## Checklist for Boss Presentation

```
□ Run demo.sh to show complete flow
□ Show multi-platform binaries (file command)
□ Show dry-run of payload with real metrics
□ Run heartbeat with local server (2 terminals)
□ Demonstrate retry logic with failing server
□ Show invalid configuration validation
□ (Optional) Launch complete stack with Docker Compose
□ Show passing unit tests
□ Show documentation (README.md, QUICKSTART.md)
```

---

## Next Steps (Post-Demo)

1. **Test on real devices**:
   - Raspberry Pi (use linux_arm64 binary)
   - Ubuntu Laptop (use linux_amd64 binary)
   - Windows PC (use gw-agent.exe)

2. **Integration with real backend**:
   - Configure api_url with real endpoint
   - Configure production tokens
   - Verify TLS with real certificates

3. **Install as service**:
   - Linux: `sudo ./scripts/install-linux.sh`
   - Windows: `.\scripts\install-windows.ps1`

4. **Long-duration tests**:
   - Leave running for 24h
   - Monitor logs
   - Verify resource consumption
