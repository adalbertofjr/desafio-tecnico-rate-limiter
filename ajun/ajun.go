package ajun

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	remoteAddrs    = map[string]int{}
	requestEnabled = true
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

func (a *ajun) RateLimiter() {
	a.Handler = rateLimiter(a.router)
}

func rateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestEnabled == false {
			requestEnabled = false
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
			return
		}

		if remoteAddrs[strings.Split(r.RemoteAddr, ":")[0]] >= 5 {
			requestEnabled = false
			fmt.Printf("Addrs: %s disable to new requests\n", r.RemoteAddr)
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))

			go func() {
				time.Sleep(time.Second * 10)
				remoteAddrs[strings.Split(r.RemoteAddr, ":")[0]] = 0
				requestEnabled = true
				fmt.Printf("Addrs: %s enable to new requests\n", r.RemoteAddr)
			}()
			return
		}

		requestEnabled = true
		remoteAddrs[strings.Split(r.RemoteAddr, ":")[0]]++
		next.ServeHTTP(w, r)
	})
}
