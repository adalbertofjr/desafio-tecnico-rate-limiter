package main

import (
	"adalbertofjr/desafio-rate-limiter/ajun"
	"adalbertofjr/desafio-rate-limiter/cmd/configs"

	"adalbertofjr/desafio-rate-limiter/ajun/middleware/ratelimiter"

	"context"
	"fmt"
	"net/http"
	"time"
)

func main() {
	config := loadConfigs()
	limitMaxRequests := config.RateLimiterMaxRequests
	timeDelay, err := time.ParseDuration(config.RateLimiterTimeDelay)
	if err != nil {
		fmt.Printf("Invalid duration format for RATE_LIMITER_TIME_DELAY: %s. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"\n", config.RateLimiterTimeDelay)
		panic(err)
	}
	tokenMaxRequests := config.RateLimiterTokenMaxRequests
	tokenTimeDelay, err := time.ParseDuration(config.RateLimiterTokenTimeDelay)
	if err != nil {
		fmt.Printf("Invalid duration format for RATE_LIMITER_TOKEN_TIME_DELAY: %s. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"\n", config.RateLimiterTokenTimeDelay)
		panic(err)
	}
	timeCleanIn, err := time.ParseDuration(config.RateLimiterCleanupInterval)
	if err != nil {
		fmt.Printf("Invalid duration format for RATE_LIMITER_CLEANUP_INTERVAL: %s. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"\n", config.RateLimiterCleanupInterval)
		panic(err)
	}
	ttl, err := time.ParseDuration(config.RateLimiterTTL)
	if err != nil {
		fmt.Printf("Invalid duration format for RATE_LIMITER_TTL: %s. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"\n", config.RateLimiterTTL)
		panic(err)
	}

	addrBackend := config.RateLimiterRedisAddr

	rateLimiterConfig := ratelimiter.NewRateLimiterConfig(
		limitMaxRequests,
		timeDelay,
		tokenMaxRequests,
		tokenTimeDelay,
		ratelimiter.Redis,
		addrBackend,
		timeCleanIn,
		ttl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ajunRouter := ajun.NewRouter(ctx)
	ajunRouter.RateLimiter(rateLimiterConfig)

	ajunRouter.HandleFunc("/health", healthHandler)
	ajunRouter.HandleFunc("/products", listProductsHandler)

	addrServer := config.ServerPort
	fmt.Println("Starting web server on port", addrServer)
	http.ListenAndServe(addrServer, ajunRouter.Handler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"OK"}`))
}

func listProductsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`"Product 1", "Product 2", "Product 3"`))
}

func loadConfigs() *configs.Config {
	config, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}
	return config
}
