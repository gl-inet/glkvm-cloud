#!/bin/bash
# ============================================================================
# rtty-go client one-click installer for macOS
# Supports: Intel (amd64) / Apple Silicon (arm64)
# ============================================================================
set -e

# --------------- Configuration (can be pre-filled by server) ----------------
RTTY_HOST=""
RTTY_PORT="5912"
RTTY_TOKEN=""
RTTY_SSL="-s -x"
DOWNLOAD_BASE_URL="https://kvm-cloud.gl-inet.com/selfhost/clients"
# ----------------------------------------------------------------------------

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="rtty-go"
CONFIG_DIR="/etc/rtty-go"
CONFIG_FILE="${CONFIG_DIR}/env"
PLIST_NAME="com.glkvm.rtty-go"
PLIST_PATH="/Library/LaunchDaemons/${PLIST_NAME}.plist"

# ========================== Helper Functions =================================

log_info()  { printf "\033[32m[INFO]\033[0m  %s\n" "$1"; }
log_warn()  { printf "\033[33m[WARN]\033[0m  %s\n" "$1"; }
log_error() { printf "\033[31m[ERROR]\033[0m %s\n" "$1"; }

usage() {
    cat <<EOF
Usage: $0 -h <host> -t <token> [-p <port>] [-u <download_base_url>]

Options:
  -h    Server host or IP address (required)
  -t    Authorization token (required)
  -p    Server port (default: 5912)
  -u    Download base URL for binaries
  --uninstall   Uninstall rtty-go client

Example:
  sudo $0 -h 107.173.152.173 -t lHEP7GyyGt4S18KlyikfpvzdZTVxnD8v
EOF
    exit 1
}

# ========================== Parse Arguments ==================================

UNINSTALL=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h) RTTY_HOST="$2";          shift 2 ;;
        -p) RTTY_PORT="$2";          shift 2 ;;
        -t) RTTY_TOKEN="$2";         shift 2 ;;
        -u) DOWNLOAD_BASE_URL="$2";  shift 2 ;;
        --uninstall) UNINSTALL=1;    shift   ;;
        *)  usage ;;
    esac
done

# Strip leading colon from port (e.g. ":5912" -> "5912")
RTTY_PORT="${RTTY_PORT#:}"

# ========================== Detect Architecture ==============================

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  echo "darwin-amd64" ;;
        arm64)   echo "darwin-arm64" ;;
        *)  log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

# ========================== Get MAC Address ==================================

get_mac() {
    MAC=""

    # Method 1: en0 (usually the primary interface)
    if command -v ifconfig &>/dev/null; then
        MAC=$(ifconfig en0 2>/dev/null | awk '/ether/{print $2}')
    fi

    # Method 2: networksetup
    if [[ -z "$MAC" ]] && command -v networksetup &>/dev/null; then
        MAC=$(networksetup -getmacaddress Wi-Fi 2>/dev/null | awk '{print $3}')
        if [[ "$MAC" == *"not"* ]] || [[ -z "$MAC" ]]; then
            MAC=$(networksetup -getmacaddress Ethernet 2>/dev/null | awk '{print $3}')
        fi
    fi

    # Method 3: Generate random locally-administered MAC
    if [[ -z "$MAC" ]] || [[ "$MAC" == "00:00:00:00:00:00" ]]; then
        log_warn "Could not detect MAC address, generating random one"
        MAC=$(printf '02:%02x:%02x:%02x:%02x:%02x' $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)))
    fi

    echo "$MAC"
}

# ========================== Generate Device ID ===============================

gen_device_id() {
    uuidgen | tr -d '-' | tr 'A-F' 'a-f' | cut -c1-8
}

# ========================== Uninstall ========================================

do_uninstall() {
    log_info "Uninstalling rtty-go client..."

    # Stop and remove LaunchDaemon
    if [[ -f "$PLIST_PATH" ]]; then
        launchctl bootout system "$PLIST_PATH" 2>/dev/null || true
        rm -f "$PLIST_PATH"
    fi

    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "${CONFIG_DIR}"

    log_info "Uninstall complete."
    exit 0
}

# ========================== Main =============================================

# Handle uninstall
[[ "$UNINSTALL" -eq 1 ]] && do_uninstall

