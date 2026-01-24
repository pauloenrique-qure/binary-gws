# Gateway Monitoring Agent - Resumen Ejecutivo

## Descripción General

**Gateway Monitoring Agent** es un daemon de monitoreo de sistemas escrito en Go que envía heartbeats periódicos a un backend API, incluyendo métricas de sistema en tiempo real.

## Características Principales

### 1. **Multiplataforma**
- ✅ Linux x86_64 (Ubuntu, servidores)
- ✅ Linux ARM64 (Raspberry Pi)
- ✅ Windows x64
- Binarios estáticos de ~9MB sin dependencias externas

### 2. **Métricas del Sistema**
- CPU: Porcentaje de uso
- Memoria: Total, usado, porcentaje
- Disco: Total, usado, porcentaje
- Actualización configurable (default: cada 120s)

### 3. **Comunicación Segura**
- **Push-only**: Sin puertos entrantes, solo comunicación saliente HTTPS
- **Autenticación**: Bearer tokens
- **TLS**: Verificación de certificados + soporte para CA bundles personalizados
- **Rotación de tokens**: Zero-downtime con dual-token support

### 4. **Resiliencia**
- Reintentos automáticos en fallos de red (3x con backoff: 5s, 15s, 30s)
- Fallback a token de gracia en errores de autenticación
- Caché de métricas para reducir overhead
- Graceful shutdown en señales del sistema

### 5. **Operación como Servicio**
- **Linux**: systemd service con usuario dedicado no-privilegiado
- **Windows**: Windows Service registrado
- Scripts de instalación automatizados
- Logging estructurado JSON con niveles configurables

## Arquitectura Técnica

```
┌─────────────────────────────────────────────────┐
│              Gateway Agent (Go)                 │
├─────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐            │
│  │ Scheduler   │  │  Collector   │            │
│  │ (Heartbeat) │──│  (Métricas)  │            │
│  └──────┬──────┘  └──────────────┘            │
│         │                                       │
│  ┌──────▼───────┐  ┌──────────────┐           │
│  │  Transport   │  │   Logger     │           │
│  │ (HTTP+Retry) │  │   (JSON)     │           │
│  └──────┬───────┘  └──────────────┘           │
└─────────┼─────────────────────────────────────┘
          │ HTTPS POST (Bearer Token)
          ▼
  ┌───────────────────┐
  │   Backend API     │
  │ (Recibe Heartbeat)│
  └───────────────────┘
```

### Módulos Internos
- **config**: Carga y validación de YAML
- **platform**: Detección de OS/arquitectura/virtualización
- **collector**: Recolección de métricas con caché thread-safe
- **scheduler**: Orquestación de heartbeats con timer
- **transport**: Cliente HTTP con retry logic y token fallback
- **logging**: JSON estructurado con redacción de información sensible

## Formato de Payload (v1.0)

```json
{
  "payload_version": "1.0",
  "uuid": "gateway-123",
  "client_id": "cliente",
  "site_id": "sitio",
  "stats": {
    "system_status": "online",
    "compute": {
      "cpu": {"usage_percent": 25.5},
      "memory": {
        "total_bytes": 17179869184,
        "used_bytes": 12552273920,
        "usage_percent": 73.06
      },
      "disk": {
        "total_bytes": 494384795648,
        "used_bytes": 100244705280,
        "usage_percent": 20.27
      }
    }
  },
  "additional_notes": {
    "metadata": {
      "platform": "ubuntu",
      "agent_version": "1.0.0",
      "build": "abc123 2024-01-01"
    }
  },
  "agent_timestamp_utc": "2026-01-24T15:00:00Z"
}
```

**Nota**: Métricas no disponibles se omiten completamente (no se envía null/0/"unknown").

## Pruebas Realizadas

### ✅ Validación Local (Offline)
1. **Build multiplataforma**: 3 binarios verificados con `file` command
2. **Dry-run**: Preview de payload sin red
3. **Servidor local**: Heartbeats end-to-end en localhost:8080
4. **Retry logic**: Servidor simulando fallos HTTP 500
5. **Tests unitarios**: Suite completa pasando con race detector

### ✅ Funcionalidades Verificadas
- Construcción de payload conforme a especificación v1.0
- Recolección de métricas reales del sistema
- Detección de plataforma
- Validación robusta de configuración
- Logging estructurado con redacción de tokens

### ⏳ Pendiente de Validación en Dispositivos Reales
- Raspberry Pi (ARM64)
- Ubuntu Server (x86_64)
- Windows PC (x64)
- Instalación como servicio en cada plataforma
- Pruebas de larga duración (24h+)

## Demostración Disponible

### Demo Automatizada (2-3 minutos)
```bash
./demo.sh
```

Incluye:
1. Build de binarios
2. Verificación con `file` command
3. Información de versión
4. Dry-run de payload
5. Heartbeat con servidor local
6. Validación de configuración
7. Tests unitarios

