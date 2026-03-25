# ============================================================================
# rtty-go client one-click installer for Windows (PowerShell)
# Supports: x86_64 (amd64)
# Run as Administrator: powershell -ExecutionPolicy Bypass -File install-windows.ps1
# ============================================================================

param(
    [Parameter(Mandatory=$false)][string]$Host_Addr = "",
    [Parameter(Mandatory=$false)][string]$Port = "5912",
    [Parameter(Mandatory=$false)][string]$Token = "",
    [Parameter(Mandatory=$false)][string]$DownloadBaseUrl = "",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"

# --------------- Configuration (can be pre-filled by server) ----------------
$RTTY_HOST  = if ($Host_Addr)      { $Host_Addr }      else { "" }
$RTTY_PORT  = if ($Port)           { $Port }            else { "5912" }
$RTTY_TOKEN = if ($Token)          { $Token }           else { "" }
$DOWNLOAD_BASE_URL = if ($DownloadBaseUrl) { $DownloadBaseUrl } else { "https://kvm-cloud.gl-inet.com/selfhost/clients" }
# ----------------------------------------------------------------------------

$INSTALL_DIR   = "$env:ProgramFiles\rtty-go"
$BINARY_NAME   = "rtty-go.exe"
$CONFIG_FILE   = "$INSTALL_DIR\config.env"
$TASK_NAME     = "rtty-go"
$LOG_FILE      = "$INSTALL_DIR\rtty-go.log"

# ========================== Helper Functions =================================

function Log-Info  { param([string]$msg) Write-Host "[INFO]  $msg" -ForegroundColor Green }
function Log-Warn  { param([string]$msg) Write-Host "[WARN]  $msg" -ForegroundColor Yellow }
function Log-Error { param([string]$msg) Write-Host "[ERROR] $msg" -ForegroundColor Red }

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# ========================== Get MAC Address ==================================

function Get-PhysicalMac {
    $mac = ""

    try {
        $adapter = Get-NetAdapter -Physical -ErrorAction SilentlyContinue |
            Where-Object { $_.Status -eq 'Up' } |
            Select-Object -First 1

        if ($adapter) {
            $mac = $adapter.MacAddress.ToLower() -replace '-', ':'
        }
    } catch {}

    # Fallback: try all adapters
    if (-not $mac) {
        try {
            $adapter = Get-NetAdapter -ErrorAction SilentlyContinue |
                Where-Object { $_.Status -eq 'Up' -and $_.InterfaceDescription -notmatch 'Virtual|Hyper-V|VPN|Loopback' } |
                Select-Object -First 1

            if ($adapter) {
                $mac = $adapter.MacAddress.ToLower() -replace '-', ':'
            }
        } catch {}
    }

    # Generate random MAC if all methods fail
    if (-not $mac) {
        Log-Warn "Could not detect MAC address, generating random one"
        $bytes = New-Object byte[] 5
        (New-Object Random).NextBytes($bytes)
        $mac = "02:" + (($bytes | ForEach-Object { $_.ToString("x2") }) -join ":")
    }

    return $mac
}

# ========================== Generate Device ID ===============================

function New-DeviceId {
    $bytes = New-Object byte[] 4
    (New-Object Random).NextBytes($bytes)
    return ($bytes | ForEach-Object { $_.ToString("x2") }) -join ""
}

# ========================== Uninstall ========================================

function Invoke-Uninstall {
    Log-Info "Uninstalling rtty-go client..."

    # Remove scheduled task
    try {
        Unregister-ScheduledTask -TaskName $TASK_NAME -Confirm:$false -ErrorAction SilentlyContinue
    } catch {}

    # Stop running process
    Get-Process -Name "rtty-go" -ErrorAction SilentlyContinue | Stop-Process -Force

    # Remove files
    if (Test-Path $INSTALL_DIR) {
        Remove-Item -Path $INSTALL_DIR -Recurse -Force
    }

    Log-Info "Uninstall complete."
    exit 0
}

# ========================== Main =============================================

# Check administrator privileges
if (-not (Test-Administrator)) {
    Log-Error "This script must be run as Administrator."
    Log-Error "Right-click PowerShell and select 'Run as Administrator', then retry."
    exit 1
}

# Handle uninstall
if ($Uninstall) {
    Invoke-Uninstall
}

# Show usage if missing required params
if (-not $RTTY_HOST -or -not $RTTY_TOKEN) {
    Write-Host @"
Usage: .\install-windows.ps1 -Host_Addr <host> -Token <token> [-Port <port>] [-DownloadBaseUrl <url>]

Options:
  -Host_Addr        Server host or IP address (required)
  -Token            Authorization token (required)
  -Port             Server port (default: 5912)
  -DownloadBaseUrl  Download base URL for binaries
  -Uninstall        Uninstall rtty-go client

Example:
  .\install-windows.ps1 -Host_Addr 107.173.152.173 -Token lHEP7GyyGt4S18KlyikfpvzdZTVxnD8v
"@
    exit 1
}

# Strip leading colon from port (e.g. ":5912" -> "5912")
$RTTY_PORT = $RTTY_PORT -replace '^:', ''

$PLATFORM   = "windows-amd64"

# Reuse existing device ID and MAC if config exists (preserve identity across reinstalls)
$EXISTING_ID  = ""
$EXISTING_MAC = ""
if (Test-Path $CONFIG_FILE) {
    Get-Content $CONFIG_FILE | ForEach-Object {
        if ($_ -match '^DEVICE_ID=(.+)$')  { $EXISTING_ID  = $Matches[1] }
        if ($_ -match '^DEVICE_MAC=(.+)$') { $EXISTING_MAC = $Matches[1] }
    }
}

$DEVICE_ID  = if ($EXISTING_ID)  { $EXISTING_ID }  else { New-DeviceId }
$DEVICE_MAC = if ($EXISTING_MAC) { $EXISTING_MAC } else { Get-PhysicalMac }

Log-Info "========================================"
Log-Info "  rtty-go Client Installer for Windows"
Log-Info "========================================"
Log-Info "Platform:    $PLATFORM"
Log-Info "Device ID:   $DEVICE_ID"
Log-Info "MAC address: $DEVICE_MAC"
Log-Info "Server:      ${RTTY_HOST}:${RTTY_PORT}"
Log-Info "========================================"

# Determine download URL
if (-not $DOWNLOAD_BASE_URL) {
    $DOWNLOAD_BASE_URL = "https://kvm-cloud.gl-inet.com/selfhost/clients"
}
$FILE_URL = "${DOWNLOAD_BASE_URL}/rtty-go-${PLATFORM}.exe"

# Create install directory
if (-not (Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
}

# Stop existing process and task before downloading (so the binary can be overwritten)
Get-Process -Name "rtty-go" -ErrorAction SilentlyContinue | Stop-Process -Force
try {
    Unregister-ScheduledTask -TaskName $TASK_NAME -Confirm:$false -ErrorAction SilentlyContinue
} catch {}

# Download binary
Log-Info "Downloading rtty-go-${PLATFORM}.exe..."
$TMP_FILE = "$env:TEMP\rtty-go.exe"

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $FILE_URL -OutFile $TMP_FILE -UseBasicParsing
} catch {
    Log-Error "Failed to download: $_"
    exit 1
}

# Install binary
Move-Item -Path $TMP_FILE -Destination "$INSTALL_DIR\$BINARY_NAME" -Force
Log-Info "Installed to $INSTALL_DIR\$BINARY_NAME"

# Save configuration
@"
RTTY_HOST=$RTTY_HOST
RTTY_PORT=$RTTY_PORT
RTTY_TOKEN=$RTTY_TOKEN
DEVICE_ID=$DEVICE_ID
DEVICE_MAC=$DEVICE_MAC
"@ | Out-File -FilePath $CONFIG_FILE -Encoding UTF8
Log-Info "Configuration saved to $CONFIG_FILE"

# Create a wrapper script for the scheduled task
$WRAPPER_SCRIPT = "$INSTALL_DIR\start-rtty-go.ps1"
@"
`$process = Get-Process -Name "rtty-go" -ErrorAction SilentlyContinue
if (-not `$process) {
    Start-Process -FilePath "$INSTALL_DIR\$BINARY_NAME" ``
        -ArgumentList "-s -x -a -I $DEVICE_ID -h $RTTY_HOST -p $RTTY_PORT -t $RTTY_TOKEN -d $DEVICE_MAC" ``
        -WindowStyle Hidden ``
        -RedirectStandardOutput "$LOG_FILE" ``
        -RedirectStandardError "$INSTALL_DIR\rtty-go-error.log"
}
"@ | Out-File -FilePath $WRAPPER_SCRIPT -Encoding UTF8

