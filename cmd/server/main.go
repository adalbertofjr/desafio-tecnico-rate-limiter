package main

import (
	"adalbertofjr/desafio-rate-limiter/ajun"
	"adalbertofjr/desafio-rate-limiter/cmd/configs"
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

	ajunRouter := ajun.NewRouter()
	ajunRouter.RateLimiter(limitMaxRequests, timeDelay)

	ajunRouter.HandleFunc("/health", healthHandler)
	ajunRouter.HandleFunc("/products", listProductsHandler)

	addr := ":8080"
	fmt.Println("Starting web server on port:", addr)
	http.ListenAndServe(addr, ajunRouter.Handler)
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