### Demo con Docker (Opcional, 5 minutos)
```bash
docker-compose -f docker-compose.demo.yml up
```

Levanta servidor + agente en red aislada.

## Seguridad

### Medidas Implementadas
- ✅ Bearer token authentication con rotación
- ✅ TLS verification habilitada por defecto
- ✅ Tokens NUNCA logueados
- ✅ UUIDs redactados en logs (solo 4 primeros + 4 últimos chars)
- ✅ Servicio corre con usuario no-privilegiado (Linux)
- ✅ Configuración con permisos restrictivos (0600)
- ✅ Sin auto-update (previene supply-chain attacks)

### Consideraciones
- ⚠️ Config files contienen tokens en texto plano (mitigado con permisos)
- ⚠️ `insecure_skip_verify` solo para desarrollo/testing

## Métricas del Proyecto

| Métrica | Valor |
|---------|-------|
| Lenguaje | Go 1.25.6 |
| Líneas de código | ~1,100 (sin tests) |
| Módulos internos | 6 |
| Tests unitarios | 3 archivos |
| Cobertura de tests | Paths críticos cubiertos |
| Tamaño binario | ~9MB (estático) |
| Dependencias directas | 2 (gopsutil, yaml) |
| Dependencias totales | 16 (incluyendo indirectas) |
| Documentación | 1,500+ líneas |

## Calidad del Código

### Fortalezas
- ✅ Separación de responsabilidades (6 módulos independientes)
- ✅ Error handling robusto con wrapping
- ✅ Thread-safety con RWMutex
- ✅ Context propagation correcta
- ✅ Interfaces inyectables para testing
- ✅ Graceful shutdown
- ✅ Logging estructurado JSON
- ✅ Código idiomático Go

### Patrones Aplicados
- Repository pattern (collector como repositorio de métricas)
- Strategy pattern (Sleeper interface para testing)
- Builder pattern (construcción de payloads)
- Circuit breaker (retry con límites)

## Roadmap de Deployment

### Fase 1: Validación Local ✅
- [x] Build de binarios
- [x] Tests offline
- [x] Servidor local

### Fase 2: Validación en Dispositivos (Pendiente)
- [ ] Raspberry Pi: Verificar binario ARM64
- [ ] Ubuntu Server: Instalar como servicio systemd
- [ ] Windows PC: Instalar como servicio Windows
- [ ] Verificar detección de plataforma

### Fase 3: Integración con Backend (Pendiente)
- [ ] Configurar endpoint real
- [ ] Configurar tokens de producción
- [ ] Validar TLS con certificados reales
- [ ] Probar rotación de tokens

### Fase 4: Producción (Pendiente)
- [ ] Deployment en todos los gateways
- [ ] Monitoreo de logs
- [ ] Pruebas de larga duración
- [ ] Documentación operativa

## Costos de Operación

### Recursos por Gateway
- **CPU**: ~0.1% (idle), ~1% (durante colección de métricas)
- **Memoria**: ~10-15MB RSS
- **Disco**: 9MB binario + config (~1KB) + logs (rotables)
- **Red**: ~1KB cada 60s (heartbeat)

### Escalabilidad
- 1,000 gateways = ~1,000 heartbeats/minuto = ~60KB/minuto de tráfico
- Backend debe soportar 16-17 req/s (1000/60)

## Conclusiones

### Estado Actual
**LISTO PARA VALIDACIÓN EN DISPOSITIVOS REALES**

El código está:
- ✅ Bien arquitecturado y modular
- ✅ Robusto con manejo de errores
- ✅ Seguro (auth, TLS, logging)
- ✅ Probado (tests unitarios)
- ✅ Documentado exhaustivamente
- ✅ Validado offline localmente

### Recomendaciones
1. **Proceder con Fase 2**: Validar en Raspberry Pi, Ubuntu, Windows
2. **Documentar resultados**: Agregar evidencia a PROOF.md
3. **Preparar backend**: Asegurar endpoint listo para recibir heartbeats
4. **Planificar deployment**: Definir estrategia de rollout gradual

### Riesgos Identificados
- **Bajo**: Funcionalidad core está probada
- **Medio**: Detección de plataforma en dispositivos reales (mitigado con override manual)
- **Bajo**: Permisos para métricas (mitigado con omisión graceful)

## Aprobación para Continuar

```
□ Aprobar arquitectura y diseño
□ Aprobar funcionalidad demostrada localmente
□ Autorizar pruebas en dispositivos reales
□ Asignar dispositivos de prueba (RPI, Ubuntu, Windows)
□ Coordinar con equipo de backend para endpoint
```

---

**Preparado por**: [Tu nombre]
**Fecha**: 2026-01-24
**Versión del agente**: dev (pre-release)
