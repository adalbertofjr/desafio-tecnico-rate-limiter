package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	maxRequests := 10
	timeDelay := time.Second * 5

	rl := NewRateLimiter(maxRequests, timeDelay)

	if rl == nil {
		t.Fatal("NewRateLimiter retornou nil")
	}

	if rl.maxRequests != maxRequests {
		t.Errorf("maxRequests: esperado %d, recebeu %d", maxRequests, rl.maxRequests)
	}

	if rl.timeDelay != timeDelay {
		t.Errorf("timeDelay: esperado %v, recebeu %v", timeDelay, rl.timeDelay)
	}

	if rl.muRemoteAddrs == nil {
		t.Error("muRemoteAddrs não deve ser nil")
	}

	if rl.muRemoteAddrsDisable == nil {
		t.Error("muRemoteAddrsDisable não deve ser nil")
	}
}

func TestRateLimiterHandler_AllowsRequestsBelowLimit(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(5, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 5 requisições (no limite)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d: esperado 200, recebeu %d", i+1, w.Code)
		}

		if w.Body.String() != "success" {
			t.Errorf("Requisição %d: body incorreto", i+1)
		}
	}
}

func TestRateLimiterHandler_BlocksRequestsAboveLimit(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(3, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 4 requisições (acima do limite de 3)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if i < 3 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d: esperado 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d: esperado 429, recebeu %d", i+1, w.Code)
			}

			if w.Body.String() != "Too many requests" {
				t.Errorf("Requisição %d: mensagem incorreta: %s", i+1, w.Body.String())
			}
		}
	}
}

func TestRateLimiterHandler_ResetsAfterTimeout(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(2, time.Millisecond*500)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 3 requisições (acima do limite de 2)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.3:12345"
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verificar que está bloqueado
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Esperado bloqueio (429), recebeu %d", w.Code)
	}

	// Aguardar timeout + margem
	time.Sleep(600 * time.Millisecond)

	// Verificar que foi desbloqueado
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Após timeout: esperado 200, recebeu %d", w.Code)
	}
}

func TestRateLimiterHandler_DifferentIPsIndependent(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(2, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// IP 1: fazer 3 requisições (bloquear)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.4:12345"
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verificar que IP 1 está bloqueado
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "10.0.0.4:12345"
	w1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1: esperado bloqueio (429), recebeu %d", w1.Code)
	}

	// IP 2: fazer requisição (deve passar)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.5:12345"
	w2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP 2: esperado 200, recebeu %d", w2.Code)
	}
}

func TestRateLimiterHandler_HandlesIPv6(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(5, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Testar IPv6
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:8080"
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("IPv6: esperado 200, recebeu %d", w.Code)
	}
}

func TestRateLimiterHandler_HandlesInvalidIP(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(5, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// IP sem porta (fallback para split por :)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100"
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("IP sem porta: esperado 200, recebeu %d", w.Code)
	}
}

func TestRateLimiterHandler_ConcurrentRequests(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(10, time.Second*2)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	var wg sync.WaitGroup
	numRequests := 20
	results := make([]int, numRequests)

	// Fazer 20 requisições concorrentes do mesmo IP
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "10.0.0.99:12345"
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			results[index] = w.Code
		}(i)
	}

	wg.Wait()

	// Contar sucessos e bloqueios
	okCount := 0
	blockedCount := 0

	for _, code := range results {
		if code == http.StatusOK {
			okCount++
		} else if code == http.StatusTooManyRequests {
			blockedCount++
		}
	}

	// Deve ter bloqueado algumas (pelo menos 9, já que limite é 10)
	if blockedCount < 9 {
		t.Errorf("Esperado pelo menos 9 bloqueios, recebeu %d", blockedCount)
	}

	if okCount == 0 {
		t.Error("Nenhuma requisição passou, esperado pelo menos algumas")
	}
}

func TestResetGlobalState(t *testing.T) {
	// Adicionar dados ao estado global
	remoteAddrs["test-ip"] = 5
	remoteAddrsDisable["test-ip"] = time.Now().Add(time.Hour)

	// Resetar
	ResetGlobalState()

	// Verificar que foi limpo
	if len(remoteAddrs) != 0 {
		t.Errorf("remoteAddrs não foi limpo: %d itens", len(remoteAddrs))
	}

	if len(remoteAddrsDisable) != 0 {
		t.Errorf("remoteAddrsDisable não foi limpo: %d itens", len(remoteAddrsDisable))
	}
}

func TestRateLimiter_AddRemoteAddr(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(5, time.Second)

	// Adicionar mesmo IP 3 vezes
	for i := 0; i < 3; i++ {
		rl.addRemoteAddr("192.168.1.1")
	}

	rl.muRemoteAddrs.RLock()
	count := remoteAddrs["192.168.1.1"]
	rl.muRemoteAddrs.RUnlock()

	if count != 3 {
		t.Errorf("Esperado count 3, recebeu %d", count)
	}
}

func TestRateLimiter_IsRemoteAddrDisabled(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	rl := NewRateLimiter(2, time.Millisecond*100)

	// Não está desabilitado inicialmente
	if rl.isRemoteAddrDisabled("192.168.1.1") {
		t.Error("IP não deveria estar desabilitado inicialmente")
	}

	// Adicionar requisições acima do limite
	for i := 0; i < 3; i++ {
		rl.addRemoteAddr("192.168.1.1")
	}

	// Agora deve estar desabilitado
	if !rl.isRemoteAddrDisabled("192.168.1.1") {
		t.Error("IP deveria estar desabilitado após exceder limite")
	}

	// Aguardar reset
	time.Sleep(150 * time.Millisecond)

	// Deve estar habilitado novamente
	if rl.isRemoteAddrDisabled("192.168.1.1") {
		t.Error("IP deveria estar habilitado após timeout")
	}
}

func TestRateLimiter_MultipleInstances(t *testing.T) {
	ResetGlobalState()
	defer ResetGlobalState()

	// Criar duas instâncias com configurações diferentes
	rl1 := NewRateLimiter(5, time.Second)
	rl2 := NewRateLimiter(10, time.Second*2)

	if rl1.maxRequests == rl2.maxRequests {
		t.Error("Instâncias deveriam ter configurações diferentes")
	}

	if rl1.timeDelay == rl2.timeDelay {
		t.Error("Instâncias deveriam ter timeDelay diferentes")
	}

	// Nota: ambas compartilham estado global (limitação conhecida)
}
