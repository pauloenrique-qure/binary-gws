#!/bin/bash
# Demo Script para Gateway Agent - Presentación para Jefe
# Este script ejecuta una demostración completa de las funcionalidades del agente

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
}

print_step() {
    echo -e "${GREEN}✓${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

pause_demo() {
    echo ""
    read -p "Presiona ENTER para continuar..."
}

# ============================================
# DEMO 1: Build y verificación de binarios
# ============================================
print_header "DEMO 1: Build de Binarios Multiplataforma"

print_info "Construyendo binarios para todas las plataformas..."
make clean > /dev/null 2>&1
make build-all

print_step "Binarios construidos exitosamente"
echo ""
echo "Verificando binarios:"
echo ""

for binary in dist/linux_amd64/gw-agent dist/linux_arm64/gw-agent dist/windows_amd64/gw-agent.exe; do
    if [ -f "$binary" ]; then
        size=$(ls -lh "$binary" | awk '{print $5}')
        file_type=$(file "$binary" | cut -d':' -f2)
        echo -e "${GREEN}✓${NC} $binary"
        echo "  Tamaño: $size"
        echo "  Tipo: $file_type"
        echo ""
    else
        print_error "No encontrado: $binary"
    fi
done

pause_demo

# ============================================
# DEMO 2: Información de versión
# ============================================
print_header "DEMO 2: Información de Versión"

print_info "Ejecutando: ./dist/gw-agent --print-version"
echo ""
./dist/gw-agent --print-version

pause_demo

# ============================================
# DEMO 3: Dry-run (sin red)
# ============================================
print_header "DEMO 3: Dry-run - Preview del Payload (SIN RED)"

print_info "Ejecutando: ./dist/gw-agent --config test-local-config.yaml --dry-run"
print_info "Esto muestra el payload JSON que se enviaría al backend"
echo ""

./dist/gw-agent --config test-local-config.yaml --dry-run --log-level info 2>&1 | grep -A 50 "payload" | tail -30

pause_demo

# ============================================
# DEMO 4: Test con servidor local
# ============================================
print_header "DEMO 4: Heartbeat End-to-End con Servidor Local"

print_info "Iniciando servidor de prueba en localhost:8080..."

# Iniciar servidor en background
go run ./cmd/test-server > /tmp/test-server.log 2>&1 &
SERVER_PID=$!
sleep 2

print_step "Servidor iniciado (PID: $SERVER_PID)"
echo ""

print_info "Enviando heartbeat al servidor local..."
echo ""

./dist/gw-agent --config test-local-config.yaml --once --log-level info

echo ""
print_step "Heartbeat enviado exitosamente"
echo ""

print_info "Logs del servidor:"
echo ""
cat /tmp/test-server.log | tail -20

# Limpiar
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

pause_demo

# ============================================
# DEMO 5: Validación de configuración
# ============================================
print_header "DEMO 5: Validación de Configuración"

print_info "Creando configuración inválida para demostrar validación..."

cat > /tmp/invalid-config.yaml <<EOF
uuid: "test"
# Falta client_id (campo requerido)
site_id: "test"
api_url: "http://localhost:8080/heartbeat"
EOF

echo ""
print_info "Intentando cargar configuración inválida..."
echo ""

if ./dist/gw-agent --config /tmp/invalid-config.yaml --dry-run 2>&1 | grep -q "client_id is required"; then
    print_step "Validación funcionando correctamente - detectó campo faltante"
else
    print_error "Validación no detectó el error"
fi

rm -f /tmp/invalid-config.yaml

pause_demo

# ============================================
# DEMO 6: Diferentes niveles de logging
# ============================================
print_header "DEMO 6: Niveles de Logging"

print_info "Ejecutando con nivel DEBUG para ver detalles internos..."
echo ""

./dist/gw-agent --config test-local-config.yaml --dry-run --log-level debug 2>&1 | head -5

pause_demo

# ============================================
# DEMO 7: Tests unitarios
# ============================================
print_header "DEMO 7: Ejecución de Tests Unitarios"

print_info "Ejecutando suite de tests con race detector..."
echo ""

if make test; then
    print_step "Todos los tests pasaron correctamente"
else
    print_error "Algunos tests fallaron"
fi

pause_demo

# ============================================
# RESUMEN FINAL
# ============================================
print_header "RESUMEN DE LA DEMOSTRACIÓN"

echo "✅ Funcionalidades demostradas:"
echo ""
echo "  1. Build multiplataforma (Linux amd64/arm64, Windows amd64)"
echo "  2. Binarios estáticos sin dependencias externas"
echo "  3. Información de versión embebida"
echo "  4. Dry-run - preview de payloads sin enviar"
echo "  5. Heartbeat end-to-end con servidor local"
echo "  6. Validación robusta de configuración"
echo "  7. Logging estructurado JSON con niveles configurables"
echo "  8. Tests unitarios con race detector"
echo ""
echo "📋 Características clave:"
echo ""
echo "  • Binarios: ~9MB estáticos, sin runtime dependencies"
echo "  • Métricas: CPU, memoria, disco (recolectadas localmente)"
echo "  • Seguridad: Bearer token auth, TLS configurable"
echo "  • Resiliencia: Reintentos automáticos con backoff"
echo "  • Multiplataforma: Linux, Windows, ARM64"
echo ""
print_step "Demostración completada exitosamente"