# Validate required parameters
[[ -z "$RTTY_HOST" ]]  && log_error "Server host is required (-h)" && usage
[[ -z "$RTTY_TOKEN" ]] && log_error "Token is required (-t)" && usage

# Check root
if [[ $(id -u) -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

PLATFORM=$(detect_arch)

# Reuse existing device ID and MAC if config exists (preserve identity across reinstalls)
# Only extract DEVICE_ID and DEVICE_MAC, do NOT source the whole file
# (sourcing would overwrite RTTY_HOST/RTTY_PORT/RTTY_TOKEN from CLI args)
EXISTING_ID=""
EXISTING_MAC=""
if [[ -f "$CONFIG_FILE" ]]; then
    EXISTING_ID=$(grep '^DEVICE_ID=' "$CONFIG_FILE" | head -1 | cut -d'=' -f2- | tr -d '"')
    EXISTING_MAC=$(grep '^DEVICE_MAC=' "$CONFIG_FILE" | head -1 | cut -d'=' -f2- | tr -d '"')
fi

DEVICE_ID="${EXISTING_ID:-$(gen_device_id)}"
DEVICE_MAC="${EXISTING_MAC:-$(get_mac)}"

log_info "========================================"
log_info "  rtty-go Client Installer for macOS"
log_info "========================================"
log_info "Platform:    ${PLATFORM}"
log_info "Device ID:   ${DEVICE_ID}"
log_info "MAC address: ${DEVICE_MAC}"
log_info "Server:      ${RTTY_HOST}:${RTTY_PORT}"
log_info "========================================"

# Determine download URL
if [[ -z "$DOWNLOAD_BASE_URL" ]]; then
    DOWNLOAD_BASE_URL="https://kvm-cloud.gl-inet.com/selfhost/clients"
fi
FILE_URL="${DOWNLOAD_BASE_URL}/${BINARY_NAME}-${PLATFORM}"

# Stop existing service before replacing binary and config
if [[ -f "$PLIST_PATH" ]]; then
    launchctl bootout system "$PLIST_PATH" 2>/dev/null || true
fi

# Download binary
log_info "Downloading ${BINARY_NAME}-${PLATFORM}..."
curl -fsSL -o "/tmp/${BINARY_NAME}" "$FILE_URL"
chmod +x "/tmp/${BINARY_NAME}"

# Remove quarantine attribute (macOS Gatekeeper)
xattr -d com.apple.quarantine "/tmp/${BINARY_NAME}" 2>/dev/null || true

# Install binary
mkdir -p "$INSTALL_DIR"
mv -f "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
log_info "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

# Save configuration
mkdir -p "$CONFIG_DIR"
cat > "$CONFIG_FILE" <<EOF
RTTY_HOST="${RTTY_HOST}"
RTTY_PORT="${RTTY_PORT}"
RTTY_TOKEN="${RTTY_TOKEN}"
DEVICE_ID="${DEVICE_ID}"
DEVICE_MAC="${DEVICE_MAC}"
EOF
chmod 600 "$CONFIG_FILE"
log_info "Configuration saved to ${CONFIG_FILE}"

# Create LaunchDaemon for auto-start on boot
log_info "Setting up LaunchDaemon for auto-start..."

cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_NAME}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BINARY_NAME}</string>
        <string>-s</string>
        <string>-x</string>
        <string>-a</string>
        <string>-I</string>
        <string>${DEVICE_ID}</string>
        <string>-h</string>
        <string>${RTTY_HOST}</string>
        <string>-p</string>
        <string>${RTTY_PORT}</string>
        <string>-t</string>
        <string>${RTTY_TOKEN}</string>
        <string>-d</string>
        <string>${DEVICE_MAC}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/rtty-go.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/rtty-go.log</string>
</dict>
</plist>
EOF

chmod 644 "$PLIST_PATH"
launchctl bootstrap system "$PLIST_PATH"

log_info "========================================"
log_info "  Installation complete!"
log_info "  Device ID:   ${DEVICE_ID}"
log_info "  MAC address: ${DEVICE_MAC}"
log_info ""
log_info "  Manage service:"
log_info "    Stop:    sudo launchctl bootout system ${PLIST_PATH}"
log_info "    Start:   sudo launchctl bootstrap system ${PLIST_PATH}"
log_info "    Log:     tail -f /var/log/rtty-go.log"
log_info "    Remove:  sudo $0 --uninstall"
log_info "========================================"
