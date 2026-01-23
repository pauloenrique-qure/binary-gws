# Start Gateway Agent service on Windows

$ServiceName = "GWAgent"

$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if (-not $service) {
    Write-Host "Error: Service '$ServiceName' not found" -ForegroundColor Red
    Write-Host "Please run install-windows.ps1 first" -ForegroundColor Yellow
    exit 1
}

if ($service.Status -eq 'Running') {
    Write-Host "Service is already running" -ForegroundColor Green
    exit 0
}

Write-Host "Starting Gateway Agent service..." -ForegroundColor Cyan
Start-Service -Name $ServiceName

Start-Sleep -Seconds 2
$service.Refresh()

if ($service.Status -eq 'Running') {
    Write-Host "Service started successfully!" -ForegroundColor Green
} else {
    Write-Host "Warning: Service may not have started properly" -ForegroundColor Yellow
    Write-Host "Status: $($service.Status)"
}

Get-Service -Name $ServiceName
