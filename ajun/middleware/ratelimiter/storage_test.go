package ratelimiter

import (
	"context"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		backend     StorageBackend
		timeCleanIn time.Duration
		ttl         time.Duration
	}{
		{
			name:        "Create storage with Memory backend",
			backend:     Memory,
			timeCleanIn: 1 * time.Minute,
			ttl:         5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			storage := NewStorage(ctx, tt.backend, "", tt.timeCleanIn, tt.ttl)
			if storage == nil {
				t.Fatal("Expected storage to be created")
			}
			if storage.backend == nil {
				t.Error("Expected backend to be initialized")
			}
			if storage.timeCleanIn != tt.timeCleanIn {
				t.Errorf("Expected timeCleanIn = %v, got %v", tt.timeCleanIn, storage.timeCleanIn)
			}
			if storage.ttl != tt.ttl {
				t.Errorf("Expected ttl = %v, got %v", tt.ttl, storage.ttl)
			}
		})
	}
}

func TestListClientIPs(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	storage := NewStorage(ctx, Memory, "", 1*time.Minute, 5*time.Minute)

	t.Run("Empty list", func(t *testing.T) {
		clientIPs := storage.ListClientIPs()
		if clientIPs == nil {
			t.Fatal("Expected empty map, got nil")
		}
		if len(clientIPs) != 0 {
			t.Errorf("Expected 0 clients, got %d", len(clientIPs))
		}
	})

	t.Run("List with multiple clients", func(t *testing.T) {
		storage.AddClientIP("192.168.1.1")
		storage.AddClientIP("192.168.1.1")
		storage.AddClientIP("192.168.1.2")
		storage.AddClientIP("10.0.0.1")
		storage.AddClientIP("10.0.0.1")
		storage.AddClientIP("10.0.0.1")

		clientIPs := storage.ListClientIPs()
		if len(clientIPs) != 3 {
			t.Errorf("Expected 3 clients, got %d", len(clientIPs))
		}

		if count, ok := clientIPs["192.168.1.1"]; !ok || count != 2 {
			t.Errorf("Expected count=2 for 192.168.1.1, got %d", count)
		}
		if count, ok := clientIPs["192.168.1.2"]; !ok || count != 1 {
			t.Errorf("Expected count=1 for 192.168.1.2, got %d", count)
		}
		if count, ok := clientIPs["10.0.0.1"]; !ok || count != 3 {
			t.Errorf("Expected count=3 for 10.0.0.1, got %d", count)
		}
	})
}

func TestCleanupOldData(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ttl := 100 * time.Millisecond
	storage := NewStorage(ctx, Memory, "", 10*time.Second, ttl)

	t.Run("Remove expired data", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Adiciona um IP
		storage.AddClientIP("192.168.1.1")

		// Verifica que o IP está lá
		count := storage.GetClientIPCount("192.168.1.1")
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}

		// Aguarda TTL expirar
		time.Sleep(ttl + 50*time.Millisecond)

		// Executa cleanup
		storage.cleanupOldData(ttl)

		// Verifica que o IP foi removido
		count = storage.GetClientIPCount("192.168.1.1")
		if count != 0 {
			t.Errorf("Expected count=0 after cleanup, got %d", count)
		}
	})

	t.Run("Preserve non-expired data", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Adiciona IP recente
		storage.AddClientIP("10.0.0.1")

		// Executa cleanup imediatamente
		storage.cleanupOldData(ttl)

		// Verifica que o IP ainda está lá
		count := storage.GetClientIPCount("10.0.0.1")
		if count != 1 {
			t.Errorf("Expected count=1 for non-expired IP, got %d", count)
		}
	})

	t.Run("Keep blocked IPs even if old", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Adiciona e bloqueia IP
		storage.AddClientIP("172.16.0.1")
		storage.DisableClientIP("172.16.0.1", 1*time.Hour)

		// Aguarda TTL expirar
		time.Sleep(ttl + 50*time.Millisecond)

		// Executa cleanup
		storage.cleanupOldData(ttl)

		// Verifica que o IP bloqueado ainda está lá
		_, isDisabled := storage.GetTimeDisabledClientIP("172.16.0.1")
		if !isDisabled {
			t.Error("Expected blocked IP to be preserved during cleanup")
		}
	})

	t.Run("Remove unblocked expired IPs", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Adiciona e bloqueia IP temporariamente
		storage.AddClientIP("192.168.2.1")
		storage.DisableClientIP("192.168.2.1", 10*time.Millisecond)

		// Aguarda bloqueio expirar e TTL expirar
		time.Sleep(ttl + 50*time.Millisecond)

		// Executa cleanup
		storage.cleanupOldData(ttl)

		// Verifica que o IP foi removido
		count := storage.GetClientIPCount("192.168.2.1")
		if count != 0 {
			t.Errorf("Expected count=0 for unblocked expired IP, got %d", count)
		}
	})
}

