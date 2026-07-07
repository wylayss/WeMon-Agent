#!/bin/bash
# ==============================================================================
# WeMon Agent Installer Script
# ==============================================================================
set -euo pipefail

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD_WHITE='\033[1;37m'
NC='\033[0m' # No Color

SERVER_URL=""
NODE_TOKEN=""
CONFIG_DIR="/etc/wemon-agent"
CONFIG_FILE="$CONFIG_DIR/config.json"
BINARY_PATH="/usr/local/bin/wemon-agent"
SERVICE_PATH="/etc/systemd/system/wemon-agent.service"

# Print Help Guide
print_help() {
    echo -e "${BOLD_WHITE}Usage: sudo ./install.sh [options]${NC}"
    echo ""
    echo "Options:"
    echo "  -s, --server <url>       Specify WeMon server URL (e.g. https://wemon.werix.net)"
    echo "  -t, --token <token>      Specify WeMon node authentication token"
    echo "  -h, --help               Display this help guide"
    echo ""
    echo "Example:"
    echo "  sudo ./install.sh -s https://wemon.werix.net -t wmn_tok_abc123xyz"
}

# Parse CLI arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--server)
            SERVER_URL="$2"
            shift 2
            ;;
        -t|--token)
            NODE_TOKEN="$2"
            shift 2
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Unknown argument: $1${NC}"
            print_help
            exit 1
            ;;
    esac
done

# Ensure script is run with root privileges
if [ "${EUID:-$(id -u)}" -ne 0 ]; then
    echo -e "${RED}❌ Error: This installer must be run with root privileges (sudo).${NC}"
    exit 1
fi

# Validate inputs
if [ -z "$SERVER_URL" ] || [ -z "$NODE_TOKEN" ]; then
    echo -e "${RED}❌ Error: Both --server and --token arguments are required.${NC}"
    print_help
    exit 1
fi

echo -e "${BLUE}⚙️ Installing WeMon Monitoring Agent...${NC}"

# 1. Detect latest release tag dynamically (to support pre-releases like v1.0.0-prototype)
TAG=$(curl -s https://api.github.com/repos/wylayss/WeMon-Agent/releases | grep -m 1 '"tag_name":' | cut -d '"' -f 4 || true)
if [ -z "$TAG" ]; then
    TAG="v1.0.0-prototype"
fi

ARCH=$(uname -m)
BINARY_URL=""

if [ "$ARCH" = "x86_64" ]; then
    echo -e "   Detected Architecture: ${BOLD_WHITE}x86_64${NC}"
    BINARY_URL="https://github.com/wylayss/WeMon-Agent/releases/download/${TAG}/wemon-agent-linux-amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    echo -e "   Detected Architecture: ${BOLD_WHITE}ARM64${NC}"
    BINARY_URL="https://github.com/wylayss/WeMon-Agent/releases/download/${TAG}/wemon-agent-linux-arm64"
else
    echo -e "${RED}❌ Unsupported architecture: $ARCH. WeMon-Agent only supports x86_64 and ARM64.${NC}"
    exit 1
fi

# Check if WeMon Agent is already installed and active/enabled
if systemctl is-active --quiet wemon-agent 2>/dev/null || systemctl is-enabled --quiet wemon-agent 2>/dev/null; then
    echo -e "${YELLOW}⚠️ Existing WeMon Agent service detected. Stopping service to replace binary...${NC}"
    systemctl stop wemon-agent || true
fi

# 2. Download pre-compiled binary
echo -e "📥 Downloading agent binary..."
echo -e "   URL: ${BOLD_WHITE}$BINARY_URL${NC}"
# Fallback to local build binary if downloading fails (useful for local development install tests)
if ! curl -sL -f -o "$BINARY_PATH" "$BINARY_URL"; then
    echo -e "${YELLOW}⚠️ Failed to download binary from GitHub. Checking for local build binary...${NC}"
    if [ -f "./wemon-agent" ]; then
        echo -e "   Found local build binary. Copying to $BINARY_PATH..."
        cp "./wemon-agent" "$BINARY_PATH"
    else
        echo -e "${RED}❌ Error: Could not download binary and no local 'wemon-agent' binary found.${NC}"
        exit 1
    fi
fi

chmod +x "$BINARY_PATH"
echo -e "${GREEN}✅ Binary installed to $BINARY_PATH${NC}"

# 3. Create configuration directory & file
echo -e "📝 Generating configuration file..."
mkdir -p "$CONFIG_DIR"
cat <<EOF > "$CONFIG_FILE"
{
  "server_url": "$SERVER_URL",
  "node_token": "$NODE_TOKEN",
  "interval_seconds": 5,
  "insecure_skip_verify": false
}
EOF
chmod 600 "$CONFIG_FILE"
echo -e "${GREEN}✅ Configuration saved to $CONFIG_FILE${NC}"

# 4. Register systemd daemon service
echo -e "⚙️ Registering systemd service..."
cat <<EOF > "$SERVICE_PATH"
[Unit]
Description=WeMon Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=$BINARY_PATH --config $CONFIG_FILE
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable wemon-agent.service
systemctl restart wemon-agent.service

echo -e "${GREEN}=========================================================================${NC}"
echo -e "${GREEN} 🎉 WeMon Agent Successfully Installed and Started!${NC}"
echo -e "    Check status: ${BOLD_WHITE}systemctl status wemon-agent.service${NC}"
echo -e "    View logs:    ${BOLD_WHITE}journalctl -u wemon-agent.service -f${NC}"
echo -e "${GREEN}=========================================================================${NC}"
