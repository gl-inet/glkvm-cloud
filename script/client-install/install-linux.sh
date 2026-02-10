#!/bin/sh
# ============================================================================
# rtty-go client one-click installer for Linux
# Supports: x86_64, x86, arm64, armv7 (Ubuntu/Debian/CentOS/OpenWrt/Raspberry Pi)
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

# ========================== Helper Functions =================================

log_info()  { printf "\033[32m[INFO]\033[0m  %s\n" "$1"; }
log_warn()  { printf "\033[33m[WARN]\033[0m  %s\n" "$1"; }
log_error() { printf "\033[31m[ERROR]\033[0m %s\n" "$1"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

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
  $0 -h 107.173.152.173 -t lHEP7GyyGt4S18KlyikfpvzdZTVxnD8v
EOF
    exit 1
}

# ========================== Parse Arguments ==================================

UNINSTALL=0

while [ $# -gt 0 ]; do
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

# ========================== Detect Platform ==================================

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)       echo "linux-amd64" ;;
        i386|i486|i586|i686) echo "linux-386" ;;
        aarch64|arm64)      echo "linux-arm64" ;;
        armv7*|armhf)       echo "linux-armv7" ;;
        armv6*)             echo "linux-armv7" ;;
        *)  log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

# ========================== Detect Init System ===============================

detect_init() {
    if [ -f /etc/openwrt_release ]; then
        echo "procd"
    elif command_exists systemctl && systemctl --version >/dev/null 2>&1; then
        echo "systemd"
    elif command_exists rc-service; then
        echo "openrc"
    else
        echo "sysvinit"
    fi
}

# ========================== Get MAC Address ==================================

