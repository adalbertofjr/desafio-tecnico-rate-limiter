package ratelimiter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	MESSAGE_429 = "You have reached the maximum number of requests or actions allowed within a certain time frame."
)

type RateLimiter struct {
	config  RateLimiterConfig
	storage Storage
}

type RateLimiterConfig struct {
	Limit       int
	Delay       time.Duration
	TokenLimit  int
	TokenDelay  time.Duration
	Backend     StorageBackend
	Addr        string
	TimeCleanIn time.Duration
	TTL         time.Duration
}

func NewRateLimiter(ctx context.Context, config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		config: config,
		storage: *NewStorage(ctx,
			config.Backend,
			config.Addr,
			config.TimeCleanIn,
			config.TTL),
	}
}

func NewRateLimiterConfig(limit int, delay time.Duration, tokenLimit int, tokenDelay time.Duration, backend StorageBackend, addr string, timeCleanIn time.Duration, ttl time.Duration) RateLimiterConfig {
	return RateLimiterConfig{
		Limit:       limit,
		Delay:       delay,
		TokenLimit:  tokenLimit,
		TokenDelay:  tokenDelay,
		Backend:     backend,
		Addr:        addr,
		TimeCleanIn: timeCleanIn,
		TTL:         ttl,
	}
}

func (rl *RateLimiter) RateLimiterHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiToken := r.Header.Get("Api_key")
		clientIP := rl.getClientIP(r)

		if rl.isRemoteAddrDisabled(clientIP, apiToken) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(MESSAGE_429))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extrai o IP do cliente, considerando proxies e load balancers
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// 1. Tentar X-Forwarded-For (proxies/load balancers)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Formato: "client, proxy1, proxy2"
		// Pegar o primeiro IP (cliente real)
		ips := strings.Split(forwarded, ",")
		clientIP := strings.TrimSpace(ips[0])
		return clientIP // Já vem sem porta
	}

	// 2. Fallback: RemoteAddr (conexão direta)
	ip := r.RemoteAddr
	clientIP, _, err := net.SplitHostPort(ip)
	if err != nil {
		// Se falhar, retorna o IP como está (pode ser IPv6 sem porta)
		return ip
	}
	return clientIP
}

func (rl *RateLimiter) isRemoteAddrDisabled(clientIP string, apiToken string) bool {
	timeDisable, exists := rl.storage.GetTimeDisabledClientIP(clientIP)

	if exists && timeDisable.After(time.Now()) {
		return true
	}

	var maxRequests int
	var timeDelay time.Duration
	if apiToken != "" {
		maxRequests = rl.config.TokenLimit
		timeDelay = rl.config.TokenDelay
	} else {
		maxRequests = rl.config.Limit
		timeDelay = rl.config.Delay
	}

	// Incrementa e verifica atomicamente para evitar race conditions
	hostCountRequests := rl.storage.IncrementAndGetCount(clientIP)

	if hostCountRequests > maxRequests {
		rl.storage.DisableClientIP(clientIP, timeDelay)
		fmt.Printf("Disable host: %s - %s\n", clientIP, time.Now().Format(time.TimeOnly))

		time.AfterFunc(timeDelay, func() {
			rl.storage.ResetClientIP(clientIP)
			fmt.Printf("Enable host: %s - %s\n", clientIP, time.Now().Format(time.TimeOnly))
		})
		return true
	}

	return false
}

func (rl *RateLimiter) ResetGlobalState() {
	rl.storage.ResetDataClientIPs()
}
