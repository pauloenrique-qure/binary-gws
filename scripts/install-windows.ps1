# Gateway Agent - Windows Installation Script
# Run as Administrator

param(
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"

$InstallDir = "C:\Program Files\GWAgent"
$ConfigDir = "C:\ProgramData\GWAgent"
$LogDir = "C:\ProgramData\GWAgent\logs"
$BinaryName = "gw-agent.exe"
$ServiceName = "GWAgent"
$ServiceDisplayName = "Gateway Monitoring Agent"

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Administrator)) {
    Write-Host "Error: This script must be run as Administrator" -ForegroundColor Red
    exit 1
}

if ($Uninstall) {
    Write-Host "Uninstalling Gateway Agent..." -ForegroundColor Yellow

    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq 'Running') {
            Write-Host "Stopping service..."
            Stop-Service -Name $ServiceName -Force
        }
        Write-Host "Removing service..."
        sc.exe delete $ServiceName
    }

    if (Test-Path $InstallDir) {
        Write-Host "Removing installation directory..."
        Remove-Item -Path $InstallDir -Recurse -Force
    }

    Write-Host "Uninstallation complete!" -ForegroundColor Green
    Write-Host "Configuration and logs at $ConfigDir were preserved"
    exit 0
}

Write-Host "Gateway Agent - Windows Installation" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

$BinaryPath = ".\dist\windows_amd64\$BinaryName"

if (-not (Test-Path $BinaryPath)) {
    Write-Host "Error: Binary not found at $BinaryPath" -ForegroundColor Red
    Write-Host "Please run 'make build-all' first" -ForegroundColor Red
    exit 1
}

Write-Host "Creating directories..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

Write-Host "Installing binary..."
Copy-Item -Path $BinaryPath -Destination "$InstallDir\$BinaryName" -Force

$ConfigPath = "$ConfigDir\config.yaml"
if (-not (Test-Path $ConfigPath)) {
    Write-Host "Creating sample configuration..."

    $configContent = @"
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
#   platform_override: "windows"

intervals:
  heartbeat_seconds: 60
  compute_seconds: 120

tls:
  # ca_bundle_path: "C:\\path\\to\\ca-bundle.pem"
  insecure_skip_verify: false
"@

    Set-Content -Path $ConfigPath -Value $configContent -Encoding UTF8
    Write-Host "Sample configuration created at $ConfigPath" -ForegroundColor Green
    Write-Host "WARNING: You MUST edit this file and set your actual values!" -ForegroundColor Yellow
} else {
    Write-Host "Configuration file already exists at $ConfigPath"
}

$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($service) {
    Write-Host "Service already exists, removing old version..."
    if ($service.Status -eq 'Running') {
        Stop-Service -Name $ServiceName -Force
    }
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

Write-Host "Installing Windows service..."
$binaryFullPath = "$InstallDir\$BinaryName"
$serviceArgs = "--config `"$ConfigPath`""

sc.exe create $ServiceName binPath= "`"$binaryFullPath`" $serviceArgs" start= auto DisplayName= $ServiceDisplayName

if ($LASTEXITCODE -eq 0) {
    Write-Host "Service installed successfully!" -ForegroundColor Green

    sc.exe description $ServiceName "Monitors gateway status and sends periodic heartbeats to the backend API"
    sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000
} else {
    Write-Host "Error: Failed to install service" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:"
Write-Host "1. Edit the configuration file: $ConfigPath"
Write-Host "2. Start the service: Start-Service -Name $ServiceName"
Write-Host "3. Check status: Get-Service -Name $ServiceName"
Write-Host "4. View logs: Get-Content $LogDir\gw-agent.log -Tail 50 -Wait"
Write-Host ""
Write-Host "To test manually before starting service:"
Write-Host "  & '$binaryFullPath' --config '$ConfigPath' --once"
Write-Host ""
Write-Host "To uninstall: .\install-windows.ps1 -Uninstall"
