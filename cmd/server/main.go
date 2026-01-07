package main

import (
	"adalbertofjr/desafio-rate-limiter/ajun"
	"adalbertofjr/desafio-rate-limiter/cmd/configs"
	"adalbertofjr/desafio-rate-limiter/internal/infra/api"

	"adalbertofjr/desafio-rate-limiter/ajun/middleware/ratelimiter"

	"context"
	"fmt"
	"net/http"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := loadConfigs()

	rateLimiterConfig := ratelimiter.NewRateLimiterConfig(
		config.RateLimiterMaxRequests,
		config.ParseTimerDuration(config.RateLimiterTimeDelay),
		config.RateLimiterTokenMaxRequests,
		config.ParseTimerDuration(config.RateLimiterTokenTimeDelay),
		ratelimiter.Redis,
		config.RateLimiterRedisAddr,
		config.ParseTimerDuration(config.RateLimiterCleanupInterval),
		config.ParseTimerDuration(config.RateLimiterTTL))

	ajunRouter := ajun.NewRouter(ctx)
	ajunRouter.RateLimiter(rateLimiterConfig)

	ajunRouter.HandleFunc("/health", api.HealthHandler)
	ajunRouter.HandleFunc("/products", api.ListProductsHandler)

	addrServer := config.ServerPort
	fmt.Println("Starting web server on port", addrServer)
	if err := http.ListenAndServe(addrServer, ajunRouter.Handler); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
}

func loadConfigs() *configs.Config {
	config, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	return config
}
