#!/bin/bash

# Script para testar o rate limiter
# Uso: ./test_rate_limit.sh [numero_de_requisições] [intervalo_em_segundos]

# Configurações
URL="http://localhost:8080/health"
NUM_REQUESTS=${1:-30}  # Padrão: 30 requisições
INTERVAL=${2:-0.1}     # Padrão: 0.1 segundos entre requisições

echo "========================================="
echo "Teste de Rate Limiter"
echo "========================================="
echo "URL: $URL"
echo "Número de requisições: $NUM_REQUESTS"
echo "Intervalo: ${INTERVAL}s"
echo "========================================="
echo ""

SUCCESS_COUNT=0
BLOCKED_COUNT=0

for i in $(seq 1 $NUM_REQUESTS); do
    # Fazer requisição e capturar status code
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" $URL)
    
    # Timestamp
    TIMESTAMP=$(date "+%H:%M:%S.%3N")
    
    # Verificar resultado
    if [ "$HTTP_CODE" -eq 200 ]; then
        echo "[$TIMESTAMP] Requisição #$i: ✅ 200 OK"
        ((SUCCESS_COUNT++))
    elif [ "$HTTP_CODE" -eq 429 ]; then
        echo "[$TIMESTAMP] Requisição #$i: ⛔ 429 Too Many Requests"
        ((BLOCKED_COUNT++))
    else
        echo "[$TIMESTAMP] Requisição #$i: ❓ $HTTP_CODE"
    fi
    
    # Aguardar antes da próxima requisição
    sleep $INTERVAL
done

echo ""
echo "========================================="
echo "Resumo:"
echo "========================================="
echo "Total de requisições: $NUM_REQUESTS"
echo "Sucesso (200): $SUCCESS_COUNT"
echo "Bloqueadas (429): $BLOCKED_COUNT"
if [ "$NUM_REQUESTS" -gt 0 ]; then
    BLOCK_RATE=$(echo "scale=2; ($BLOCKED_COUNT * 100) / $NUM_REQUESTS" | bc)
    echo "Taxa de bloqueio: ${BLOCK_RATE}%"
fi
echo "========================================="
