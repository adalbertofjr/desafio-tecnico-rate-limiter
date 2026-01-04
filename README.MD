# Rate Limiter - Desafio Técnico FullCycle

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![Tests](https://img.shields.io/badge/tests-67%20passing-brightgreen.svg)](https://github.com)
[![Coverage](https://img.shields.io/badge/coverage-93.5%25-brightgreen.svg)](https://github.com)
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
- ✅ **Suporte a proxies**: Detecta IP real via header `X-Forwarded-For` (load balancers, CDN)
- ✅ **IPv4 e IPv6**: Suporte completo para ambos protocolos
- ✅ **Strategy Pattern**: Backend plugável com interface para múltiplas implementações
- ✅ **In-memory storage**: Armazenamento local de alta performance
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

Copie o arquivo de exemplo para criar seu `.env`:

```bash
cp cmd/server/.env_example cmd/server/.env
```

Edite `cmd/server/.env` conforme necessário:

```env
# Configurações do Server
SERVER_PORT=:8080

# Configurações de Rate Limiting por IP
RATE_LIMITER_MAX_REQUESTS=5
RATE_LIMITER_TIME_DELAY=20s

# Configurações de Rate Limiting por Token
RATE_LIMITER_TOKEN_MAX_REQUESTS=7
RATE_LIMITER_TOKEN_TIME_DELAY=10s

# Configurações de Cleanup Automático
RATE_LIMITER_CLEANUP_INTERVAL=30s
RATE_LIMITER_TTL=2m

# Configuração do Backend Para Redis (opcional caso use Memory)
RATE_LIMITER_REDIS_ADDR=localhost:6379
```

### Variáveis de Ambiente

| Variável | Descrição | Exemplo | Padrão |
|----------|-----------|---------|--------|
| `SERVER_PORT` | Porta do servidor | `:8080` | - |
| `RATE_LIMITER_MAX_REQUESTS` | Número máximo de requisições por IP | `100` | - |
| `RATE_LIMITER_TIME_DELAY` | Tempo de bloqueio após exceder limite (IP) | `60s`, `5m`, `1h` | - |
| `RATE_LIMITER_TOKEN_MAX_REQUESTS` | Número máximo de requisições por token | `200` | - |
| `RATE_LIMITER_TOKEN_TIME_DELAY` | Tempo de bloqueio após exceder limite (token) | `60s`, `5m`, `1h` | - |
| `RATE_LIMITER_CLEANUP_INTERVAL` | Intervalo de execução do cleanup | `10m`, `30m`, `1h` | - |
| `RATE_LIMITER_TTL` | Tempo de vida dos dados antes da limpeza | `1h`, `2h`, `24h` | - |
| `RATE_LIMITER_REDIS_ADDR` | Endereço do servidor Redis | `localhost:6379` | - |

### Formatos de Duração

O projeto aceita durações no formato Go:
- `s` - segundos (ex: `30s`)
- `m` - minutos (ex: `5m`)
- `h` - horas (ex: `1h`)
- Combinados (ex: `1h30m`, `2h45m30s`)

## 🏃 Como Usar

### 🐳 Com Docker Compose (Recomendado)

A forma mais fácil de testar o projeto é usando Docker Compose, que sobe automaticamente a aplicação e o Redis:

```bash
# Subir todos os serviços (app + Redis + Redis Commander)
docker-compose up -d

# Ver logs
docker-compose logs -f app

# Parar todos os serviços
docker-compose down
```

Serviços disponíveis:
- **Aplicação**: http://localhost:8080
- **Redis Commander** (interface web): http://localhost:8081

### 💻 Executar localmente

Se preferir executar sem Docker:

```bash
# Desenvolvimento
go run cmd/server/main.go

# Build e execução
go build -o server cmd/server/main.go
./server
```

> **Nota**: Se o projeto estiver configurado para usar Redis (`ratelimiter.Redis` no `main.go`), você precisará ter o Redis rodando localmente ou via Docker.

### Integração em seu código

```go
package main

import (
    "adalbertofjr/desafio-rate-limiter/ajun"
    "adalbertofjr/desafio-rate-limiter/ajun/middleware/ratelimiter"
    "adalbertofjr/desafio-rate-limiter/cmd/configs"
    "adalbertofjr/desafio-rate-limiter/internal/infra/api"
    "context"
    "fmt"
    "net/http"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Carregar configurações do .env
    config, err := configs.LoadConfig(".")
    if err != nil {
        panic(err)
    }

    // Criar configuração do rate limiter
    rateLimiterConfig := ratelimiter.NewRateLimiterConfig(
        config.RateLimiterMaxRequests,
        config.ParseTimerDuration(config.RateLimiterTimeDelay),
        config.RateLimiterTokenMaxRequests,
        config.ParseTimerDuration(config.RateLimiterTokenTimeDelay),
        ratelimiter.Memory,  // ou ratelimiter.Redis
        config.RateLimiterRedisAddr,
        config.ParseTimerDuration(config.RateLimiterCleanupInterval),
        config.ParseTimerDuration(config.RateLimiterTTL),
    )

    // Criar router e aplicar middleware
    ajunRouter := ajun.NewRouter(ctx)
    ajunRouter.RateLimiter(rateLimiterConfig)

    // Adicionar handlers (use os handlers do pacote api ou crie os seus)
    ajunRouter.HandleFunc("/health", api.HealthHandler)
    ajunRouter.HandleFunc("/products", api.ListProductsHandler)

    // Iniciar servidor
    fmt.Println("Starting web server on port", config.ServerPort)
    http.ListenAndServe(config.ServerPort, ajunRouter.Handler)
}
```

**Nota**: Os handlers estão em `internal/infra/api/handlers.go`. Você pode usá-los diretamente ou criar seus próprios handlers.

### Alternar entre Memory e Redis

Para trocar o backend, edite `cmd/server/main.go` linha 25:

```go
// Backend Memory (em memória)
ratelimiter.Memory,

// Ou Backend Redis
ratelimiter.Redis,
```

Certifique-se de configurar `RATE_LIMITER_REDIS_ADDR` no `.env` se usar Redis.

## 📡 Testando o Rate Limiter

### Teste Rápido Manual

#### Requisição sem API Key (limitada por IP)

```bash
curl http://localhost:8080/health
```

#### Requisição com API Key (limitada por token)

```bash
curl -H "Api_key: seu-token-123" http://localhost:8080/products
```

#### Resposta quando bloqueado (429)

```
HTTP/1.1 429 Too Many Requests

You have reached the maximum number of requests or actions allowed within a certain time frame.
```

### 🎯 Script de Teste Automatizado

O projeto inclui um script para simular múltiplas requisições e validar o bloqueio:

```bash
# Dar permissão de execução
chmod +x scripts/test_rate_limit.sh

# Executar com valores padrão (30 requisições, 0.1s intervalo)
./scripts/test_rate_limit.sh

# Personalizar número de requisições e intervalo
./scripts/test_rate_limit.sh 50 0.05
```

**Saída esperada:**

```
=========================================
Teste de Rate Limiter
=========================================
URL: http://localhost:8080/health
Número de requisições: 30
Intervalo: 0.1s
=========================================

[14:23:10.123] Requisição #1: ✅ 200 OK
[14:23:10.234] Requisição #2: ✅ 200 OK
[14:23:10.345] Requisição #3: ✅ 200 OK
[14:23:10.456] Requisição #4: ✅ 200 OK
[14:23:10.567] Requisição #5: ✅ 200 OK
[14:23:10.678] Requisição #6: ⛔ 429 Too Many Requests
[14:23:10.789] Requisição #7: ⛔ 429 Too Many Requests
...

=========================================
Resumo:
=========================================
Total de requisições: 30
Sucesso (200): 5
Bloqueadas (429): 25
Taxa de bloqueio: 83.33%
=========================================
```

**Parâmetros do script:**
- `$1`: Número de requisições (padrão: 30)
- `$2`: Intervalo em segundos entre requisições (padrão: 0.1)

**Dica:** Configure um limite baixo no `.env` (ex: `RATE_LIMITER_MAX_REQUESTS=5`) para ver o bloqueio acontecer mais rápido.

**💡 Limpando o rate limit entre testes:**

Se você estiver bloqueado e quiser testar novamente, reinicie o app:

```bash
# Com Docker Compose
docker-compose restart app

# Ou executando localmente
# Ctrl+C e executar novamente
```

### 🌐 Teste com Múltiplos IPs

O projeto suporta `X-Forwarded-For` para ambientes com proxies/load balancers. Para testar com múltiplos IPs simulados:

```bash
# Dar permissão
chmod +x scripts/test_multiple_ips.sh

# 15 requisições distribuídas entre 3 IPs diferentes
./scripts/test_multiple_ips.sh 15 3 0.1
```

**Saída esperada:**
```
IP 192.168.1.1: 2 OK | 3 Blocked | 60.0% bloqueio
IP 192.168.1.2: 2 OK | 3 Blocked | 60.0% bloqueio
IP 192.168.1.3: 1 OK | 4 Blocked | 80.0% bloqueio
```

Cada IP tem seu próprio limite independente, como ocorre em produção com load balancers.

**Parâmetros:**
- `$1`: Total de requisições (padrão: 100)
- `$2`: Número de IPs diferentes (padrão: 5)
- `$3`: Intervalo entre requisições em segundos (padrão: 0.05)

### ⚡ Stress Test

Para teste de carga com requisições concorrentes:

```bash
chmod +x scripts/stress_test.sh

# 10 processos fazendo 20 requisições cada = 200 requisições totais
./scripts/stress_test.sh 20 10
```

**Saída esperada:**
```
=========================================
Resultado do Stress Test:
=========================================
Duração: 1s
Requisições/segundo: 50.00

Total de requisições: 50
Sucesso (200): 5
Bloqueadas (429): 45
Taxa de bloqueio: 90.00%
=========================================
```

**Parâmetros:**
- `$1`: Requisições por processo (padrão: 20)
- `$2`: Número de processos concorrentes (padrão: 10)

**Nota:** Se todas as requisições aparecerem bloqueadas (100%), reinicie o app antes do teste.

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
│           ├── storage_test.go   # Testes de storage
│           ├── interface.go      # Backend interface (Strategy)
│           ├── memory.go         # Implementação in-memory
│           ├── memory_test.go    # Testes do backend memory
│           ├── redis.go          # Implementação Redis
│           ├── redis_test.go     # Testes do backend Redis
│           └── types.go          # ClientIPData struct
├── cmd/
│   ├── configs/
│   │   └── configs.go            # Carregamento e validação de configs
│   └── server/
│       ├── main.go               # Entry point
│       └── .env_example          # Exemplo de configuração
├── internal/
│   └── infra/
│       └── api/
│           └── handlers.go       # HTTP handlers (health, products)
├── api/
│   └── health.http               # Exemplos de requisições HTTP
└── scripts/
    └── test_rate_limit.sh        # Script de teste de stress
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

// Observação: NewRedisBackend(ctx, "localhost:6379") cria o cliente internamente

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
middleware/ratelimiter: 92.5% ✅
storage:              79.7% ✅
memory backend:       86.5% ✅
redis backend:        78.4% ✅
```

**Total: 67 testes**
- 11 testes de integração (router)
- 32 testes unitários (ratelimiter)
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

## 🐳 Docker

O projeto inclui configuração completa de Docker Compose.

### Arquitetura do Compose

```yaml
services:
  app:              # Aplicação Rate Limiter
  redis:            # Redis (backend de armazenamento)
  redis-insight:    # Redis Commander (interface web)
```

### Comandos úteis

```bash
# Subir todos os serviços
docker-compose up -d

# Rebuild após alterações
docker-compose up -d --build

# Ver logs da aplicação
docker-compose logs -f app

# Ver logs do Redis
docker-compose logs -f redis

# Parar todos os serviços
docker-compose down

# Parar e limpar volumes (dados do Redis)
docker-compose down -v

# Acessar shell do container
docker-compose exec app sh
```

### Portas expostas

- `8080`: Aplicação
- `6379`: Redis
- `8081`: Redis Commander (http://localhost:8081)

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