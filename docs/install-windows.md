# Windows 10/11 Installation Guide

## Prerequisites

- PowerShell open as **Administrator**
- `gw-agent.exe` binary compiled (see build section below)

---

## 1. Build the binary (from Linux/macOS)

```bash
make build-all
```

Produces `dist/windows_amd64/gw-agent.exe`.

Transfer it to the Windows machine via USB, network share, SCP, etc.

---

## 2. Create directories

```powershell
New-Item -ItemType Directory -Force -Path "C:\Program Files\GWAgent"
New-Item -ItemType Directory -Force -Path "C:\ProgramData\GWAgent\logs"
```

---

## 3. Copy the binary

```powershell
Copy-Item -Path "C:\Users\<USERNAME>\Desktop\gw-agent.exe" -Destination "C:\Program Files\GWAgent\gw-agent.exe"
```

Verify:

```powershell
Test-Path "C:\Program Files\GWAgent\gw-agent.exe"
# Should return: True
```

> **Note:** Always run `Test-Path` to confirm the copy succeeded before continuing.
> If it returns `False`, create the directory first (step 2) and retry.

---

## 4. Get the machine UUID

If no UUID has been assigned, use the Windows `MachineGuid`:

```powershell
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Cryptography" -Name MachineGuid | Select-Object -ExpandProperty MachineGuid
```

Copy the output carefully — it will be used in the next step.

---

## 5. Create the configuration

```powershell
@"
uuid: "REPLACE_UUID"
client_id: "REPLACE_CLIENT_ID"
site_id: "REPLACE_SITE_ID"

api_url: "https://pulseapi.qure.ai/api/v1/service-stats-data/"

auth:
  token_current: "REPLACE_TOKEN"

intervals:
  heartbeat_seconds: 15
  compute_seconds: 120

monitored_processes:
  - DCMIO
  - dcmio_backend
  - postgres
  - Orthanc
  - scheduler

tls:
  insecure_skip_verify: false
"@ | Set-Content -Path "C:\ProgramData\GWAgent\config.yaml" -Encoding UTF8
```

> **Warning:** When pasting multi-line content in PowerShell, characters can be silently dropped.
> Always verify the UUID after creating the file:
>
> ```powershell
> Select-String "uuid" "C:\ProgramData\GWAgent\config.yaml"
> ```

> If the site runs DCMIO with a native PostgreSQL (no Docker), add the `dcmio` block:
>
> ```yaml
> dcmio:
>   postgres_url: "postgres://USER:PASS@127.0.0.1:PORT/DB?sslmode=disable"
>   interval_seconds: 60
>   window_hours: 12
> ```
>
> Note: if the password contains `@`, encode it as `%40` in the URL.

---

## 6. Find DCMIO PostgreSQL credentials (if applicable)

DCMIO runs PostgreSQL on a **non-standard port** (not 5432). Find the actual port:

```powershell
Get-NetTCPConnection -State Listen | Where-Object { $_.OwningProcess -in (Get-Process postgres).Id } | Select-Object LocalPort, LocalAddress
```

Find credentials and database name in:

```
C:\Users\<USERNAME>\AppData\Roaming\DCMIO\runtimeConfig.json
```

The relevant fields are `db.user`, `db.password`, `db.name`, and `db.port`.

> **Note:** The postgres user is typically `qure`, not `postgres`. The database name may also
> differ from `dcmio` — always check `runtimeConfig.json`.

Verify the connection before writing the config:

```powershell
& "C:\Program Files\DCMIO\resources\assets\pgsql\bin\psql.exe" -U <USER> -d <DB> -p <PORT> -c "SELECT 1;"
```

> **Note:** `psql` is not in the system PATH. It ships with DCMIO at:
> `C:\Program Files\DCMIO\resources\assets\pgsql\bin\psql.exe`

---

## 7. Test before installing the service

```powershell
& 'C:\Program Files\GWAgent\gw-agent.exe' --config 'C:\ProgramData\GWAgent\config.yaml' --once
```

