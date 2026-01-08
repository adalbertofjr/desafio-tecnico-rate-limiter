# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copiar go.mod e go.sum primeiro para cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Build da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copiar binário compilado
COPY --from=builder /app/server .

# Copiar arquivo .env de exemplo (será sobrescrito por env_file no compose)
COPY --from=builder /app/.env_example .env

EXPOSE 8080

CMD ["./server"]
