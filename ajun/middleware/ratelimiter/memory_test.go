package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestNewMemoryBackend(t *testing.T) {
	backend := NewMemoryBackend()

	if backend == nil {
		t.Fatal("NewMemoryBackend() retornou nil")
	}

	if backend.data == nil {
		t.Error("data map não foi inicializado")
	}

	if len(backend.data) != 0 {
		t.Errorf("esperado data vazio, got %d items", len(backend.data))
	}
}

func TestMemoryBackend_Set(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"
	data := &ClientIPData{
		Count: 5,
		Time:  time.Now(),
	}

	err := backend.Set(clientIP, data)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verificar se foi armazenado
	if backend.data[clientIP] == nil {
		t.Error("data não foi armazenado")
	}

	if backend.data[clientIP].Count != 5 {
		t.Errorf("esperado count = 5, got %d", backend.data[clientIP].Count)
	}
}

func TestMemoryBackend_Get(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"
	expectedTime := time.Now()
	expectedData := &ClientIPData{
		Count: 10,
		Time:  expectedTime,
	}

	// Armazenar dados primeiro
	backend.data[clientIP] = expectedData

	// Recuperar dados
	result, err := backend.Get(clientIP)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result == nil {
		t.Fatal("Get() retornou nil")
	}

	if result.Count != 10 {
		t.Errorf("esperado count = 10, got %d", result.Count)
	}

	if !result.Time.Equal(expectedTime) {
		t.Errorf("time não corresponde")
	}
}

func TestMemoryBackend_Get_NotFound(t *testing.T) {
	backend := NewMemoryBackend()

	result, err := backend.Get("ip-inexistente")
	if err != ErrNotFound {
		t.Fatalf("esperado ErrNotFound, got %v", err)
	}

	if result != nil {
		t.Error("esperado nil para IP não encontrado")
	}
}

func TestMemoryBackend_Delete(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"

	// Adicionar dados
	backend.data[clientIP] = &ClientIPData{Count: 5}

	// Verificar que existe
	if backend.data[clientIP] == nil {
		t.Fatal("setup: data não foi armazenado")
	}

	// Deletar
	err := backend.Delete(clientIP)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verificar que foi removido
	if backend.data[clientIP] != nil {
		t.Error("data não foi deletado")
	}
}

func TestMemoryBackend_Delete_NotFound(t *testing.T) {
	backend := NewMemoryBackend()

	err := backend.Delete("ip-inexistente")
	if err != nil {
		t.Errorf("Delete() error = %v, esperado nil", err)
	}
}

func TestMemoryBackend_List(t *testing.T) {
	backend := NewMemoryBackend()

	// Adicionar múltiplos IPs
	ips := map[string]*ClientIPData{
		"192.168.1.1": {Count: 5, Time: time.Now()},
		"192.168.1.2": {Count: 10, Time: time.Now()},
		"192.168.1.3": {Count: 3, Time: time.Now()},
	}

	for ip, data := range ips {
		backend.data[ip] = data
	}

	// Listar todos
	result, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("esperado 3 itens, got %d", len(result))
	}

	// Verificar que os dados estão corretos
	for ip, expectedData := range ips {
		resultData, exists := result[ip]
		if !exists {
			t.Errorf("IP %s não encontrado no resultado", ip)
			continue
		}

		if resultData.Count != expectedData.Count {
			t.Errorf("IP %s: esperado count %d, got %d", ip, expectedData.Count, resultData.Count)
		}
	}
}

func TestMemoryBackend_List_Empty(t *testing.T) {
	backend := NewMemoryBackend()

	result, err := backend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("esperado lista vazia, got %d itens", len(result))
	}
}

func TestMemoryBackend_List_IsolatedCopy(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"

	backend.data[clientIP] = &ClientIPData{Count: 5}

	// Obter lista
	result, _ := backend.List()

	// Modificar a cópia não deve afetar o original
	result[clientIP].Count = 999

	if backend.data[clientIP].Count != 5 {
		t.Error("modificação na lista retornada afetou o backend original - deveria retornar cópia")
	}
}

func TestMemoryBackend_ConcurrentAccess(t *testing.T) {
	backend := NewMemoryBackend()
	var wg sync.WaitGroup
	iterations := 100

	// Escritas concorrentes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientIP := "192.168.1." + string(rune(idx%256))
			data := &ClientIPData{Count: idx}
			backend.Set(clientIP, data)
		}(i)
	}

	// Leituras concorrentes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientIP := "192.168.1." + string(rune(idx%256))
			backend.Get(clientIP)
		}(i)
	}

	// Deletar concorrentemente
	for i := 0; i < iterations/2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientIP := "192.168.1." + string(rune(idx%256))
			backend.Delete(clientIP)
		}(i)
	}

	wg.Wait()
	// Se chegou aqui sem race condition, o teste passou
}

func TestMemoryBackend_SetUpdate(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"

	// Primeiro Set
	data1 := &ClientIPData{Count: 5}
	backend.Set(clientIP, data1)

	// Segundo Set (atualização)
	data2 := &ClientIPData{Count: 10}
	backend.Set(clientIP, data2)

	result, _ := backend.Get(clientIP)
	if result.Count != 10 {
		t.Errorf("esperado count atualizado = 10, got %d", result.Count)
	}
}

func TestMemoryBackend_DisableUntil(t *testing.T) {
	backend := NewMemoryBackend()
	clientIP := "192.168.1.1"
	disableTime := time.Now().Add(1 * time.Hour)

	data := &ClientIPData{
		Count:        5,
		Time:         time.Now(),
		DisableUntil: disableTime,
	}

	backend.Set(clientIP, data)

	result, _ := backend.Get(clientIP)
	if !result.DisableUntil.Equal(disableTime) {
		t.Error("disableUntil não foi armazenado corretamente")
	}
}

func TestMemoryBackend_ImplementsInterface(t *testing.T) {
	var _ Backend = (*MemoryBackend)(nil)
}

func BenchmarkMemoryBackend_Set(b *testing.B) {
	backend := NewMemoryBackend()
	data := &ClientIPData{Count: 1, Time: time.Now()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.Set("192.168.1.1", data)
	}
}

func BenchmarkMemoryBackend_Get(b *testing.B) {
	backend := NewMemoryBackend()
	backend.Set("192.168.1.1", &ClientIPData{Count: 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.Get("192.168.1.1")
	}
}

func BenchmarkMemoryBackend_ConcurrentSet(b *testing.B) {
	backend := NewMemoryBackend()
	data := &ClientIPData{Count: 1, Time: time.Now()}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			backend.Set("192.168.1.1", data)
			i++
		}
	})
}

func BenchmarkMemoryBackend_ConcurrentGet(b *testing.B) {
	backend := NewMemoryBackend()
	backend.Set("192.168.1.1", &ClientIPData{Count: 1})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			backend.Get("192.168.1.1")
		}
	})
}
