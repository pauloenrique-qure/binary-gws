#!/bin/bash
set -e

echo "Gateway Agent - Linux Installation Script"
echo "=========================================="

if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

INSTALL_DIR="/opt/gw-agent"
CONFIG_DIR="/etc/gw-agent"
LOG_DIR="/var/log/gw-agent"
BINARY_NAME="gw-agent"
SERVICE_USER="gwagent"
SERVICE_NAME="gw-agent"

ARCH=$(uname -m)
case $ARCH in
    x86_64)
        DIST_DIR="linux_amd64"
        ;;
    aarch64|arm64)
        DIST_DIR="linux_arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

BINARY_PATH="./dist/${DIST_DIR}/${BINARY_NAME}"

if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    echo "Please run 'make build-all' first"
    exit 1
fi

echo "Creating user $SERVICE_USER..."
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
    echo "User $SERVICE_USER created"
else
    echo "User $SERVICE_USER already exists"
fi

echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$LOG_DIR"

echo "Installing binary..."
cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"
chown root:root "$INSTALL_DIR/$BINARY_NAME"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "Creating sample configuration..."
    cat > "$CONFIG_DIR/config.yaml" <<EOF
# Gateway Agent Configuration
# Replace the values below with your actual settings

uuid: "REPLACE_WITH_GATEWAY_UUID"
client_id: "REPLACE_WITH_CLIENT_ID"
site_id: "REPLACE_WITH_SITE_ID"
api_url: "https://api.example.com/heartbeat"

auth:
  token_current: "REPLACE_WITH_CURRENT_TOKEN"
  # token_grace: "optional-grace-token-for-rotation"

# platform:
#   platform_override: "ubuntu"  # Optional: raspberry_pi, ubuntu, windows, vm, linux

intervals:
  heartbeat_seconds: 60
  compute_seconds: 120

tls:
  # ca_bundle_path: "/path/to/ca-bundle.pem"  # Optional
  insecure_skip_verify: false
EOF
    chmod 600 "$CONFIG_DIR/config.yaml"
    chown "$SERVICE_USER:$SERVICE_USER" "$CONFIG_DIR/config.yaml"
    echo "Sample configuration created at $CONFIG_DIR/config.yaml"
    echo "WARNING: You MUST edit this file and set your actual values!"
else
    echo "Configuration file already exists at $CONFIG_DIR/config.yaml"
fi

chown "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR"

echo "Installing systemd service..."
cp ./scripts/gw-agent.service /etc/systemd/system/
systemctl daemon-reload

echo "Installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit the configuration file: $CONFIG_DIR/config.yaml"
echo "2. Enable the service: systemctl enable $SERVICE_NAME"
echo "3. Start the service: systemctl start $SERVICE_NAME"
echo "4. Check status: systemctl status $SERVICE_NAME"
echo "5. View logs: journalctl -u $SERVICE_NAME -f"
echo ""
echo "To test manually before enabling service:"
echo "  sudo -u $SERVICE_USER $INSTALL_DIR/$BINARY_NAME --config $CONFIG_DIR/config.yaml --once"
