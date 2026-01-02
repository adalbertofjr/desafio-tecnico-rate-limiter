package ajun

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	remoteAddrs           = map[string]int{}
	remoteAddrsDisable    = map[string]time.Time{}
	remoteAddrTimeDelay   = time.Second * 5
	remoteAddrMaxRequests = 4
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
	rateLimiter := &rateLimiter{
		muRemoteAddrs:        &sync.RWMutex{},
		muRemoteAddrsDisable: &sync.RWMutex{},
		timeDelay:            remoteAddrTimeDelay,
		maxRequests:          remoteAddrMaxRequests,
	}

	a.Handler = rateLimiter.rateLimiterHandler(a.router)
}

type rateLimiter struct {
	muRemoteAddrs        *sync.RWMutex
	muRemoteAddrsDisable *sync.RWMutex
	timeDelay            time.Duration
	maxRequests          int
}

func (rl *rateLimiter) rateLimiterHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		host, _, err := net.SplitHostPort(ip)
		if err != nil {
			host = strings.Split(ip, ":")[0]
		}

		rl.addRemoteAddr(host)

		if rl.isRemoteAddrDisabled(host) {
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

func (rl *rateLimiter) addRemoteAddrDisable(host string) {
	rl.muRemoteAddrsDisable.Lock()
	defer rl.muRemoteAddrsDisable.Unlock()
	remoteAddrsDisable[host] = time.Now().Add(rl.timeDelay)
}

func (rl *rateLimiter) isRemoteAddrDisabled(host string) bool {
	rl.muRemoteAddrsDisable.RLock()
	timeDisable, exists := remoteAddrsDisable[host]
	rl.muRemoteAddrsDisable.RUnlock()

	if exists && timeDisable.After(time.Now()) {
		return true
	}

	rl.muRemoteAddrs.RLock()
	hostCountRequests := remoteAddrs[host]
	rl.muRemoteAddrs.RUnlock()

	if hostCountRequests > rl.maxRequests && !exists {

		go func() {
			rl.addRemoteAddrDisable(host)
			fmt.Printf("Disable host: %s - %s\n", host, time.Now().Format(time.TimeOnly))
		}()

		go func() {
			time.Sleep(rl.timeDelay)
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

func ConfigureRateLimiter(maxRequests int, timeDelay time.Duration) {
	remoteAddrMaxRequests = maxRequests
	remoteAddrTimeDelay = timeDelay
}
