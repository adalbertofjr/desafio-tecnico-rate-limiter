package ajun

import (
	"adalbertofjr/desafio-rate-limiter/ajun/middleware"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowsRequestsBelowLimit(t *testing.T) {
	router := NewRouter()
	config := middleware.NewRateLimiterConfig(5, time.Second*4, 0, 0)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

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
	router := NewRouter()
	config := middleware.NewRateLimiterConfig(5, time.Second*4, 0, 0)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

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
	// Aguardar goroutines de testes anteriores
	time.Sleep(200 * time.Millisecond)

	// Criar router com timeout curto para teste
	config := middleware.NewRateLimiterConfig(3, time.Second*1, 0, 0)
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.ResetGlobalState()
	router := NewRouter()
	router.Handler = rateLimiter.RateLimiterHandler(router.router)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 4 requisições para bloquear (limite é 3)
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
	// Criar router com limite de 3 requisições
	config := middleware.NewRateLimiterConfig(3, time.Second*1, 0, 0)
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.ResetGlobalState()
	router := NewRouter()
	router.Handler = rateLimiter.RateLimiterHandler(router.router)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// IP 1: fazer 4 requisições para bloquear (limite é 3)
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
	router := NewRouter()
	config := middleware.NewRateLimiterConfig(5, time.Second*4, 0, 0)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

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
	router := NewRouter()
	config := middleware.NewRateLimiterConfig(5, time.Second*4, 0, 0)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

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

func TestRateLimiter_WithAPIKey_AllowsRequestsBelowTokenLimit(t *testing.T) {
	router := NewRouter()
	// Configurar: limite IP = 2, limite Token = 10
	config := middleware.NewRateLimiterConfig(2, time.Second*4, 10, time.Second*4)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 10 requisições com API_KEY (no limite de token)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("Api_key", "abc123")
		w := httptest.NewRecorder()

		router.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d com API_KEY: esperado status 200, recebeu %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_WithAPIKey_BlocksRequestsAboveTokenLimit(t *testing.T) {
	router := NewRouter()
	// Configurar: limite IP = 2, limite Token = 5
	config := middleware.NewRateLimiterConfig(2, time.Second*4, 5, time.Second*4)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 7 requisições com API_KEY (acima do limite de token = 5)
	for i := 0; i < 7; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("Api_key", "abc123")
		w := httptest.NewRecorder()

		router.Handler.ServeHTTP(w, req)

		if i < 5 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d com API_KEY: esperado status 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d com API_KEY: esperado status 429, recebeu %d", i+1, w.Code)
			}
		}
	}
}

func TestRateLimiter_WithAPIKey_HasDifferentLimitThanIP(t *testing.T) {
	// Aguardar goroutines de testes anteriores
	time.Sleep(200 * time.Millisecond)

	router := NewRouter()
	// Configurar: limite IP = 3, limite Token = 10
	config := middleware.NewRateLimiterConfig(3, time.Second*4, 10, time.Second*4)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 4 requisições SEM API_KEY (acima do limite IP = 3)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.Handler.ServeHTTP(w, req)

		if i < 3 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d sem API_KEY: esperado status 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d sem API_KEY: esperado status 429, recebeu %d", i+1, w.Code)
			}
		}
	}
}

func TestRateLimiter_WithAPIKey_UnblocksAfterTokenTimeout(t *testing.T) {
	// Criar router com timeout curto para teste com token
	config := middleware.NewRateLimiterConfig(2, time.Second*2, 3, time.Millisecond*600)
	rateLimiter := middleware.NewRateLimiter(config)
	defer rateLimiter.ResetGlobalState()
	router := NewRouter()
	router.Handler = rateLimiter.RateLimiterHandler(router.router)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// Fazer 4 requisições com API_KEY para bloquear (limite token é 3)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("Api_key", "xyz789")
		w := httptest.NewRecorder()
		router.Handler.ServeHTTP(w, req)
	}

	// Aguardar timeout do token (600ms + margem)
	t.Log("Aguardando 800ms para desbloqueio...")
	time.Sleep(800 * time.Millisecond)

	// Verificar que foi desbloqueado
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Api_key", "xyz789")
	w := httptest.NewRecorder()
	router.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Com API_KEY após timeout: esperado status 200, recebeu %d", w.Code)
	}
}

func TestRateLimiter_WithAPIKey_DifferentIPsSameKey(t *testing.T) {
	router := NewRouter()
	// Configurar: limite IP = 2, limite Token = 5
	config := middleware.NewRateLimiterConfig(2, time.Second*4, 5, time.Second*4)
	router.RateLimiter(config)
	defer router.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.HandleFunc("/test", handler)

	// IP 1: fazer 6 requisições com API_KEY (acima do limite token = 5)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("Api_key", "shared-key")
		w := httptest.NewRecorder()
		router.Handler.ServeHTTP(w, req)
	}

	// Verificar que IP 1 está bloqueado
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	req1.Header.Set("Api_key", "shared-key")
	w1 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 com API_KEY: esperado bloqueio (429), recebeu %d", w1.Code)
	}

	// IP 2 com mesma API_KEY: fazer requisição (deve passar - limites são por IP)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	req2.Header.Set("Api_key", "shared-key")
	w2 := httptest.NewRecorder()
	router.Handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP 2 com mesma API_KEY: esperado status 200, recebeu %d", w2.Code)
	}
}
