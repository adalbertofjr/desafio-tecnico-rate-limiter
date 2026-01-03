package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	maxRequests := 10
	timeDelay := time.Second * 5

	config := NewRateLimiterConfig(maxRequests, timeDelay, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)

	if rl == nil {
		t.Fatal("NewRateLimiter retornou nil")
	}

	if rl.config.Limit != maxRequests {
		t.Errorf("Limit: esperado %d, recebeu %d", maxRequests, rl.config.Limit)
	}

	if rl.config.Delay != timeDelay {
		t.Errorf("Delay: esperado %v, recebeu %v", timeDelay, rl.config.Delay)
	}

	if rl.datasource == nil {
		t.Error("datasource não deve ser nil")
	}
}

func TestRateLimiterHandler_AllowsRequestsBelowLimit(t *testing.T) {
	config := NewRateLimiterConfig(5, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(3, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(2, time.Millisecond*500, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(2, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(5, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(5, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(10, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

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
	config := NewRateLimiterConfig(5, time.Second, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)

	// Adicionar dados
	rl.datasource.AddClientIP("test-ip")
	rl.datasource.AddClientIP("test-ip")
	rl.datasource.AddClientIP("test-ip")

	// Resetar
	rl.ResetGlobalState()

	// Verificar que foi limpo
	count := rl.datasource.GetClientIPCount("test-ip")
	if count != 0 {
		t.Errorf("ClientIP não foi limpo: count %d", count)
	}
}

func TestRateLimiter_AddRemoteAddr(t *testing.T) {
	config := NewRateLimiterConfig(5, time.Second, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	// Adicionar mesmo IP 3 vezes
	for i := 0; i < 3; i++ {
		rl.datasource.AddClientIP("192.168.1.1")
	}

	count := rl.datasource.GetClientIPCount("192.168.1.1")

	if count != 3 {
		t.Errorf("Esperado count 3, recebeu %d", count)
	}
}

func TestRateLimiter_IsRemoteAddrDisabled(t *testing.T) {
	config := NewRateLimiterConfig(2, time.Millisecond*100, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	// Não está desabilitado inicialmente
	if rl.isRemoteAddrDisabled("192.168.1.1", "") {
		t.Error("IP não deveria estar desabilitado inicialmente")
	}

	// Adicionar requisições acima do limite
	for i := 0; i < 3; i++ {
		rl.datasource.AddClientIP("192.168.1.1")
	}

	// Agora deve estar desabilitado
	if !rl.isRemoteAddrDisabled("192.168.1.1", "") {
		t.Error("IP deveria estar desabilitado após exceder limite")
	}

	// Aguardar reset
	time.Sleep(150 * time.Millisecond)

	// Deve estar habilitado novamente
	if rl.isRemoteAddrDisabled("192.168.1.1", "") {
		t.Error("IP deveria estar habilitado após timeout")
	}
}

func TestRateLimiter_MultipleInstances(t *testing.T) {
	// Criar duas instâncias com configurações diferentes
	config1 := NewRateLimiterConfig(5, time.Second, 0, 0, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl1 := NewRateLimiter(ctx, config1)
	defer rl1.ResetGlobalState()

	config2 := NewRateLimiterConfig(10, time.Second*2, 0, 0, 30*time.Second, 45*time.Second)
	rl2 := NewRateLimiter(ctx, config2)

	if rl1.config.Limit == rl2.config.Limit {
		t.Error("Instâncias deveriam ter configurações diferentes")
	}

	if rl1.config.Delay == rl2.config.Delay {
		t.Error("Instâncias deveriam ter Delay diferentes")
	}

	// Cada instância agora tem seu próprio datasource
	if rl1.datasource == rl2.datasource {
		t.Error("Instâncias deveriam ter datasources diferentes")
	}
}

func TestRateLimiterHandler_WithAPIKey_AllowsRequestsBelowTokenLimit(t *testing.T) {
	// Configurar: limite IP = 2, limite Token = 10
	config := NewRateLimiterConfig(2, time.Second*2, 10, time.Second*2, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 10 requisições com API_KEY (no limite de token)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.10:12345"
		req.Header.Set("Api_key", "token123")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Requisição %d com API_KEY: esperado 200, recebeu %d", i+1, w.Code)
		}

		if w.Body.String() != "success" {
			t.Errorf("Requisição %d: body incorreto", i+1)
		}
	}
}

func TestRateLimiterHandler_WithAPIKey_BlocksRequestsAboveTokenLimit(t *testing.T) {
	// Configurar: limite IP = 2, limite Token = 5
	config := NewRateLimiterConfig(2, time.Second*2, 5, time.Second*2, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 7 requisições com API_KEY (acima do limite token = 5)
	for i := 0; i < 7; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.11:12345"
		req.Header.Set("Api_key", "token456")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if i < 5 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d com API_KEY: esperado 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d com API_KEY: esperado 429, recebeu %d", i+1, w.Code)
			}

			if w.Body.String() != "Too many requests" {
				t.Errorf("Requisição %d: mensagem incorreta: %s", i+1, w.Body.String())
			}
		}
	}
}

func TestRateLimiterHandler_WithAPIKey_HasDifferentLimitThanIP(t *testing.T) {
	// Aguardar goroutines de testes anteriores
	time.Sleep(100 * time.Millisecond)
	// Configurar: limite IP = 3, limite Token = 10
	config := NewRateLimiterConfig(3, time.Second*2, 10, time.Second*2, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 4 requisições SEM API_KEY (acima do limite IP = 3)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.12:12345"
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if i < 3 {
			if w.Code != http.StatusOK {
				t.Errorf("Requisição %d sem API_KEY: esperado 200, recebeu %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Requisição %d sem API_KEY: esperado 429, recebeu %d", i+1, w.Code)
			}
		}
	}
}

func TestRateLimiterHandler_WithAPIKey_UnblocksAfterTokenTimeout(t *testing.T) {
	// Criar rate limiter com timeout curto para token
	config := NewRateLimiterConfig(2, time.Second*2, 3, time.Millisecond*500, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// Fazer 4 requisições com API_KEY para bloquear (limite token é 3)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.13:12345"
		req.Header.Set("Api_key", "token789")
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verificar que está bloqueado
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.13:12345"
	req.Header.Set("Api_key", "token789")
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Esperado bloqueio (429), recebeu %d", w.Code)
	}

	// Aguardar timeout + margem
	time.Sleep(600 * time.Millisecond)

	// Verificar que foi desbloqueado
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.13:12345"
	req.Header.Set("Api_key", "token789")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Com API_KEY após timeout: esperado 200, recebeu %d", w.Code)
	}
}

func TestRateLimiterHandler_WithAPIKey_DifferentIPsSameKey(t *testing.T) {
	// Configurar: limite IP = 2, limite Token = 5
	config := NewRateLimiterConfig(2, time.Second*2, 5, time.Second*2, 30*time.Second, 45*time.Second)
	ctx := context.Background()
	rl := NewRateLimiter(ctx, config)
	defer rl.ResetGlobalState()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := rl.RateLimiterHandler(handler)

	// IP 1: fazer 6 requisições com API_KEY (acima do limite token = 5)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.14:12345"
		req.Header.Set("Api_key", "shared-token")
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verificar que IP 1 está bloqueado
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "10.0.0.14:12345"
	req1.Header.Set("Api_key", "shared-token")
	w1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 com API_KEY: esperado bloqueio (429), recebeu %d", w1.Code)
	}

	// IP 2 com mesma API_KEY: fazer requisição (deve passar - limites são por IP)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.15:12345"
	req2.Header.Set("Api_key", "shared-token")
	w2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP 2 com mesma API_KEY: esperado 200, recebeu %d", w2.Code)
	}
}