get_mac() {
    MAC=""

    # Method 1: OpenWrt GL.iNet specific
    if [ -f /proc/gl-hw-info/device_mac ]; then
        MAC=$(cat /proc/gl-hw-info/device_mac 2>/dev/null)
    fi

    # Method 2: ip command (most modern Linux)
    if [ -z "$MAC" ] && command_exists ip; then
        MAC=$(ip link show 2>/dev/null | awk '
            /^[0-9]+:/ { iface=$2; gsub(/:$/, "", iface) }
            /ether/ && iface !~ /^(lo|docker|br-|veth|virbr)/ { print $2; exit }
        ')
    fi

    # Method 3: /sys/class/net (fallback)
    if [ -z "$MAC" ]; then
        for iface_path in /sys/class/net/*; do
            iface=$(basename "$iface_path")
            case "$iface" in lo|docker*|br-*|veth*|virbr*) continue ;; esac
            if [ -f "${iface_path}/address" ]; then
                addr=$(cat "${iface_path}/address" 2>/dev/null)
                if [ -n "$addr" ] && [ "$addr" != "00:00:00:00:00:00" ]; then
                    MAC="$addr"
                    break
                fi
            fi
        done
    fi

    # Method 4: Generate random locally-administered MAC
    if [ -z "$MAC" ] || [ "$MAC" = "00:00:00:00:00:00" ]; then
        log_warn "Could not detect MAC address, generating random one"
        MAC=$(od -An -N5 -tx1 /dev/urandom | tr -d ' \n' | sed 's/\(..\)/\1:/g; s/:$//')
        MAC="02:${MAC}"
    fi

    echo "$MAC"
}

# ========================== Generate Device ID ===============================

gen_device_id() {
    # OpenWrt GL.iNet specific
    if [ -f /proc/gl-hw-info/device_ddns ]; then
        cat /proc/gl-hw-info/device_ddns
        return
    fi

    # Generate random ID: 8 chars alphanumeric
    od -An -N4 -tx1 /dev/urandom | tr -d ' \n' | cut -c1-8
}

# ========================== Download Binary ==================================

download_file() {
    url="$1"
    dest="$2"

    if command_exists curl; then
        curl -fsSL -o "$dest" "$url"
    elif command_exists wget; then
        wget -qO "$dest" "$url"
    else
        log_error "Neither curl nor wget found. Please install one."
        exit 1
    fi
}

# ========================== Uninstall ========================================

do_uninstall() {
    log_info "Uninstalling rtty-go client..."

    INIT_SYSTEM=$(detect_init)

    case "$INIT_SYSTEM" in
        systemd)
            systemctl stop rtty-go 2>/dev/null || true
            systemctl disable rtty-go 2>/dev/null || true
            rm -f /etc/systemd/system/rtty-go.service
            systemctl daemon-reload 2>/dev/null || true
            ;;
        procd)
            /etc/init.d/rtty-go stop 2>/dev/null || true
            /etc/init.d/rtty-go disable 2>/dev/null || true
            rm -f /etc/init.d/rtty-go
            ;;
        openrc)
            rc-service rtty-go stop 2>/dev/null || true
            rc-update del rtty-go default 2>/dev/null || true
            rm -f /etc/init.d/rtty-go
            ;;
        *)
            crontab -l 2>/dev/null | grep -v "rtty-go" | crontab - 2>/dev/null || true
            pkill -f "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true
            ;;
    esac

    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "${CONFIG_DIR}"

    log_info "Uninstall complete."
    exit 0
}

# ========================== Service Setup ====================================

setup_systemd() {
    log_info "Setting up systemd service..."

    cat > /etc/systemd/system/rtty-go.service <<EOF
[Unit]
Description=rtty-go remote terminal client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${CONFIG_FILE}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} ${RTTY_SSL} -a -I \${DEVICE_ID} -h \${RTTY_HOST} -p \${RTTY_PORT} -t \${RTTY_TOKEN} -d \${DEVICE_MAC}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable rtty-go
    systemctl restart rtty-go
    log_info "systemd service started."
}

setup_procd() {
    log_info "Setting up OpenWrt procd service..."

    cat > /etc/init.d/rtty-go <<'INITEOF'
#!/bin/sh /etc/rc.common

START=99
STOP=10

USE_PROCD=1

start_service() {
    . /etc/rtty-go/env

    procd_open_instance
    procd_set_param command /usr/local/bin/rtty-go \
INITEOF

    # Append the dynamic parts
    cat >> /etc/init.d/rtty-go <<EOF
        ${RTTY_SSL} -a \\
EOF

    cat >> /etc/init.d/rtty-go <<'INITEOF'
        -I "$DEVICE_ID" \
        -h "$RTTY_HOST" \
        -p "$RTTY_PORT" \
        -t "$RTTY_TOKEN" \
        -d "$DEVICE_MAC"
    procd_set_param respawn 3600 5 0
    procd_close_instance
}
INITEOF

    chmod +x /etc/init.d/rtty-go
    /etc/init.d/rtty-go enable
    /etc/init.d/rtty-go restart
    log_info "OpenWrt procd service started."
}

setup_openrc() {
    log_info "Setting up OpenRC service..."

    cat > /etc/init.d/rtty-go <<EOF
#!/sbin/openrc-run

description="rtty-go remote terminal client"
command="${INSTALL_DIR}/${BINARY_NAME}"
command_args="${RTTY_SSL} -a -I \${DEVICE_ID} -h \${RTTY_HOST} -p \${RTTY_PORT} -t \${RTTY_TOKEN} -d \${DEVICE_MAC}"
command_background=true
pidfile="/run/rtty-go.pid"

depend() {
    need net
    after firewall
}

start_pre() {
    . ${CONFIG_FILE}
}
EOF

    chmod +x /etc/init.d/rtty-go
    rc-update add rtty-go default
    rc-service rtty-go restart
    log_info "OpenRC service started."
}

setup_crontab() {
    log_info "Setting up crontab auto-start (fallback)..."

    # Create a wrapper script
    cat > "${CONFIG_DIR}/start.sh" <<EOF
#!/bin/sh
. ${CONFIG_FILE}
if ! pgrep -f "${INSTALL_DIR}/${BINARY_NAME}" >/dev/null 2>&1; then
    ${INSTALL_DIR}/${BINARY_NAME} ${RTTY_SSL} -a -I "\${DEVICE_ID}" -h "\${RTTY_HOST}" -p "\${RTTY_PORT}" -t "\${RTTY_TOKEN}" -d "\${DEVICE_MAC}" &
fi
EOF
    chmod +x "${CONFIG_DIR}/start.sh"

    # Add to crontab: check every minute + run at reboot
    (crontab -l 2>/dev/null | grep -v "rtty-go"; \
     echo "@reboot sleep 10 && ${CONFIG_DIR}/start.sh"; \
     echo "*/5 * * * * ${CONFIG_DIR}/start.sh") | crontab -

    # Start now
    "${CONFIG_DIR}/start.sh"
    log_info "Crontab auto-start configured."
}

# ========================== Main =============================================

# Handle uninstall
[ "$UNINSTALL" -eq 1 ] && do_uninstall

# Validate required parameters
[ -z "$RTTY_HOST" ]  && log_error "Server host is required (-h)" && usage
[ -z "$RTTY_TOKEN" ] && log_error "Token is required (-t)" && usage

# Check root
if [ "$(id -u)" -ne 0 ]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

PLATFORM=$(detect_arch)
INIT_SYSTEM=$(detect_init)

# Reuse existing device ID and MAC if config exists (preserve identity across reinstalls)
# Only extract DEVICE_ID and DEVICE_MAC, do NOT source the whole file
# (sourcing would overwrite RTTY_HOST/RTTY_PORT/RTTY_TOKEN from CLI args)
EXISTING_ID=""
EXISTING_MAC=""
if [ -f "$CONFIG_FILE" ]; then
    EXISTING_ID=$(grep '^DEVICE_ID=' "$CONFIG_FILE" | head -1 | cut -d'=' -f2- | tr -d '"')
    EXISTING_MAC=$(grep '^DEVICE_MAC=' "$CONFIG_FILE" | head -1 | cut -d'=' -f2- | tr -d '"')
fi

DEVICE_ID="${EXISTING_ID:-$(gen_device_id)}"
DEVICE_MAC="${EXISTING_MAC:-$(get_mac)}"

log_info "========================================"
log_info "  rtty-go Client Installer for Linux"
log_info "========================================"
log_info "Platform:    ${PLATFORM}"
log_info "Init system: ${INIT_SYSTEM}"
log_info "Device ID:   ${DEVICE_ID}"
log_info "MAC address: ${DEVICE_MAC}"
log_info "Server:      ${RTTY_HOST}:${RTTY_PORT}"
log_info "========================================"

# Determine download URL
if [ -z "$DOWNLOAD_BASE_URL" ]; then
    DOWNLOAD_BASE_URL="https://kvm-cloud.gl-inet.com/selfhost/clients"
fi
FILE_URL="${DOWNLOAD_BASE_URL}/${BINARY_NAME}-${PLATFORM}"

# Stop existing service before replacing binary and config
case "$INIT_SYSTEM" in
    systemd)  systemctl stop rtty-go 2>/dev/null || true ;;
    procd)    /etc/init.d/rtty-go stop 2>/dev/null || true ;;
    openrc)   rc-service rtty-go stop 2>/dev/null || true ;;
    *)        pkill -f "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true ;;
esac

# Download binary
log_info "Downloading ${BINARY_NAME}-${PLATFORM}..."
download_file "$FILE_URL" "/tmp/${BINARY_NAME}"
chmod +x "/tmp/${BINARY_NAME}"

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

# Setup auto-start service
case "$INIT_SYSTEM" in
    systemd)  setup_systemd  ;;
    procd)    setup_procd    ;;
    openrc)   setup_openrc   ;;
    *)        setup_crontab  ;;
esac

log_info "========================================"
log_info "  Installation complete!"
log_info "  Device ID:   ${DEVICE_ID}"
log_info "  MAC address: ${DEVICE_MAC}"
log_info "========================================"
