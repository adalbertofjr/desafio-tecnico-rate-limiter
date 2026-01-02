package ajun

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowsRequestsBelowLimit(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 5
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 5 requisições (no limite de 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d: esperado status 200, recebeu %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksRequestsAboveLimit(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 5
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 6 requisições (acima do limite de 5)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.Handler.ServeHTTP(w, req)

		if i < 5 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d: esperado status 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d: esperado status 429, recebeu %d", i+1, w.Code)
			}

			body := w.Body.String()
			if body != "Too many requests" {
				t.Errorf("Esperado mensagem 'Too many requests', recebeu '%s'", body)
			}
		}
	}
}

func TestRateLimiter_UnblocksAfterTimeout(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 3
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 4 requisições para bloquear
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.Handler.ServeHTTP(w, req)
	}

	// Verificar que está bloqueado
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Esperado bloqueio (429), recebeu %d", w.Code)
	}

	// Aguardar timeout
	t.Log("Aguardando 1.5 segundos para desbloqueio...")
	time.Sleep(1500 * time.Millisecond)

	// Verificar que foi desbloqueado
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	router.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Após timeout: esperado status 200, recebeu %d", w.Code)
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 3
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// IP 1: fazer 4 requisições para bloquear
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.Handler.ServeHTTP(w, req)
	}

	// Verificar que IP 1 está bloqueado
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1: esperado bloqueio (429), recebeu %d", w1.Code)
	}

	// IP 2: fazer requisição (deve passar)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	w2 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP 2: esperado status 200, recebeu %d", w2.Code)
	}
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 5
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	var wg sync.WaitGroup
	numRequests := 10
	results := make([]int, numRequests)

	// Fazer 10 requisições concorrentes do mesmo IP
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.Handler.ServeHTTP(w, req)
			results[index] = w.Code
		}(i)
	}

	wg.Wait()

	// Contar quantas requisições passaram (200) e quantas foram bloqueadas (429)
	okCount := 0
	blockedCount := 0

	for _, code := range results {
		if code == http.StatusOK {
			okCount++
		} else if code == http.StatusTooManyRequests {
			blockedCount++
		}
	}

	t.Logf("Requisições bem-sucedidas: %d, Bloqueadas: %d", okCount, blockedCount)

	// Deve ter algumas bloqueadas (pelo menos 4, já que limite é 5)
	if blockedCount < 4 {
		t.Errorf("Esperado pelo menos 4 requisições bloqueadas em teste concorrente, recebeu %d", blockedCount)
	}
}

func TestRateLimiter_HandlesIPv4AndIPv6(t *testing.T) {
	// Aguardar goroutines de testes anteriores finalizarem completamente
	time.Sleep(2500 * time.Millisecond)

	// Limpar completamente o estado antes do teste
	remoteAddrs = make(map[string]int)
	remoteAddrsDisable = make(map[string]time.Time)
	remoteAddrMaxRequests = 5
	remoteAddrTimeDelay = time.Second * 1

	// Cleanup após o teste
	defer func() {
		remoteAddrs = make(map[string]int)
		remoteAddrsDisable = make(map[string]time.Time)
	}()

	router := NewRouter()
	router.RateLimiter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/test", handler)

	// Testar IPv4
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req4.RemoteAddr = "192.168.1.1:8080"
	w4 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Errorf("IPv4: esperado status 200, recebeu %d", w4.Code)
	}

	// Testar IPv6
	req6 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req6.RemoteAddr = "[::1]:8080"
	w6 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w6, req6)

	if w6.Code != http.StatusOK {
		t.Errorf("IPv6: esperado status 200, recebeu %d", w6.Code)
	}
}
