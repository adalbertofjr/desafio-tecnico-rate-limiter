package ajun

import (
	"adalbertofjr/desafio-rate-limiter/ajun/middleware/ratelimiter"
	"context"
	"net/http"
)

type ajun struct {
	ctx         context.Context
	router      *http.ServeMux
	Handler     http.Handler
	rateLimiter interface{ ResetGlobalState() } // Interface para acessar reset
}

func newMux() *http.ServeMux {
	return http.NewServeMux()
}

func NewRouter(ctx context.Context) *ajun {
	mux := newMux()
	return &ajun{
		ctx:     ctx,
		router:  mux,
		Handler: mux,
	}
}

func (a *ajun) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	a.router.HandleFunc(pattern, handler)
}

func (a *ajun) RateLimiter(config ratelimiter.RateLimiterConfig) {
	rateLimiter := ratelimiter.NewRateLimiter(a.ctx, config)
	a.rateLimiter = rateLimiter
	a.Handler = rateLimiter.RateLimiterHandler(a.router)
}

// ResetGlobalState expõe o método reset do rate limiter para testes
func (a *ajun) ResetGlobalState() {
	if a.rateLimiter != nil {
		a.rateLimiter.ResetGlobalState()
	}
}
