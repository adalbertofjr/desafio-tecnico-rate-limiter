package middleware

import (
	"adalbertofjr/desafio-rate-limiter/ajun/internal/database/local"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	muRemoteAddrs        *sync.RWMutex
	muRemoteAddrsDisable *sync.RWMutex
	config               RateLimiterConfig
	datasource           *local.DataSource
}

type RateLimiterConfig struct {
	Limit      int
	Delay      time.Duration
	TokenLimit int
	TokenDelay time.Duration
}

func NewRateLimiter(config RateLimiterConfig) *rateLimiter {
	return &rateLimiter{
		muRemoteAddrs:        &sync.RWMutex{},
		muRemoteAddrsDisable: &sync.RWMutex{},
		config:               config,
		datasource:           local.InitDataSource(),
	}
}

func NewRateLimiterConfig(limit int, delay time.Duration, tokenLimit int, tokenDelay time.Duration) RateLimiterConfig {
	return RateLimiterConfig{
		Limit:      limit,
		Delay:      delay,
		TokenLimit: tokenLimit,
		TokenDelay: tokenDelay,
	}
}

func (rl *rateLimiter) RateLimiterHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		clientIP, _, err := net.SplitHostPort(ip)
		if err != nil {
			clientIP = strings.Split(ip, ":")[0]
		}

		rl.datasource.AddClientIP(clientIP)
		apiToken := r.Header.Get("Api_key")

		if rl.isRemoteAddrDisabled(clientIP, apiToken) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *rateLimiter) addRemoteAddrDisable(host string, apiToken string) {
	// Definir limites para tokens de API, se fornecidos
	var timeDelay time.Duration
	if apiToken != "" {
		timeDelay = rl.config.TokenDelay
	} else {
		timeDelay = rl.config.Delay
	}

	rl.muRemoteAddrsDisable.Lock()
	rl.datasource.DisableClientIP(host, timeDelay)
	rl.muRemoteAddrsDisable.Unlock()
}

func (rl *rateLimiter) isRemoteAddrDisabled(host string, apiToken string) bool {
	rl.muRemoteAddrsDisable.RLock()

	timeDisable, exists := rl.datasource.GetTimeDisabledClientIP(host)
	rl.muRemoteAddrsDisable.RUnlock()

	if exists && timeDisable.After(time.Now()) {
		return true
	}

	rl.muRemoteAddrs.RLock()
	hostCountRequests := rl.datasource.GetClientIPCount(host)
	rl.muRemoteAddrs.RUnlock()

	var maxRequests int
	var timeDelay time.Duration
	if apiToken != "" {
		maxRequests = rl.config.TokenLimit
		timeDelay = rl.config.TokenDelay
	} else {
		maxRequests = rl.config.Limit
		timeDelay = rl.config.Delay
	}

	if hostCountRequests > maxRequests {

		go func() {
			rl.addRemoteAddrDisable(host, apiToken)
			fmt.Printf("Disable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()

		go func() {
			time.Sleep(timeDelay)
			rl.datasource.ResetClientIP(host)
			fmt.Printf("Enable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()
		return true
	}

	return false
}

func (rl *rateLimiter) ResetGlobalState() {
	rl.datasource.ResetDataClientIPs()
}
