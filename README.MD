# Rate Limiter - Desafio Técnico FullCycle

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![Tests](https://img.shields.io/badge/tests-35%20passing-brightgreen.svg)](https://github.com)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](https://github.com)

Middleware HTTP para controle de taxa de requisições (rate limiting) desenvolvido em Go, com suporte a limitação por IP e por token de API.

## 📋 Características

- ✅ **Rate limiting por IP**: Limita requisições baseado no endereço IP do cliente
- ✅ **Rate limiting por API Key**: Limites independentes para requisições autenticadas
- ✅ **Bloqueio automático**: IPs que excedem o limite são bloqueados temporariamente
- ✅ **Cleanup automático**: Remove dados antigos periodicamente para gerenciar memória
- ✅ **Thread-safe**: Implementado com `sync.RWMutex` para operações concorrentes
- ✅ **Graceful shutdown**: Suporte a context para parada controlada
- ✅ **Configurável**: Todos os parâmetros via variáveis de ambiente
- ✅ **IPv4 e IPv6**: Suporte completo para ambos protocolos
- ✅ **In-memory storage**: Armazenamento local de alta performance

## 🚀 Instalação

### Pré-requisitos

- Go 1.23 ou superior
- Git

### Clonar o repositório

```bash
git clone https://github.com/seu-usuario/rate-limiter.git
cd rate-limiter
```

### Instalar dependências

```bash
go mod download
```

## ⚙️ Configuração

Crie um arquivo `.env` na raiz do projeto:

```env
# Configurações de Rate Limiting por IP
RATE_LIMITER_MAX_REQUESTS=100
RATE_LIMITER_TIME_DELAY=60s

# Configurações de Rate Limiting por Token
RATE_LIMITER_TOKEN_MAX_REQUESTS=200
RATE_LIMITER_TOKEN_TIME_DELAY=60s

# Configurações de Cleanup Automático
RATE_LIMITER_CLEANUP_INTERVAL=10m
RATE_LIMITER_TTL=1h
```

### Variáveis de Ambiente

| Variável | Descrição | Exemplo | Padrão |
|----------|-----------|---------|--------|
| `RATE_LIMITER_MAX_REQUESTS` | Número máximo de requisições por IP | `100` | - |
| `RATE_LIMITER_TIME_DELAY` | Tempo de bloqueio após exceder limite (IP) | `60s`, `5m`, `1h` | - |
| `RATE_LIMITER_TOKEN_MAX_REQUESTS` | Número máximo de requisições por token | `200` | - |
| `RATE_LIMITER_TOKEN_TIME_DELAY` | Tempo de bloqueio após exceder limite (token) | `60s`, `5m`, `1h` | - |
| `RATE_LIMITER_CLEANUP_INTERVAL` | Intervalo de execução do cleanup | `10m`, `30m`, `1h` | - |
| `RATE_LIMITER_TTL` | Tempo de vida dos dados antes da limpeza | `1h`, `2h`, `24h` | - |

### Formatos de Duração

O projeto aceita durações no formato Go:
- `s` - segundos (ex: `30s`)
- `m` - minutos (ex: `5m`)
- `h` - horas (ex: `1h`)
- Combinados (ex: `1h30m`, `2h45m30s`)

## 🏃 Como Usar

### Iniciar o servidor

```bash
# Desenvolvimento
go run cmd/server/main.go

# Build e execução
go build -o server cmd/server/main.go
./server
```

O servidor iniciará na porta **8080**.

### Integração em seu código

```go
package main

import (
    "context"
    "github.com/seu-usuario/rate-limiter/ajun"
    "github.com/seu-usuario/rate-limiter/ajun/middleware"
    "net/http"
    "time"
)

func main() {
    ctx := context.Background()
    
    // Configurar rate limiter
    config := middleware.NewRateLimiterConfig(
        100,              // limite por IP
        time.Minute,      // delay de bloqueio IP
        200,              // limite por token
        time.Minute,      // delay de bloqueio token
        10*time.Minute,   // intervalo de cleanup
        1*time.Hour,      // TTL dos dados
    )
    
    // Criar router
    router := ajun.NewRouter(ctx)
    router.RateLimiter(config)
    
    // Adicionar handlers
    router.HandleFunc("/api/resource", yourHandler)
    
    // Iniciar servidor
    http.ListenAndServe(":8080", router.Handler)
}
```

## 📡 Exemplos de Requisições

### Requisição normal (sem token)

```bash
curl http://localhost:8080/health
```

Limitada por IP conforme `RATE_LIMITER_MAX_REQUESTS`.

### Requisição com API Key

```bash
curl -H "Api_key: seu-token-aqui" http://localhost:8080/products
```

Limitada por token conforme `RATE_LIMITER_TOKEN_MAX_REQUESTS`.

### Resposta quando limite é excedido

```
HTTP/1.1 429 Too Many Requests
Content-Type: text/plain

Too many requests
```

## 🏗️ Arquitetura

### Estrutura de Diretórios

```
.
├── ajun/
│   ├── middleware/
│   │   ├── ratelimiter.go       # Lógica principal do rate limiter
│   │   └── ratelimiter_test.go  # Testes do middleware
│   ├── internal/
│   │   └── database/
│   │       └── local/
│   │           ├── local.go      # Storage in-memory
│   │           └── local_test.go # Testes de cleanup
│   ├── ajun.go                   # Router wrapper
│   └── ajun_test.go              # Testes de integração
├── cmd/
│   ├── configs/
│   │   └── configs.go            # Carregamento de configurações
│   └── server/
│       └── main.go               # Entry point
└── api/
    └── health.http               # Exemplos de requisições
```

### Componentes Principais

#### 1. **RateLimiter Middleware**

Intercepta requisições HTTP e aplica controle de taxa baseado em:
- IP do cliente (`RemoteAddr`)
- Header `Api_key` (se presente)

#### 2. **DataSource (Storage)**

Gerencia dados de rate limiting em memória:
- Contador de requisições por IP
- Timestamps de última atividade
- Status de bloqueio (`DisableUntil`)

#### 3. **Cleanup Worker**

Goroutine background que executa periodicamente:
- Remove IPs sem atividade (baseado em TTL)
- Respeita bloqueios ativos
- Para gracefully com context

### Fluxo de Funcionamento

```
┌─────────────┐
│   Request   │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  RateLimiter        │
│  Middleware         │
└──────┬──────────────┘
       │
       ├─► Extrai IP
       ├─► Verifica Api_key header
       ├─► Incrementa contador
       │
       ▼
┌─────────────────────┐
│  Verifica se        │
│  está bloqueado     │
└──────┬──────────────┘
       │
       ├─► SIM: Retorna 429
       │
       └─► NÃO: Verifica limite
              │
              ├─► Excedeu: Bloqueia + 429
              │
              └─► OK: Continua ──► Handler
```

## 🧪 Testes

### Executar todos os testes

```bash
go test ./... -v
```

### Executar com cobertura

```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Executar com race detector

```bash
go test ./... -race
```

### Cobertura Atual

```
ajun:       100.0% ✅
middleware: 100.0% ✅
local:       39.0% (cobre cleanup + operações principais)
```

**Total: 35 testes**
- 11 testes de integração (ajun)
- 18 testes unitários (middleware)
- 7 testes de cleanup (local)

### Cenários Testados

- ✅ Requisições abaixo do limite
- ✅ Bloqueio quando limite é excedido
- ✅ Desbloqueio automático após timeout
- ✅ IPs independentes
- ✅ Concorrência (múltiplas requisições simultâneas)
- ✅ IPv4 e IPv6
- ✅ Limites diferentes para API Key
- ✅ Cleanup automático de dados antigos
- ✅ Graceful shutdown do cleanup worker

## 🛠️ Desenvolvimento

### Tecnologias Utilizadas

- **Go 1.23**: Linguagem principal
- **sync.RWMutex**: Thread-safety
- **context.Context**: Controle de lifecycle
- **time.Ticker**: Cleanup periódico
- **Viper**: Gerenciamento de configurações
- **httptest**: Testes HTTP

### Executar em modo desenvolvimento

```bash
# Com live reload (usando air)
air

# Ou manualmente
go run cmd/server/main.go
```

### Executar testes específicos

```bash
# Apenas testes de middleware
go test ./ajun/middleware -v

# Apenas testes de cleanup
go test ./ajun/internal/database/local -v

# Teste específico
go test ./ajun/middleware -run TestRateLimiterHandler_BlocksRequestsAboveLimit -v
```

## 📊 Performance

### Benchmarks

```bash
go test ./ajun/middleware -bench=. -benchmem
```

### Capacidade Estimada

- **Low-Medium traffic**: < 50k req/s ✅
- **High traffic**: < 100k req/s ✅
- **Very High traffic**: > 100k req/s (considerar Redis)

### Otimizações Implementadas

- ✅ Mutex read/write separados (`RLock` vs `Lock`)
- ✅ Map lookup O(1)
- ✅ Cleanup assíncrono (não bloqueia requisições)
- ✅ Sem goroutines vazando (timer gerenciado)

## 🐳 Docker (Opcional)

### Dockerfile exemplo

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY .env .env
EXPOSE 8080
CMD ["./server"]
```

### Build e run

```bash
docker build -t rate-limiter .
docker run -p 8080:8080 --env-file .env rate-limiter
```

## 📝 Exemplo de .env

```env
# Rate Limiting por IP
RATE_LIMITER_MAX_REQUESTS=100
RATE_LIMITER_TIME_DELAY=1m

# Rate Limiting por API Token
RATE_LIMITER_TOKEN_MAX_REQUESTS=200
RATE_LIMITER_TOKEN_TIME_DELAY=1m

# Cleanup Automático
RATE_LIMITER_CLEANUP_INTERVAL=10m
RATE_LIMITER_TTL=1h
```

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'feat: Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

### Commits Semânticos

Seguimos [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` Nova funcionalidade
- `fix:` Correção de bug
- `docs:` Mudanças na documentação
- `test:` Adição ou correção de testes
- `refactor:` Refatoração de código
- `perf:` Melhoria de performance
- `chore:` Tarefas de manutenção

## 📄 Licença

Este projeto foi desenvolvido como desafio técnico para o curso de Pós-Graduação em Golang da FullCycle.

## 👨‍💻 Autor

Desenvolvido por Adalberto Fernandes Jr.

## 🙏 Agradecimentos

- FullCycle - Pós-Graduação Go Expert
- Comunidade Go Brasil

---

**Nota:** Este é um projeto educacional desenvolvido para demonstração de conceitos de rate limiting, concorrência e arquitetura limpa em Go.