# Create scheduled task for auto-start
Log-Info "Setting up scheduled task for auto-start..."

$action = New-ScheduledTaskAction `
    -Execute "$INSTALL_DIR\$BINARY_NAME" `
    -Argument "-s -x -a -I $DEVICE_ID -h $RTTY_HOST -p $RTTY_PORT -t $RTTY_TOKEN -d $DEVICE_MAC"

$triggerBoot = New-ScheduledTaskTrigger -AtStartup
$triggerBoot.Delay = "PT30S"

$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -RestartCount 999 `
    -ExecutionTimeLimit (New-TimeSpan -Seconds 0)

$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask `
    -TaskName $TASK_NAME `
    -Action $action `
    -Trigger $triggerBoot `
    -Settings $settings `
    -Principal $principal `
    -Description "rtty-go remote terminal client" `
    -Force | Out-Null

# Start the task now
Start-ScheduledTask -TaskName $TASK_NAME
Start-Sleep -Seconds 2

# Verify
$proc = Get-Process -Name "rtty-go" -ErrorAction SilentlyContinue
if ($proc) {
    Log-Info "rtty-go is running (PID: $($proc.Id))"
} else {
    Log-Warn "rtty-go process not detected yet, it may take a moment to start"
}

Log-Info "========================================"
Log-Info "  Installation complete!"
Log-Info "  Device ID:   $DEVICE_ID"
Log-Info "  MAC address: $DEVICE_MAC"
Log-Info ""
Log-Info "  Manage service:"
Log-Info "    Stop:    Stop-ScheduledTask -TaskName $TASK_NAME"
Log-Info "    Start:   Start-ScheduledTask -TaskName $TASK_NAME"
Log-Info "    Status:  Get-ScheduledTask -TaskName $TASK_NAME"
Log-Info "    Log:     Get-Content $LOG_FILE -Tail 50"
Log-Info "    Remove:  .\install-windows.ps1 -Uninstall"
Log-Info "========================================"