func TestStartCleanupWorker(t *testing.T) {
	t.Run("Cleanup worker stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		ttl := 50 * time.Millisecond
		cleanupInterval := 30 * time.Millisecond
		storage := NewStorage(ctx, Memory, "", cleanupInterval, ttl)

		// Adiciona IP
		storage.AddClientIP("192.168.1.1")

		// Aguarda um pouco
		time.Sleep(100 * time.Millisecond)

		// Cancela o contexto
		cancel()

		// Aguarda para garantir que o worker parou
		time.Sleep(100 * time.Millisecond)

		// Não há como verificar diretamente se parou, mas não deve causar panic
		// Se chegou até aqui sem panic, o teste passou
	})

	t.Run("Cleanup worker runs periodically", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ttl := 50 * time.Millisecond
		cleanupInterval := 60 * time.Millisecond
		storage := NewStorage(ctx, Memory, "", cleanupInterval, ttl)

		// Adiciona IP
		storage.AddClientIP("192.168.1.100")

		// Verifica que existe
		count := storage.GetClientIPCount("192.168.1.100")
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}

		// Aguarda TTL + cleanup interval
		time.Sleep(ttl + cleanupInterval + 50*time.Millisecond)

		// Verifica que foi removido pelo worker
		count = storage.GetClientIPCount("192.168.1.100")
		if count != 0 {
			t.Errorf("Expected count=0 after automatic cleanup, got %d", count)
		}
	})
}

func TestDisableClientIP_EdgeCases(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	storage := NewStorage(ctx, Memory, "", 1*time.Minute, 5*time.Minute)

	t.Run("Disable non-existent client", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Bloqueia IP que não existe
		storage.DisableClientIP("192.168.99.99", 1*time.Hour)

		// Verifica que foi criado e bloqueado
		disableUntil, isDisabled := storage.GetTimeDisabledClientIP("192.168.99.99")
		if !isDisabled {
			t.Error("Expected IP to be disabled")
		}
		if disableUntil.IsZero() {
			t.Error("Expected disableUntil to be set")
		}
	})

	t.Run("Disable existing client", func(t *testing.T) {
		storage.ResetDataClientIPs()

		// Adiciona IP primeiro
		storage.AddClientIP("10.0.0.5")
		storage.AddClientIP("10.0.0.5")

		// Bloqueia
		storage.DisableClientIP("10.0.0.5", 30*time.Minute)

		// Verifica bloqueio
		disableUntil, isDisabled := storage.GetTimeDisabledClientIP("10.0.0.5")
		if !isDisabled {
			t.Error("Expected IP to be disabled")
		}
		if time.Until(disableUntil) < 29*time.Minute {
			t.Error("Expected disable duration to be ~30 minutes")
		}

		// Verifica que o contador foi preservado
		count := storage.GetClientIPCount("10.0.0.5")
		if count != 2 {
			t.Errorf("Expected count=2 after disable, got %d", count)
		}
	})
}

func TestIncrementAndGetCount_Concurrency(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	storage := NewStorage(ctx, Memory, "", 1*time.Minute, 5*time.Minute)
	storage.ResetDataClientIPs()

	clientIP := "192.168.1.50"
	iterations := 100

	// Executa incrementos concorrentes
	done := make(chan bool, iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			storage.IncrementAndGetCount(clientIP)
			done <- true
		}()
	}

	// Aguarda todas as goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}

	// Verifica contagem final
	finalCount := storage.GetClientIPCount(clientIP)
	if finalCount != iterations {
		t.Errorf("Expected count=%d after concurrent increments, got %d", iterations, finalCount)
	}
}
