package main

import (
	"adalbertofjr/desafio-rate-limiter/ajun"
	"fmt"
	"net/http"
	"time"
)

func main() {
	ajunRouter := ajun.NewRouter()
	ajunRouter.RateLimiter(5, time.Second*5)

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
