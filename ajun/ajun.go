package ajun

import (
	"adalbertofjr/desafio-rate-limiter/ajun/middleware"
	"net/http"
)

type ajun struct {
	router      *http.ServeMux
	Handler     http.Handler
	rateLimiter interface{ ResetGlobalState() } // Interface para acessar reset
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

func (a *ajun) RateLimiter(config middleware.RateLimiterConfig) {
	rateLimiter := middleware.NewRateLimiter(config)
	a.rateLimiter = rateLimiter
	a.Handler = rateLimiter.RateLimiterHandler(a.router)
}

// ResetGlobalState expõe o método reset do rate limiter para testes
func (a *ajun) ResetGlobalState() {
	if a.rateLimiter != nil {
		a.rateLimiter.ResetGlobalState()
	}
}
