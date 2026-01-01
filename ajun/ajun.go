package ajun

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	remoteAddrs           = map[string]int{}
	remoteAddrsDisable    = map[string]time.Time{}
	remoteAddrTimeDelay   = time.Second * 60
	remoteAddrMaxRequests = 20
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
		ip := r.RemoteAddr
		host, _, err := net.SplitHostPort(ip)
		if err == nil {
			host = strings.Split(ip, ":")[0]
		}

		addRemoteAddr(host)

		if isRemoteAddrDisabled(host) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func addRemoteAddr(host string) {
	remoteAddrs[host]++
	fmt.Printf("Request host: %s; QTD: %d; %s\n", host, remoteAddrs[host], time.Now().Format(time.TimeOnly))
}

func resetRemoteAddrs(host string) {
	remoteAddrs[host] = 0
	delete(remoteAddrsDisable, host)
}

func addRemoteAddrDisable(host string) {
	remoteAddrsDisable[host] = time.Now().Add(remoteAddrTimeDelay)
}

func isRemoteAddrDisabled(host string) bool {
	timeDisable, exists := remoteAddrsDisable[host]
	if exists && timeDisable.After(time.Now()) {
		return true
	}

	if remoteAddrs[host] > remoteAddrMaxRequests && !exists {
		addRemoteAddrDisable(host)
		fmt.Printf("Disable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		go func() {
			time.Sleep(remoteAddrTimeDelay)
			resetRemoteAddrs(host)
			fmt.Printf("Enable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()
		return true
	}
	return false
}
