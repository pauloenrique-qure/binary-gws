# Build Instructions

## Prerequisites

- Go 1.21 or later
- Make
- Git (optional, for version information)

## Building

### Quick Build (Current Platform)

```bash
make build
```

Output: `./dist/gw-agent`

### Build All Platform Binaries

```bash
make build-all
```

Outputs:
- `./dist/linux_amd64/gw-agent` - Ubuntu/Linux x86_64
- `./dist/linux_arm64/gw-agent` - Raspberry Pi ARM64
- `./dist/windows_amd64/gw-agent.exe` - Windows 11 x86_64

### With Version Information

```bash
# Version from git tags
VERSION=$(git describe --tags --always) make build-all

# Or set manually
VERSION=1.0.0 COMMIT=$(git rev-parse --short HEAD) make build-all
```

### Other Targets

```bash
make test      # Run all tests with race detection
make lint      # Run linters (golangci-lint or go vet)
make clean     # Remove build artifacts
make deps      # Download and tidy dependencies
```

## Binary Details

### Linux Binaries

- **Architecture**: amd64 (Ubuntu), arm64 (Raspberry Pi)
- **CGO**: Disabled (static binary)
- **Size**: ~6-7 MB
- **Stripped**: Yes (ldflags -s -w)
- **Dependencies**: None (statically linked)

### Windows Binary

- **Architecture**: amd64
- **CGO**: Disabled
- **Size**: ~6-7 MB
- **Stripped**: Yes
- **Dependencies**: None (no DLL dependencies)

## Cross-Compilation

The Makefile handles cross-compilation automatically:

```makefile
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ...
```

You can build from any platform (macOS, Linux, Windows) and produce binaries for all target platforms.

## Embedded Version Information

Version info is embedded at build time via ldflags:

```bash
-X main.Version=$(VERSION)
-X main.Commit=$(COMMIT)
-X main.BuildDate=$(BUILD_DATE)
```

Check version: `./gw-agent --print-version`

## Testing Binaries

### Functional Test

```bash
# Create test config
cat > test-config.yaml <<EOF
uuid: "test-gateway"
client_id: "test"
site_id: "test"
api_url: "https://httpbin.org/post"
auth:
  token_current: "test-token"
EOF

# Dry run (no network call)
./dist/gw-agent --config test-config.yaml --dry-run

# Single heartbeat test (will fail to httpbin but tests full flow)
./dist/gw-agent --config test-config.yaml --once
```

### Platform Verification

| Platform | How to Test |
|----------|-------------|
| Ubuntu | Copy `dist/linux_amd64/gw-agent` to Ubuntu VM/server and run |
| Raspberry Pi | Copy `dist/linux_arm64/gw-agent` to RPi and run |
| Windows | Copy `dist/windows_amd64/gw-agent.exe` to Windows machine and run |

## Dependencies

Go dependencies (managed by go.mod):

```
github.com/shirou/gopsutil/v3  # System metrics collection
gopkg.in/yaml.v3               # YAML config parsing
```

All dependencies are vendored into the binary at build time (CGO_ENABLED=0).

## Build Environment

The agent builds cleanly on:
- macOS (M1/Intel)
- Linux (any architecture)
- Windows
- CI/CD systems (GitHub Actions, GitLab CI, etc.)

No special build environment required beyond Go toolchain.

## CI/CD Integration

Example GitHub Actions workflow:

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

## Distribution

Recommended distribution methods:
1. **Direct binary download** - Upload to S3/CDN
2. **Package managers** - Create .deb, .rpm, .msi wrappers
3. **Container images** - Use scratch or distroless base
4. **Tar archives** - Bundle binary + scripts + example config

Example tar package:

```bash
tar czf gw-agent-linux-amd64.tar.gz \
  -C dist/linux_amd64 gw-agent \
  -C ../../scripts gw-agent.service install-linux.sh \
  -C .. config.example.yaml README.md
```