Expected output:

```
"msg":"Heartbeat sent successfully"
"msg":"Single heartbeat completed"
```

Do not proceed to service installation until this test passes.

---

## 8. Install the Windows service

> **Important:** Use `New-Service`, not `sc.exe create`. The `sc.exe create` command fails
> silently with paths that contain spaces (e.g. `C:\Program Files\...`).

```powershell
New-Service -Name "GWAgent" -DisplayName "Gateway Monitoring Agent" -BinaryPathName '"C:\Program Files\GWAgent\gw-agent.exe" --config "C:\ProgramData\GWAgent\config.yaml"' -StartupType Automatic

sc.exe description GWAgent "Monitors gateway status and sends periodic heartbeats to the backend API"

sc.exe failure GWAgent reset= 86400 actions= restart/60000/restart/60000/restart/60000
```

---

## 9. Start the service

```powershell
Start-Service -Name GWAgent
Get-Service -Name GWAgent
# Status should be: Running
```

---

## Service management commands

```powershell
# Check status
Get-Service -Name GWAgent

# Restart (after config changes)
Restart-Service -Name GWAgent

# Stop
Stop-Service -Name GWAgent

# Uninstall
Stop-Service -Name GWAgent -Force
sc.exe delete GWAgent
```

---

## Full uninstallation

```powershell
Stop-Service -Name GWAgent -Force
sc.exe delete GWAgent
Remove-Item -Path "C:\Program Files\GWAgent" -Recurse -Force
# Optional: remove config and logs
# Remove-Item -Path "C:\ProgramData\GWAgent" -Recurse -Force
```

---

## Typical monitored processes on Windows

| Process | Description |
|---|---|
| `DCMIO` | DCMIO main application |
| `dcmio_backend` | DCMIO backend |
| `postgres` | PostgreSQL database |
| `Orthanc` | DICOM server |
| `scheduler` | DCMIO task scheduler |

> Names must match exactly what `Get-Process` shows.
> To verify: `Get-Process | Where-Object { $_.Name -like "*dcmio*" }`

---

## Troubleshooting

### `Test-Path` returns `False` after copying the binary
The directory does not exist. Run step 2 first, then retry the copy.

### `HTTP 400: Must be a valid UUID`
The UUID in `config.yaml` is malformed — likely a character was dropped when pasting.
Check and fix it:
```powershell
Select-String "uuid" "C:\ProgramData\GWAgent\config.yaml"
# Correct format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12 hex chars)
```

### PowerShell pipe error: "An empty pipe element is not allowed"
This happens when a command is pasted across multiple lines with `|` at the start of a new line.
Always paste piped commands as a single line.

### `sc.exe create` prints usage instead of creating the service
Use `New-Service` instead (see step 8). `sc.exe create` does not handle quoted paths with spaces reliably.

### `Start-Service` hangs indefinitely
The binary does not implement the Windows Service Control Manager protocol.
Make sure you are using a binary compiled after the `service_windows.go` change was introduced.
Kill the stuck process and reinstall with the correct binary:
```powershell
taskkill /F /IM gw-agent.exe
sc.exe delete GWAgent
```

### `role "postgres" does not exist` when connecting to DCMIO database
DCMIO does not use the `postgres` superuser. Check the actual credentials in:
```
C:\Users\<USERNAME>\AppData\Roaming\DCMIO\runtimeConfig.json
```

### `connection refused` on port 5432
DCMIO PostgreSQL runs on a non-standard port. Find the actual port:
```powershell
Get-NetTCPConnection -State Listen | Where-Object { $_.OwningProcess -in (Get-Process postgres).Id } | Select-Object LocalPort, LocalAddress
```

### Process list shows total count but no individual processes
The process names in `monitored_processes` do not match exactly.
Verify the exact names with:
```powershell
Get-Process | Where-Object { $_.Name -like "*dcmio*" -or $_.Name -like "*postgres*" } | Select-Object Name
```
