#!/bin/bash
set -e

# --- Colors and logging ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

# --- Detect OS and architecture ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    log_error "Unsupported operating system: $OS"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    log_error "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# --- Require root on Linux ---
if [ "$OS" = "linux" ] && [ "$EUID" -ne 0 ]; then
  log_error "This script must be run as root on Linux. Use sudo."
  exit 1
fi

# --- Defaults ---
REPO="snana7mi/conchtalk-dlc"
SERVICE_NAME="conchtalk-dlc"
DLC_TOKEN=""
DLC_SERVER="wss://api.conch-talk.com/relay"
DLC_VERSION="latest"

if [ "$OS" = "linux" ]; then
  INSTALL_DIR="/opt/conchtalk"
else
  INSTALL_DIR="/usr/local/conchtalk"
fi

# --- Parse arguments ---
while [ $# -gt 0 ]; do
  case "$1" in
    --token)
      DLC_TOKEN="$2"
      shift 2
      ;;
    --server)
      DLC_SERVER="$2"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    --version)
      DLC_VERSION="$2"
      shift 2
      ;;
    *)
      log_error "Unknown argument: $1"
      echo "Usage: $0 --token <TOKEN> [--server <URL>] [--install-dir <DIR>] [--version <VER>]"
      exit 1
      ;;
  esac
done

if [ -z "$DLC_TOKEN" ]; then
  log_error "--token is required"
  echo "Usage: $0 --token <TOKEN> [--server <URL>] [--install-dir <DIR>] [--version <VER>]"
  exit 1
fi

TARGET_DIR="$INSTALL_DIR"

# --- Print config ---
log_info "OS: $OS | Arch: $ARCH"
log_info "Install dir: $TARGET_DIR"
log_info "Server: $DLC_SERVER"
log_info "Version: $DLC_VERSION"

# --- Stop existing service ---
log_info "Stopping existing service (if any)..."
if [ "$OS" = "linux" ]; then
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  systemctl disable "$SERVICE_NAME" 2>/dev/null || true
else
  launchctl unload /Library/LaunchDaemons/com.conchtalk.dlc.plist 2>/dev/null || true
fi

# --- Determine download URL ---
if [ "$DLC_VERSION" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${SERVICE_NAME}-${OS}-${ARCH}"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${DLC_VERSION}/${SERVICE_NAME}-${OS}-${ARCH}"
fi

# --- Create directories ---
log_info "Creating directories..."
mkdir -p "$TARGET_DIR"
mkdir -p "$HOME/.conchtalk/skills"

# --- Download binary ---
log_info "Downloading binary from $DOWNLOAD_URL ..."
if command -v curl &>/dev/null; then
  curl -fSL -o "$TARGET_DIR/$SERVICE_NAME" "$DOWNLOAD_URL"
elif command -v wget &>/dev/null; then
  wget -q -O "$TARGET_DIR/$SERVICE_NAME" "$DOWNLOAD_URL"
else
  log_error "Neither curl nor wget found. Please install one and retry."
  exit 1
fi
chmod +x "$TARGET_DIR/$SERVICE_NAME"
log_success "Binary installed to $TARGET_DIR/$SERVICE_NAME"

# --- Save token ---
log_info "Saving token..."
echo "$DLC_TOKEN" > "$HOME/.conchtalk/token"
chmod 600 "$HOME/.conchtalk/token"
log_success "Token saved to ~/.conchtalk/token"

# --- Install service ---
if [ "$OS" = "linux" ]; then
  log_info "Installing systemd service..."

  # Write token to environment file (not in unit file)
  cat > /etc/conchtalk-dlc.env <<ENVEOF
DLC_TOKEN=${DLC_TOKEN}
ENVEOF
  chmod 600 /etc/conchtalk-dlc.env
  log_success "Token written to /etc/conchtalk-dlc.env"

  cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=ConchTalk DLC Daemon
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/conchtalk-dlc.env
ExecStart=${TARGET_DIR}/conchtalk-dlc --token \${DLC_TOKEN} --server ${DLC_SERVER}
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  systemctl start "$SERVICE_NAME"
  log_success "systemd service installed and started"

else
  log_info "Installing launchd daemon..."
  cat > /Library/LaunchDaemons/com.conchtalk.dlc.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.conchtalk.dlc</string>
    <key>ProgramArguments</key>
    <array>
        <string>${TARGET_DIR}/conchtalk-dlc</string>
        <string>--token</string>
        <string>${DLC_TOKEN}</string>
        <string>--server</string>
        <string>${DLC_SERVER}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/conchtalk-dlc.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/conchtalk-dlc.err</string>
</dict>
</plist>
EOF

  launchctl load /Library/LaunchDaemons/com.conchtalk.dlc.plist
  log_success "launchd daemon installed and loaded"
fi

log_success "ConchTalk DLC installation complete!"
