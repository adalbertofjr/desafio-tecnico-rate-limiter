#!/bin/bash

# Script para testar rate limiter com múltiplos IPs simulados
# Uso: ./test_multiple_ips.sh [total_requisições] [número_de_ips] [intervalo]

# Configurações
URL="http://localhost:8080/health"
TOTAL_REQUESTS=${1:-100}     # Total de requisições
NUM_IPS=${2:-5}               # Número de IPs diferentes a simular
INTERVAL=${3:-0.05}           # Intervalo entre requisições

echo "========================================="
echo "Teste de Rate Limiter - Múltiplos IPs"
echo "========================================="
echo "URL: $URL"
echo "Total de requisições: $TOTAL_REQUESTS"
echo "Número de IPs simulados: $NUM_IPS"
echo "Intervalo: ${INTERVAL}s"
echo "Requisições por IP: $((TOTAL_REQUESTS / NUM_IPS))"
echo "========================================="
echo ""

# Arrays para contadores por IP
declare -A success_count
declare -A blocked_count

# Inicializar contadores
for i in $(seq 1 $NUM_IPS); do
    success_count[$i]=0
    blocked_count[$i]=0
done

TOTAL_SUCCESS=0
TOTAL_BLOCKED=0

# Gerar IPs de forma distribuída
for req in $(seq 1 $TOTAL_REQUESTS); do
    # Distribuir requisições entre IPs de forma circular
    IP_INDEX=$(( (req - 1) % NUM_IPS + 1 ))
    FAKE_IP="192.168.1.$IP_INDEX"
    
    # Fazer requisição com header X-Forwarded-For
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "X-Forwarded-For: $FAKE_IP" \
        $URL)
    
    TIMESTAMP=$(date "+%H:%M:%S")
    
    if [ "$HTTP_CODE" -eq 200 ]; then
        echo "[$TIMESTAMP] IP: $FAKE_IP - Req #$req: ✅ 200 OK"
        ((success_count[$IP_INDEX]++))
        ((TOTAL_SUCCESS++))
    elif [ "$HTTP_CODE" -eq 429 ]; then
        echo "[$TIMESTAMP] IP: $FAKE_IP - Req #$req: ⛔ 429 Blocked"
        ((blocked_count[$IP_INDEX]++))
        ((TOTAL_BLOCKED++))
    else
        echo "[$TIMESTAMP] IP: $FAKE_IP - Req #$req: ❓ $HTTP_CODE"
    fi
    
    sleep $INTERVAL
done

echo ""
echo "========================================="
echo "Resumo por IP:"
echo "========================================="
for i in $(seq 1 $NUM_IPS); do
    IP="192.168.1.$i"
    TOTAL_IP=$((success_count[$i] + blocked_count[$i]))
    if [ "$TOTAL_IP" -gt 0 ]; then
        BLOCK_RATE=$(echo "scale=1; (${blocked_count[$i]} * 100) / $TOTAL_IP" | bc)
    else
        BLOCK_RATE=0
    fi
    echo "IP $IP: ${success_count[$i]} OK | ${blocked_count[$i]} Blocked | ${BLOCK_RATE}% bloqueio"
done

echo ""
echo "========================================="
echo "Resumo Geral:"
echo "========================================="
echo "Total de requisições: $TOTAL_REQUESTS"
echo "Sucesso (200): $TOTAL_SUCCESS"
echo "Bloqueadas (429): $TOTAL_BLOCKED"
if [ "$TOTAL_REQUESTS" -gt 0 ]; then
    BLOCK_RATE=$(echo "scale=2; ($TOTAL_BLOCKED * 100) / $TOTAL_REQUESTS" | bc)
    echo "Taxa de bloqueio global: ${BLOCK_RATE}%"
fi
echo "========================================="
