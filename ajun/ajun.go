package ajun

import (
	"adalbertofjr/desafio-rate-limiter/ajun/middleware"
	"net/http"
	"time"
)

type ajun struct {
	router  *http.ServeMux
	Handler http.Handler
}

func newMux() *http.ServeMux {
	return http.NewServeMux()
}

func NewRouter() *ajun {
	mux := newMux()
	return &ajun{
		router:  mux,
		Handler: mux,
	}
}

func (a *ajun) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	a.router.HandleFunc(pattern, handler)
}

func (a *ajun) RateLimiter(limit int, delay time.Duration) {
	rateLimiter := middleware.NewRateLimiter(limit, delay)
	a.Handler = rateLimiter.RateLimiterHandler(a.router)
}
