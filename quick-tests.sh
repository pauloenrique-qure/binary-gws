#!/bin/bash
# Quick Tests - Pruebas individuales rápidas

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

show_menu() {
    clear
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}  Gateway Agent - Quick Tests  ${NC}"
    echo -e "${CYAN}================================${NC}"
    echo ""
    echo "1. Build binarios (todas las plataformas)"
    echo "2. Verificar binarios con 'file' command"
    echo "3. Print version"
    echo "4. Dry-run (ver payload sin enviar)"
    echo "5. Test con servidor local (2 terminales necesarias)"
    echo "6. Test de retry logic (servidor con fallos)"
    echo "7. Test de validación de config"
    echo "8. Ejecutar tests unitarios"
    echo "9. Demo Docker (levantar stack completo)"
    echo "10. Limpiar builds"
    echo ""
    echo "0. Salir"
    echo ""
    read -p "Selecciona opción: " option
    echo ""
    return $option
}

test_build() {
    echo -e "${BLUE}Building binarios...${NC}"
    make build-all
    echo ""
    echo -e "${GREEN}✓ Binarios construidos exitosamente${NC}"
    echo ""
    ls -lh dist/*/gw-agent*
}

test_verify() {
    echo -e "${BLUE}Verificando formato de binarios...${NC}"
    echo ""
    for binary in dist/linux_amd64/gw-agent dist/linux_arm64/gw-agent dist/windows_amd64/gw-agent.exe; do
        if [ -f "$binary" ]; then
            echo -e "${GREEN}$binary:${NC}"
            file "$binary"
            echo ""
        fi
    done
}

test_version() {
    echo -e "${BLUE}Versión del agente:${NC}"
    echo ""
    ./dist/gw-agent --print-version
}

test_dryrun() {
    echo -e "${BLUE}Ejecutando dry-run (payload preview):${NC}"
    echo ""
    ./dist/gw-agent --config test-local-config.yaml --dry-run
}

test_server() {
    echo -e "${BLUE}Test con servidor local${NC}"
    echo ""
    echo "INSTRUCCIONES:"
    echo "1. Abre una segunda terminal"
    echo "2. Ejecuta: go run test-server.go"
    echo "3. Vuelve aquí y presiona ENTER"
    echo ""
    read -p "¿Servidor listo? (ENTER para continuar) "
    echo ""
    echo -e "${BLUE}Enviando heartbeat...${NC}"
    ./dist/gw-agent --config test-local-config.yaml --once --log-level info
    echo ""
    echo -e "${GREEN}✓ Heartbeat enviado. Revisa la otra terminal.${NC}"
}

test_retry() {
    echo -e "${BLUE}Test de retry logic${NC}"
    echo ""
    echo "Este test levantará un servidor que falla 2 veces antes de responder OK."
    echo ""

    # Levantar servidor en background
    go run test-retry-server.go > /tmp/retry-server.log 2>&1 &
    SERVER_PID=$!
    sleep 2

    echo -e "${BLUE}Servidor iniciado (PID: $SERVER_PID)${NC}"
    echo -e "${BLUE}Enviando heartbeat (esperará ~20s con reintentos)...${NC}"
    echo ""

    ./dist/gw-agent --config test-local-config.yaml --once --log-level debug

    echo ""
    echo -e "${GREEN}✓ Test completado${NC}"
    echo ""
    echo "Logs del servidor:"
    cat /tmp/retry-server.log

    # Limpiar
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
}

test_validation() {
    echo -e "${BLUE}Test de validación de configuración${NC}"
    echo ""

    echo "Test 1: Campo requerido faltante"
    cat > /tmp/invalid-config.yaml <<EOF
uuid: "test"
# client_id faltante
site_id: "test"
api_url: "http://localhost:8080/heartbeat"
EOF

    if ./dist/gw-agent --config /tmp/invalid-config.yaml --dry-run 2>&1 | grep -q "client_id is required"; then
        echo -e "${GREEN}✓ Detectó campo faltante correctamente${NC}"
    else
        echo -e "✗ No detectó el error"
    fi
    echo ""

    echo "Test 2: URL inválida"
    cat > /tmp/invalid-config2.yaml <<EOF
uuid: "test"
client_id: "test"
site_id: "test"
api_url: "not-a-url"
auth:
  token_current: "test"
EOF

    if ./dist/gw-agent --config /tmp/invalid-config2.yaml --dry-run 2>&1 | grep -q "HTTP"; then
        echo -e "${GREEN}✓ Detectó URL inválida correctamente${NC}"
    else
        echo -e "✗ No detectó el error"
    fi

    rm -f /tmp/invalid-config*.yaml
}

test_unit() {
    echo -e "${BLUE}Ejecutando tests unitarios...${NC}"
    echo ""
    make test
}

test_docker() {
    echo -e "${BLUE}Demo con Docker Compose${NC}"
    echo ""
    echo "Levantando servidor + agente en contenedores..."
    echo "Presiona Ctrl+C para detener"
    echo ""
    docker-compose -f docker-compose.demo.yml up --build
}

test_clean() {
    echo -e "${BLUE}Limpiando builds...${NC}"
    make clean
    echo -e "${GREEN}✓ Build limpiado${NC}"
}

# Main loop
while true; do
    show_menu
    option=$?

    case $option in
        1) test_build ;;
        2) test_verify ;;
        3) test_version ;;
        4) test_dryrun ;;
        5) test_server ;;
        6) test_retry ;;
        7) test_validation ;;
        8) test_unit ;;
        9) test_docker ;;
        10) test_clean ;;
        0) echo "Saliendo..."; exit 0 ;;
        *) echo "Opción inválida" ;;
    esac

    echo ""
    read -p "Presiona ENTER para volver al menú..."
done
