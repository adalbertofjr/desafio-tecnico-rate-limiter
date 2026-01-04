# Rate Limiter - Desafio Técnico FullCycle

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![Tests](https://img.shields.io/badge/tests-53%20passing-brightgreen.svg)](https://github.com)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](https://github.com)
[![Architecture](https://img.shields.io/badge/pattern-Strategy-orange.svg)](https://refactoring.guru/design-patterns/strategy)

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
- ✅ **Strategy Pattern**: Backend plugável com interface para múltiplas implementações
- ✅ **In-memory storage**: Armazenamento local de alta performance com deep copy
- ✅ **Redis storage**: Backend Redis com serialização JSON e thread-safety
- ✅ **Extensível**: Fácil adição de novos backends (PostgreSQL, DynamoDB, etc.)

## 🚀 Instalação

### Pré-requisitos

- Go 1.23 ou superior
- Git
- Redis (opcional, apenas se usar RedisBackend)

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

# Configuração do Backend (opcional)
RATE_LIMITER_BACKEND=memory  # ou 'redis'
RATE_LIMITER_REDIS_ADDR=localhost:6379
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
| `RATE_LIMITER_BACKEND` | Backend de armazenamento | `memory`, `redis` | `memory` |
| `RATE_LIMITER_REDIS_ADDR` | Endereço do servidor Redis | `localhost:6379` | - |

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
    config := ratelimiter.NewRateLimiterConfig(
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

// Exemplo com backend Redis
func mainWithRedis() {
    ctx := context.Background()
    
    // Criar backend Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    redisBackend := ratelimiter.NewRedisBackend(ctx, redisClient)
    
    // Criar Storage com backend Redis
    storage := ratelimiter.NewStorage(
        ctx,
        redisBackend,           // Strategy Pattern em ação
        10*time.Minute,         // cleanup interval
        1*time.Hour,            // TTL
    )
    
    // Configurar rate limiter
    config := ratelimiter.NewRateLimiterConfig(
        100,              // limite por IP
        time.Minute,      // delay de bloqueio IP
        200,              // limite por token
        time.Minute,      // delay de bloqueio token
        10*time.Minute,   // intervalo de cleanup
        1*time.Hour,      // TTL dos dados
    )
    
    router := ajun.NewRouter(ctx)
    router.RateLimiter(config)
    // Storage com Redis será usado automaticamente
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
├── ajun/                         # Framework/Router customizado
│   ├── ajun.go                   # Router wrapper
│   ├── ajun_test.go              # Testes de integração
│   └── middleware/
│       └── ratelimiter/
│           ├── ratelimiter.go    # Lógica principal do rate limiter
│           ├── ratelimiter_test.go
│           ├── storage.go        # Camada de storage com cleanup
│           ├── interface.go      # Backend interface (Strategy)
│           ├── memory.go         # Implementação in-memory
│           ├── memory_test.go    # Testes do backend memory
│           ├── redis.go          # Implementação Redis
│           ├── redis_test.go     # Testes do backend Redis
│           └── types.go          # ClientIPData struct
├── cmd/
│   ├── configs/
│   │   └── configs.go            # Carregamento de configurações
│   └── server/
│       └── main.go               # Entry point
└── api/
    └── health.http               # Exemplos de requisições HTTP
```

### Componentes Principais

#### 1. **RateLimiter Middleware**

Intercepta requisições HTTP e aplica controle de taxa baseado em:
- IP do cliente (`RemoteAddr`)
- Header `Api_key` (se presente)

#### 2. **Storage Layer**

Gerencia dados de rate limiting:
- Contador de requisições por IP
- Timestamps de última atividade
- Status de bloqueio (`DisableUntil`)
- Integração com backend via interface

#### 3. **Backend Interface (Strategy Pattern)**

Interface que define operações de armazenamento:
```go
type Backend interface {
    Get(clientIP string) (*ClientIPData, error)
    Set(clientIP string, data *ClientIPData) error
    Delete(clientIP string) error
    List() (map[string]*ClientIPData, error)
    Clear() error
}
```

**Implementações disponíveis:**
- `MemoryBackend`: In-memory com deep copy e thread-safety
- `RedisBackend`: Redis com serialização JSON, mutex e miniredis para testes
- **Futuro**: `PostgreSQLBackend`, `DynamoDBBackend`, etc.

#### 4. **Cleanup Worker**

Goroutine background que executa periodicamente:
- Remove IPs sem atividade (baseado em TTL)
- Respeita bloqueios ativos
- Para gracefully com context

### Design Patterns

#### Strategy Pattern - Backend Plugável

O projeto implementa o **Strategy Pattern** para permitir diferentes estratégias de armazenamento sem alterar a lógica do rate limiter:

```go
// Interface Strategy
type Backend interface {
    Get(clientIP string) (*ClientIPData, error)
    Set(clientIP string, data *ClientIPData) error
    Delete(clientIP string) error
    List() (map[string]*ClientIPData, error)
}

// Estratégia Concreta 1: In-Memory
type MemoryBackend struct {
    mu   sync.RWMutex
    data map[string]*ClientIPData
}

// Estratégia Concreta 2: Redis
type RedisBackend struct {
    ctx    context.Context
    client *redis.Client
    mu     sync.RWMutex
}

// Context que usa a Strategy
type Storage struct {
    backend Backend  // <- Composição com interface
    // ... outros campos
}
```

**Benefícios:**
- ✅ **Open/Closed Principle**: Aberto para extensão, fechado para modificação
- ✅ **Testabilidade**: Mock fácil do backend para testes
- ✅ **Flexibilidade**: Trocar backend em runtime se necessário
- ✅ **Separação de responsabilidades**: Storage não conhece detalhes de implementação

**Como adicionar novo backend:**
```go
// 1. Implementar a interface Backend
type PostgreSQLBackend struct {
    db *sql.DB
}

func (p *PostgreSQLBackend) Get(clientIP string) (*ClientIPData, error) {
    // implementação específica do PostgreSQL
}
// ... implementar outros métodos

// 2. Usar no Storage
backend := NewPostgreSQLBackend(connStr)
storage := NewStorage(ctx, backend, cleanupInterval, ttl)

// Pronto! O resto do código continua funcionando
```

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
router:              100.0% ✅
middleware/ratelimiter: 100.0% ✅
storage:             100.0% ✅
memory backend:      100.0% ✅
redis backend:       100.0% ✅
```

**Total: 53 testes**
- 11 testes de integração (router)
- 18 testes unitários (ratelimiter)
- 12 testes de backend memory
- 12 testes de backend Redis (com miniredis)

**Benchmarks:**
- `BenchmarkMemoryBackend_Set`: ~35ns/op (0 alocações)
- `BenchmarkMemoryBackend_Get`: ~22ns/op (0 alocações)
- `BenchmarkMemoryBackend_ConcurrentSet`: ~202ns/op
- `BenchmarkMemoryBackend_ConcurrentGet`: ~110ns/op

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
- ✅ Deep copy no backend para isolamento de dados
- ✅ Thread-safety testado com race detector
- ✅ Backend plugável via Strategy Pattern
- ✅ Redis backend com miniredis (testes sem servidor real)
- ✅ Serialização/deserialização JSON no Redis
- ✅ Clear() method para reset de dados nos testes

## 🛠️ Desenvolvimento

### Tecnologias e Padrões

- **Go 1.23**: Linguagem principal
- **Strategy Pattern**: Backend plugável e extensível
- **sync.RWMutex**: Thread-safety com read/write locks
- **context.Context**: Controle de lifecycle e graceful shutdown
- **time.Ticker**: Cleanup periódico
- **Viper**: Gerenciamento de configurações
- **httptest**: Testes HTTP
- **Deep Copy**: Isolamento de dados no backend
- **Redis**: Cliente go-redis/v9 com suporte a JSON
- **miniredis**: Mock in-memory do Redis para testes

### Executar em modo desenvolvimento

```bash
go run cmd/server/main.go
```

### Executar testes específicos

```bash
# Apenas testes de middleware
go test ./ajun/middleware/ratelimiter -v

# Apenas testes do backend memory
go test ./ajun/middleware/ratelimiter -run TestMemoryBackend -v

# Apenas testes do backend Redis
go test ./ajun/middleware/ratelimiter -run TestRedis -v

# Teste específico
go test ./ajun/middleware/ratelimiter -run TestRateLimiterHandler_BlocksRequestsAboveLimit -v

# Benchmarks do backend
go test ./ajun/middleware/ratelimiter -bench=BenchmarkMemoryBackend -benchmem
```

## 📊 Performance

### Benchmarks

```bash
go test ./ajun/middleware/ratelimiter -bench=. -benchmem
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
- ✅ Deep copy apenas no `List()` (Get/Set sem alocações extras)
- ✅ Backend interface sem overhead (chamadas diretas)
- ✅ Zero alocações nas operações Get/Set (~22-35ns/op)

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