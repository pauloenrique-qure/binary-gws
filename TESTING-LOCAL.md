# Local Testing - Quick Guide

This guide shows you how to run local tests of the Gateway Agent without needing to connect to a real backend.

## Option 1: Interactive Script (Recommended)

Interactive menu with all available tests:

```bash
./quick-tests.sh
```

Available options:
1. Build binaries
2. Verify binaries
3. Print version
4. Dry-run (view payload)
5. Test with local server
6. Test retry logic
7. Validation test
8. Unit tests
9. Complete Docker demo
10. Clean builds

## Option 2: Complete Automated Demo

Run all tests sequentially:

```bash
./demo.sh
```

Duration: 2-3 minutes

## Option 3: Individual Manual Tests

### 1. Build and Verification

```bash
# Build
make build-all

# Verify binaries
file dist/linux_amd64/gw-agent
file dist/linux_arm64/gw-agent
file dist/windows_amd64/gw-agent.exe

# Sizes
ls -lh dist/*/gw-agent*
```

### 2. Version Info

```bash
./dist/gw-agent --print-version
```

### 3. Dry-run (No Network)

```bash
# Basic
./dist/gw-agent --config test-local-config.yaml --dry-run

# With debug
./dist/gw-agent --config test-local-config.yaml --dry-run --log-level debug
```

### 4. Local Server (Requires 2 Terminals)

**Terminal 1 - Server**:
```bash
go run test-server.go
```

**Terminal 2 - Agent**:
```bash
# One heartbeat
./dist/gw-agent --config test-local-config.yaml --once

# Continuous (Ctrl+C to stop)
./dist/gw-agent --config test-local-config.yaml
```

### 5. Retry Logic Test

**Terminal 1 - Server with simulated failures**:
```bash
go run test-retry-server.go
```

**Terminal 2 - Agent**:
```bash
./dist/gw-agent --config test-local-config.yaml --once --log-level debug
```

Observe the retries every 5s, 15s until it works.

### 6. Unit Tests

```bash
# All tests
make test

# With verbose
go test -v ./...

# With coverage
go test -cover ./...
```

## Option 4: Docker

### Simple Test

```bash
# Build image
docker build -f Dockerfile.demo -t gw-agent:demo .

# Test version
docker run --rm gw-agent:demo /usr/local/bin/gw-agent --print-version

# Dry-run
docker run --rm \
  -v $(pwd)/test-local-config.yaml:/etc/gw-agent/config.yaml:ro \
  gw-agent:demo \
  /usr/local/bin/gw-agent --config /etc/gw-agent/config.yaml --dry-run
```

### Complete Stack (Server + Agent)

```bash
# Launch
docker-compose -f docker-compose.demo.yml up --build

# View logs
docker-compose -f docker-compose.demo.yml logs -f

# Stop
docker-compose -f docker-compose.demo.yml down
```

## What to Look For in Each Test

### Build (Expected)
```
dist/linux_amd64/gw-agent:       ELF 64-bit, statically linked
dist/linux_arm64/gw-agent:       ELF 64-bit, ARM aarch64, statically linked
dist/windows_amd64/gw-agent.exe: PE32+ executable, for MS Windows

Sizes: ~9MB each
```

### Dry-run (Expected)
```json
{
  "payload_version": "1.0",
  "uuid": "local-test-001",
  "stats": {
    "system_status": "online",
    "compute": {
      "cpu": {"usage_percent": 25.5},
      "memory": {...},
      "disk": {...}
    }
  },
  ...
}
```

### Local Server (Expected)
```
[Request #1] Method: POST, Path: /heartbeat
[Request #1] Authorization: Bearer test-token
[Request #1] ✅ System Status: online
[Request #1] ✅ Compute metrics present: true
[Request #1] ✅ Response sent successfully
```

### Retry Test (Expected)
```
[Request #1] 🔴 HTTP 500 (failure)
[Request #2] 🔴 HTTP 500 (failure, after 5s)
[Request #3] 🟢 SUCCESS (after 15s more)
```

### Unit Tests (Expected)
```
ok      .../internal/config      0.123s
ok      .../internal/scheduler   0.456s
ok      .../internal/transport   0.789s
```

## Troubleshooting

### Error: "make: command not found"
Run go commands directly:
```bash
go build -o dist/gw-agent ./cmd/agent
```

### Error: "port 8080 already in use"
Kill the process using the port:
```bash
lsof -ti:8080 | xargs kill
```

### Error: Binary doesn't run in Docker
Make sure to use the correct binary for the platform:
```bash
# For Docker (Linux)
make build-all  # Use dist/linux_amd64/gw-agent
```

### Dry-run doesn't show metrics
This is normal if you don't have permissions. Metrics are gracefully omitted.

## Next Steps

After validating locally:

1. **Validate on real devices**:
   - Copy appropriate binary to each device
   - Run `--print-version` and `--dry-run`
   - Test `--once` with real endpoint or test-server

2. **Install as service**:
   - Linux: `sudo ./scripts/install-linux.sh`
   - Windows: `.\scripts\install-windows.ps1` (as Administrator)

3. **Integrate with backend**:
   - Configure `api_url` with real endpoint
   - Configure production tokens
   - Verify TLS

## Additional Documentation

- **DEMO-GUIDE.md**: Complete guide of all demos (10 detailed demos)
- **TEST-RESULTS-CONSOLIDATED.md**: Consolidated test results and evidence
- **README.md**: Complete project documentation
- **QUICKSTART.md**: Quick installation guide

## Validation Checklist

```
□ Successful binary build
□ Verification with file command
□ Print version works
□ Dry-run shows valid payload
□ Successful heartbeat to local server
□ Server receives correct payload with metrics
□ Unit tests pass
□ (Optional) Docker works
```

If all items are ✓, the agent is ready for testing on real devices.
