## Evidence: Local Validation

This document records the local validation evidence for the agent binaries and
localhost heartbeat flow. It is intended for presentation and audit trail.
It also includes a short checklist for on-device validation in real targets.

### 1) Cross-compiled binaries (no network)

Command:
```
GOMODCACHE=/Users/pauloenrique/go/pkg/mod GOPROXY=off GOSUMDB=off \
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/linux_amd64/gw-agent ./cmd/agent
GOMODCACHE=/Users/pauloenrique/go/pkg/mod GOPROXY=off GOSUMDB=off \
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/linux_arm64/gw-agent ./cmd/agent
GOMODCACHE=/Users/pauloenrique/go/pkg/mod GOPROXY=off GOSUMDB=off \
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/windows_amd64/gw-agent.exe ./cmd/agent
```

Binary format verification:
```
dist/linux_amd64/gw-agent:       ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, BuildID[sha1]=ae3bb3fa06db982a01c6492dd1ad441fce6ec456, with debug_info, not stripped
dist/linux_arm64/gw-agent:       ELF 64-bit LSB executable, ARM aarch64, version 1 (SYSV), statically linked, BuildID[sha1]=b3af37d7cb304d7f2a441a80893dc085fefb25f3, with debug_info, not stripped
dist/windows_amd64/gw-agent.exe: PE32+ executable (console) x86-64, for MS Windows
```

### 2) Local execution proof

Print version (local build):
```
./agent --print-version
Gateway Agent
Version: dev
Commit: none
Build Date: unknown
```

### 3) Localhost end-to-end heartbeat

Local test server:
```
GOMODCACHE=/Users/pauloenrique/go/pkg/mod GOPROXY=off GOSUMDB=off go run test-server.go
```

Agent run (single heartbeat):
```
./agent --config test-local-config.yaml --once
```

Server log evidence (re-run):
```
[Request #1] Method: POST, Path: /heartbeat
[Request #1] Authorization: Bearer test-token
[Request #1] Payload received:
{
  "additional_notes": {
    "metadata": {
      "agent_version": "dev",
      "build": "none unknown",
      "platform": "linux"
    }
  },
  "agent_timestamp_utc": "2026-01-24T14:28:23Z",
  "client_id": "test_client",
  "payload_version": "1.0",
  "site_id": "test_site",
  "stats": {
    "compute": {
      "cpu": {
        "usage_percent": 19.558676028052776
      },
      "disk": {
        "total_bytes": 494384795648,
        "usage_percent": 20.27665619218877,
        "used_bytes": 100244705280
      },
      "memory": {
        "total_bytes": 17179869184,
        "usage_percent": 73.06385040283203,
        "used_bytes": 12552273920
      }
    },
    "system_status": "online"
  },
  "uuid": "local-test-001"
}
[Request #1] ✅ System Status: online
[Request #1] ✅ Compute metrics present: true
[Request #1] ✅ Response sent successfully
```

### 4) On-device validation checklist (RPI, laptops, VMs)

Run these steps on each target host after copying the appropriate binary.

1. Verify binary format:
```
file gw-agent
```

2. Print version:
```
./gw-agent --print-version
```

3. Dry-run (no network call):
```
./gw-agent --config test-config.yaml --dry-run
```

4. One-shot heartbeat (requires backend or localhost test server):
```
./gw-agent --config test-config.yaml --once
```

### 5) Coverage matrix (local vs pending)

Covered locally:
- Binary builds and formats (Linux amd64/arm64, Windows amd64)
- Local execution (`--print-version`)
- Heartbeat flow to localhost with auth header and payload fields
- Metrics present in payload (cpu/memory/disk)

Pending on real targets / environments:
- Platform detection on RPI/Windows/VM/Ubuntu
- TLS verification and custom CA bundle handling
- Token rotation (`token_current` -> `token_grace` on 401/403)
- Retry/backoff behavior on 5xx or timeouts
- Long-running scheduler behavior (intervals, graceful shutdown)
- Service installation (systemd / Windows service)
