package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	remoteAddrs        = map[string]int{}
	remoteAddrsDisable = map[string]time.Time{}
)

type rateLimiter struct {
	muRemoteAddrs        *sync.RWMutex
	muRemoteAddrsDisable *sync.RWMutex
	config               RateLimiterConfig
	// timeDelay            time.Duration
	// maxRequests          int
	// tokenTimeDelay       time.Duration
	// tokenMaxRequests     int
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
		host, _, err := net.SplitHostPort(ip)
		if err != nil {
			host = strings.Split(ip, ":")[0]
		}

		rl.addRemoteAddr(host)

		apiToken := r.Header.Get("Api_key")

		if rl.isRemoteAddrDisabled(host, apiToken) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *rateLimiter) addRemoteAddr(host string) {
	rl.muRemoteAddrs.Lock()
	defer rl.muRemoteAddrs.Unlock()
	remoteAddrs[host]++
	fmt.Printf("Request host: %s; QTD: %d; %s\n", host, remoteAddrs[host], time.Now().Format(time.TimeOnly))
}

func (rl *rateLimiter) resetRemoteAddrs(host string) {
	rl.muRemoteAddrs.Lock()
	remoteAddrs[host] = 0
	rl.muRemoteAddrs.Unlock()

	rl.muRemoteAddrsDisable.Lock()
	delete(remoteAddrsDisable, host)
	rl.muRemoteAddrsDisable.Unlock()
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
	defer rl.muRemoteAddrsDisable.Unlock()
	remoteAddrsDisable[host] = time.Now().Add(timeDelay)
}

func (rl *rateLimiter) isRemoteAddrDisabled(host string, apiToken string) bool {
	rl.muRemoteAddrsDisable.RLock()
	timeDisable, exists := remoteAddrsDisable[host]
	rl.muRemoteAddrsDisable.RUnlock()

	if exists && timeDisable.After(time.Now()) {
		return true
	}

	rl.muRemoteAddrs.RLock()
	hostCountRequests := remoteAddrs[host]
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

	if hostCountRequests > maxRequests && !exists {

		go func() {
			rl.addRemoteAddrDisable(host, apiToken)
			fmt.Printf("Disable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()

		go func() {
			time.Sleep(timeDelay)
			rl.resetRemoteAddrs(host)
			fmt.Printf("Enable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()
		return true
	}

	return false
}

func ResetGlobalState() {
	remoteAddrs = map[string]int{}
	remoteAddrsDisable = map[string]time.Time{}
}

// func ConfigureRateLimiter(maxRequests int, timeDelay time.Duration) {
// 	remoteAddrMaxRequests = maxRequests
// 	remoteAddrTimeDelay = timeDelay
// }
