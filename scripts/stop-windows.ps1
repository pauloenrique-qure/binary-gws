# Stop Gateway Agent service on Windows

$ServiceName = "GWAgent"

$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if (-not $service) {
    Write-Host "Error: Service '$ServiceName' not found" -ForegroundColor Red
    exit 1
}

if ($service.Status -eq 'Stopped') {
    Write-Host "Service is already stopped" -ForegroundColor Green
    exit 0
}

Write-Host "Stopping Gateway Agent service..." -ForegroundColor Cyan
Stop-Service -Name $ServiceName -Force

Start-Sleep -Seconds 2
$service.Refresh()

if ($service.Status -eq 'Stopped') {
    Write-Host "Service stopped successfully!" -ForegroundColor Green
} else {
    Write-Host "Warning: Service may still be running" -ForegroundColor Yellow
    Write-Host "Status: $($service.Status)"
}

Get-Service -Name $ServiceName
