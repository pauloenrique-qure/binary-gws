# Guía de Instalación — Gateway Agent en Ubuntu (Linux x86_64)

Este documento explica cómo instalar, configurar y operar el **Gateway Agent** (`gw-agent`) en un servidor Ubuntu con arquitectura x86_64 (amd64).

El agente es un daemon ligero que se ejecuta en segundo plano y envía métricas del sistema (CPU, memoria, disco, temperatura, procesos) a un backend API central de forma periódica. No abre ningún puerto ni acepta conexiones entrantes.

---

## Tabla de contenidos

1. [Requisitos previos](#1-requisitos-previos)
2. [Obtener el binario](#2-obtener-el-binario)
3. [Instalación](#3-instalación)
4. [Configuración](#4-configuración)
5. [Probar antes de activar el servicio](#5-probar-antes-de-activar-el-servicio)
6. [Activar el servicio systemd](#6-activar-el-servicio-systemd)
7. [Verificar que el servicio está corriendo](#7-verificar-que-el-servicio-está-corriendo)
8. [Comandos de operación diaria](#8-comandos-de-operación-diaria)
9. [Solución de problemas comunes](#9-solución-de-problemas-comunes)
10. [Desinstalación](#10-desinstalación)

---

## 1. Requisitos previos

| Requisito | Detalle |
|-----------|---------|
| Sistema operativo | Ubuntu 20.04 LTS o superior |
| Arquitectura | x86_64 (amd64) |
| Acceso | Usuario con `sudo` o root |
| Red | Acceso HTTPS saliente al URL de la API backend |
| Credenciales | UUID, Client ID, Site ID y token de autenticación (entregados por el equipo técnico) |

No se requiere instalar ningún runtime (Go, Python, Java, etc.). El binario es completamente estático.

---

## 2. Obtener el binario

Copia el archivo `gw-agent` (binario Linux amd64) al servidor donde lo instalarás. Puedes hacerlo de varias formas:

**Opción A — Desde tu máquina local con `scp`:**
```bash
scp dist/linux_amd64/gw-agent usuario@IP_DEL_SERVIDOR:/tmp/gw-agent
```

**Opción B — Desde el servidor con `wget` o `curl`** (si el binario está en un servidor de descarga interno):
```bash
wget -O /tmp/gw-agent https://URL_DE_DESCARGA/gw-agent
```

Una vez copiado, conéctate al servidor por SSH y verifica que el archivo llegó correctamente:
```bash
ls -lh /tmp/gw-agent
```

---

## 3. Instalación

Todos los comandos de esta sección deben ejecutarse como **root** o con `sudo`.

### 3.1 Crear el usuario del sistema

El agente se ejecuta bajo un usuario sin privilegios (`gwagent`) por seguridad. Créalo si no existe:

```bash
sudo useradd --system --no-create-home --shell /bin/false gwagent
```

> Este usuario no tiene contraseña ni directorio home. Solo sirve para ejecutar el proceso.

### 3.2 Crear los directorios necesarios

```bash
sudo mkdir -p /opt/gw-agent
sudo mkdir -p /etc/gw-agent
sudo mkdir -p /var/log/gw-agent
```

| Directorio | Propósito |
|------------|-----------|
| `/opt/gw-agent` | Binario ejecutable |
| `/etc/gw-agent` | Archivo de configuración |
| `/var/log/gw-agent` | Directorio de logs (reservado) |

### 3.3 Instalar el binario

```bash
sudo cp /tmp/gw-agent /opt/gw-agent/gw-agent
sudo chmod 755 /opt/gw-agent/gw-agent
sudo chown root:root /opt/gw-agent/gw-agent
```

Verifica que el binario funciona:

```bash
/opt/gw-agent/gw-agent --print-version
```

Deberías ver algo como:
```
gw-agent version 1.2.3 (commit: abc1234, built: 2026-04-10)
```

### 3.4 Instalar el servicio systemd

Crea el archivo de servicio:

```bash
sudo tee /etc/systemd/system/gw-agent.service > /dev/null <<'EOF'
[Unit]
Description=Gateway Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=gwagent
Group=gwagent
ExecStart=/opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gw-agent

# Hardening de seguridad
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/gw-agent
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF
```

Recarga la configuración de systemd:

```bash
sudo systemctl daemon-reload
```

### 3.5 Ajustar permisos del directorio de logs

```bash
sudo chown gwagent:gwagent /var/log/gw-agent
```

---

## 4. Configuración

Crea el archivo de configuración en `/etc/gw-agent/config.yaml`:

```bash
sudo nano /etc/gw-agent/config.yaml
```

Pega el siguiente contenido y **reemplaza todos los valores marcados con `REEMPLAZAR`**:

```yaml
# ─────────────────────────────────────────────
# Identificadores del gateway (OBLIGATORIOS)
# ─────────────────────────────────────────────
uuid: "REEMPLAZAR_CON_UUID_DEL_GATEWAY"
client_id: "REEMPLAZAR_CON_NOMBRE_DEL_CLIENTE"
site_id: "REEMPLAZAR_CON_NOMBRE_DEL_SITIO"

# ─────────────────────────────────────────────
# API backend (OBLIGATORIO)
# ─────────────────────────────────────────────
api_url: "https://REEMPLAZAR_CON_URL_DE_LA_API/api/v1/service-stats-data/"

# URL de respaldo (opcional): se usa si la principal no responde
# api_url_fallbacks:
#   - "https://URL_ALTERNATIVA/api/v1/service-stats-data/"

# ─────────────────────────────────────────────
# Autenticación (OBLIGATORIO)
# ─────────────────────────────────────────────
auth:
  token_current: "REEMPLAZAR_CON_TOKEN_DE_AUTENTICACION"
  # token_grace: "TOKEN_ANTERIOR_DURANTE_ROTACION"  # Opcional

# ─────────────────────────────────────────────
# Intervalos de envío (opcionales)
# ─────────────────────────────────────────────
intervals:
  heartbeat_seconds: 15   # Cada cuántos segundos enviar heartbeat (default: 15)
  compute_seconds: 120    # Cada cuántos segundos refrescar métricas de CPU/mem/disco (default: 120)

# ─────────────────────────────────────────────
# Disco a monitorear (opcional pero recomendado)
# ─────────────────────────────────────────────
# En Ubuntu con múltiples particiones, el agente puede detectar /boot en lugar del
# disco de datos. Especifica aquí el punto de montaje correcto.
# disk_path: "/media/root/storage"

# ─────────────────────────────────────────────
# Procesos a monitorear individualmente (opcional)
# ─────────────────────────────────────────────
# monitored_processes:
#   - nginx
#   - python3

# ─────────────────────────────────────────────
# TLS (no modificar en producción)
# ─────────────────────────────────────────────
tls:
  insecure_skip_verify: false
```

### 4.1 Ajustar permisos del archivo de configuración

El archivo de configuración contiene tokens de autenticación. Restringe su lectura:

```bash
sudo chmod 600 /etc/gw-agent/config.yaml
sudo chown gwagent:gwagent /etc/gw-agent/config.yaml
```

Verifica los permisos:

```bash
ls -la /etc/gw-agent/config.yaml
# Debe mostrar: -rw------- 1 gwagent gwagent ...
```

### 4.2 Referencia de campos de configuración

| Campo | Obligatorio | Descripción |
|-------|:-----------:|-------------|
| `uuid` | Sí | Identificador único de este gateway (entregado por el equipo técnico) |
| `client_id` | Sí | Nombre del cliente u organización |
| `site_id` | Sí | Nombre del sitio o instalación física |
| `api_url` | Sí | URL completa del endpoint de heartbeat |
| `auth.token_current` | Sí | Token Bearer para autenticación en la API |
| `auth.token_grace` | No | Token anterior, usado durante rotaciones de token |
| `intervals.heartbeat_seconds` | No | Frecuencia de envío en segundos (default: 15) |
| `intervals.compute_seconds` | No | Frecuencia de refresco de métricas (default: 120) |
| `disk_path` | No | Punto de montaje del disco a monitorear |
| `monitored_processes` | No | Lista de procesos a monitorear individualmente |
| `tls.insecure_skip_verify` | No | Solo `true` en entornos de prueba. Default: `false` |

---

## 5. Probar antes de activar el servicio

Antes de habilitar el servicio en background, ejecuta el agente una sola vez para verificar que la configuración es correcta y que puede conectarse a la API:

```bash
sudo -u gwagent /opt/gw-agent/gw-agent \
  --config /etc/gw-agent/config.yaml \
  --once \
  --log-level debug
```

**Resultado esperado (éxito):**
```
{"timestamp":"...","level":"info","msg":"heartbeat sent successfully","status_code":200}
```

**Si ves un error de conexión:**
- Verifica que el servidor tiene acceso a internet o a la red del backend
- Confirma que `api_url` está escrito correctamente
- Revisa que el token en `auth.token_current` es válido

**Si ves error de configuración:**
- Verifica que todos los campos obligatorios (`uuid`, `client_id`, `site_id`, `api_url`, `auth.token_current`) están completos y sin comillas extra

También puedes hacer un **dry-run** (construye el payload JSON pero no lo envía) para inspeccionar qué métricas se recolectan:

```bash
sudo -u gwagent /opt/gw-agent/gw-agent \
  --config /etc/gw-agent/config.yaml \
  --dry-run
```

---

## 6. Activar el servicio systemd

Una vez confirmado que el agente funciona correctamente, habilítalo para que arranque automáticamente con el sistema:

```bash
# Habilitar inicio automático
sudo systemctl enable gw-agent

# Iniciar el servicio ahora
sudo systemctl start gw-agent
```

---

## 7. Verificar que el servicio está corriendo

```bash
sudo systemctl status gw-agent
```

Deberías ver algo similar a:

```
● gw-agent.service - Gateway Monitoring Agent
     Loaded: loaded (/etc/systemd/system/gw-agent.service; enabled; vendor preset: enabled)
     Active: active (running) since Thu 2026-04-16 10:00:00 UTC; 5s ago
   Main PID: 12345 (gw-agent)
      Tasks: 6
     Memory: 8.2M
        CPU: 0.012s
```

El campo clave es `Active: active (running)`.

Para ver los logs en tiempo real:

```bash
sudo journalctl -u gw-agent -f
```

Para ver los últimos 50 mensajes:

```bash
sudo journalctl -u gw-agent -n 50
```

---

## 8. Comandos de operación diaria

| Acción | Comando |
|--------|---------|
| Ver estado del servicio | `sudo systemctl status gw-agent` |
| Ver logs en tiempo real | `sudo journalctl -u gw-agent -f` |
| Ver logs recientes | `sudo journalctl -u gw-agent -n 100` |
| Reiniciar el servicio | `sudo systemctl restart gw-agent` |
| Detener el servicio | `sudo systemctl stop gw-agent` |
| Iniciar el servicio | `sudo systemctl start gw-agent` |
| Deshabilitar inicio automático | `sudo systemctl disable gw-agent` |
| Probar configuración (una vez) | `sudo -u gwagent /opt/gw-agent/gw-agent --config /etc/gw-agent/config.yaml --once` |
| Ver versión instalada | `/opt/gw-agent/gw-agent --print-version` |

### Actualizar el binario

Para reemplazar el binario por una versión más nueva:

```bash
# 1. Detener el servicio
sudo systemctl stop gw-agent

# 2. Copiar el nuevo binario
sudo cp /tmp/gw-agent-nuevo /opt/gw-agent/gw-agent
sudo chmod 755 /opt/gw-agent/gw-agent
sudo chown root:root /opt/gw-agent/gw-agent

# 3. Reiniciar el servicio
sudo systemctl start gw-agent

# 4. Verificar que arrancó correctamente
sudo systemctl status gw-agent
```

### Rotar el token de autenticación

Para cambiar el token sin interrumpir el envío de heartbeats:

1. Edita la configuración y agrega el token nuevo como `token_current` y el anterior como `token_grace`:

```bash
sudo nano /etc/gw-agent/config.yaml
```

```yaml
auth:
  token_current: "NUEVO_TOKEN"
  token_grace: "TOKEN_ANTERIOR"
```

2. Reinicia el servicio:

```bash
sudo systemctl restart gw-agent
```

3. Una vez confirmado que el nuevo token funciona, elimina `token_grace` del archivo y reinicia de nuevo.

---

## 9. Solución de problemas comunes

### El servicio no arranca

```bash
sudo journalctl -u gw-agent -n 50 --no-pager
```

Causas frecuentes:
- **Archivo de configuración no encontrado**: verifica que `/etc/gw-agent/config.yaml` existe
- **Permisos incorrectos**: el usuario `gwagent` debe poder leer el config (`chmod 600`, `chown gwagent`)
- **Campos obligatorios vacíos**: `uuid`, `client_id`, `site_id`, `api_url`, `token_current` no pueden estar vacíos

### El agente arranca pero no envía datos

1. Prueba conectividad al backend:
   ```bash
   curl -v https://URL_DE_LA_API/api/v1/service-stats-data/
   ```

2. Ejecuta en modo debug una sola vez:
   ```bash
   sudo -u gwagent /opt/gw-agent/gw-agent \
     --config /etc/gw-agent/config.yaml \
     --once --log-level debug
   ```

3. Verifica que el token no haya expirado (error HTTP 401 o 403 en los logs).

### Las métricas de disco muestran valores inesperados

El agente puede estar monitoreando `/boot` en lugar del disco principal. Identifica el punto de montaje correcto:

```bash
df -h
```

Busca la partición de datos (normalmente la de mayor tamaño). Luego agrega al config:

```yaml
disk_path: "/media/root/storage"   # Ajusta según tu sistema
```

Reinicia el servicio después de editar:

```bash
sudo systemctl restart gw-agent
```

### El servicio se reinicia en loop

El servicio está configurado para reiniciarse automáticamente (`Restart=always`). Si hay un error persistente, los logs mostrarán el motivo:

```bash
sudo journalctl -u gw-agent -n 100 --no-pager | grep -i error
```

---

## 10. Desinstalación

Para remover completamente el agente del sistema:

```bash
# 1. Detener y deshabilitar el servicio
sudo systemctl stop gw-agent
sudo systemctl disable gw-agent

# 2. Eliminar el archivo de servicio
sudo rm /etc/systemd/system/gw-agent.service
sudo systemctl daemon-reload

# 3. Eliminar binario, configuración y logs
sudo rm -rf /opt/gw-agent
sudo rm -rf /etc/gw-agent
sudo rm -rf /var/log/gw-agent

# 4. Eliminar el usuario del sistema
sudo userdel gwagent
```

> El paso 3 elimina la configuración permanentemente. Si quieres conservarla para una futura reinstalación, omite `sudo rm -rf /etc/gw-agent`.

---

## Resumen de rutas del sistema

| Ruta | Contenido |
|------|-----------|
| `/opt/gw-agent/gw-agent` | Binario ejecutable |
| `/etc/gw-agent/config.yaml` | Archivo de configuración |
| `/etc/systemd/system/gw-agent.service` | Definición del servicio systemd |
| `/var/log/gw-agent/` | Directorio de logs (los logs principales van a journald) |
