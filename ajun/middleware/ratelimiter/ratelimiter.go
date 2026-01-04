package ratelimiter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
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
	TimeCleanIn time.Duration
	TTL         time.Duration
}

func NewRateLimiter(ctx context.Context, config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		config: config,
		storage: *NewStorage(ctx,
			NewRedisBackend(ctx),
			config.TimeCleanIn,
			config.TTL),
	}
}

func NewRateLimiterConfig(limit int, delay time.Duration, tokenLimit int, tokenDelay time.Duration, timeCleanIn time.Duration, ttl time.Duration) RateLimiterConfig {
	return RateLimiterConfig{
		Limit:       limit,
		Delay:       delay,
		TokenLimit:  tokenLimit,
		TokenDelay:  tokenDelay,
		TimeCleanIn: timeCleanIn,
		TTL:         ttl,
	}
}

func (rl *RateLimiter) RateLimiterHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		clientIP, _, err := net.SplitHostPort(ip)
		if err != nil {
			clientIP = strings.Split(ip, ":")[0]
		}

		apiToken := r.Header.Get("Api_key")

		if rl.isRemoteAddrDisabled(clientIP, apiToken) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
			return
		}

		rl.storage.AddClientIP(clientIP)
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) isRemoteAddrDisabled(host string, apiToken string) bool {
	timeDisable, exists := rl.storage.GetTimeDisabledClientIP(host)

	if exists && timeDisable.After(time.Now()) {
		return true
	}

	hostCountRequests := rl.storage.GetClientIPCount(host)

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
		rl.storage.DisableClientIP(host, timeDelay)
		fmt.Printf("Disable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))

		time.AfterFunc(timeDelay, func() {
			rl.storage.ResetClientIP(host)
			fmt.Printf("Enable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		})
		return true
	}

	return false
}

func (rl *RateLimiter) ResetGlobalState() {
	rl.storage.ResetDataClientIPs()
}
