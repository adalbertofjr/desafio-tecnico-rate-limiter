#!/bin/bash

# Script de stress test para rate limiter
# Envia requisições concorrentes de múltiplos processos
# Uso: ./stress_test.sh [requisições_por_processo] [número_de_processos]

URL="http://localhost:8080/health"
REQUESTS_PER_PROC=${1:-20}
NUM_PROCESSES=${2:-10}
TOTAL_REQUESTS=$((REQUESTS_PER_PROC * NUM_PROCESSES))

echo "========================================="
echo "Stress Test - Rate Limiter"
echo "========================================="
echo "URL: $URL"
echo "Processos concorrentes: $NUM_PROCESSES"
echo "Requisições por processo: $REQUESTS_PER_PROC"
echo "Total de requisições: $TOTAL_REQUESTS"
echo "========================================="
echo ""
echo "Iniciando teste..."

# Arquivo temporário para resultados
TEMP_FILE=$(mktemp)

# Função para fazer requisições em paralelo
make_requests() {
    local proc_id=$1
    local success=0
    local blocked=0
    
    for i in $(seq 1 $REQUESTS_PER_PROC); do
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" $URL)
        
        if [ "$HTTP_CODE" -eq 200 ]; then
            ((success++))
        elif [ "$HTTP_CODE" -eq 429 ]; then
            ((blocked++))
        fi
    done
    
    echo "$success $blocked" >> $TEMP_FILE
}

# Iniciar processos em paralelo
START_TIME=$(date +%s)

for proc in $(seq 1 $NUM_PROCESSES); do
    make_requests $proc &
done

# Aguardar todos os processos terminarem
wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Calcular totais
TOTAL_SUCCESS=0
TOTAL_BLOCKED=0

while read success blocked; do
    TOTAL_SUCCESS=$((TOTAL_SUCCESS + success))
    TOTAL_BLOCKED=$((TOTAL_BLOCKED + blocked))
done < $TEMP_FILE

# Limpar arquivo temporário
rm -f $TEMP_FILE

# Calcular requisições por segundo
if [ "$DURATION" -gt 0 ]; then
    REQ_PER_SEC=$(echo "scale=2; $TOTAL_REQUESTS / $DURATION" | bc)
else
    REQ_PER_SEC=$TOTAL_REQUESTS
fi

echo ""
echo "========================================="
echo "Resultado do Stress Test:"
echo "========================================="
echo "Duração: ${DURATION}s"
echo "Requisições/segundo: $REQ_PER_SEC"
echo ""
echo "Total de requisições: $TOTAL_REQUESTS"
echo "Sucesso (200): $TOTAL_SUCCESS"
echo "Bloqueadas (429): $TOTAL_BLOCKED"
echo "Perdidas: $((TOTAL_REQUESTS - TOTAL_SUCCESS - TOTAL_BLOCKED))"

if [ "$TOTAL_REQUESTS" -gt 0 ]; then
    SUCCESS_RATE=$(echo "scale=2; ($TOTAL_SUCCESS * 100) / $TOTAL_REQUESTS" | bc)
    BLOCK_RATE=$(echo "scale=2; ($TOTAL_BLOCKED * 100) / $TOTAL_REQUESTS" | bc)
    echo ""
    echo "Taxa de sucesso: ${SUCCESS_RATE}%"
    echo "Taxa de bloqueio: ${BLOCK_RATE}%"
fi
echo "========================================="
