package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) (*RedisBackend, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Erro ao iniciar miniredis: %v", err)
	}

	ctx := context.Background()
	backend := &RedisBackend{
		ctx: ctx,
		client: redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		}),
	}

	return backend, mr
}

func TestRedisBackend_SetAndGet(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	data := &ClientIPData{
		Count:        5,
		Time:         time.Now(),
		DisableUntil: time.Now().Add(time.Minute),
	}

	// Test Set
	err := backend.Set("192.168.1.1", data)
	if err != nil {
		t.Fatalf("Set retornou erro: %v", err)
	}

	// Test Get
	retrieved, err := backend.Get("192.168.1.1")
	if err != nil {
		t.Fatalf("Get retornou erro: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Get retornou nil")
	}

	if retrieved.Count != data.Count {
		t.Errorf("Count esperado %d, obtido %d", data.Count, retrieved.Count)
	}
}

func TestRedisBackend_GetNonExistent(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	retrieved, err := backend.Get("nao-existe")
	if err == nil {
		t.Error("Get deveria retornar erro para chave inexistente")
	}

	if retrieved != nil {
		t.Error("Get deveria retornar nil para chave inexistente")
	}
}

func TestRedisBackend_Delete(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	data := &ClientIPData{
		Count: 3,
		Time:  time.Now(),
	}

	// Criar entrada
	err := backend.Set("192.168.1.2", data)
	if err != nil {
		t.Fatalf("Set retornou erro: %v", err)
	}

	// Deletar
	err = backend.Delete("192.168.1.2")
	if err != nil {
		t.Fatalf("Delete retornou erro: %v", err)
	}

	// Verificar que foi deletado
	retrieved, err := backend.Get("192.168.1.2")
	if err == nil {
		t.Error("Get deveria retornar erro após Delete")
	}

	if retrieved != nil {
		t.Error("Dado deveria estar deletado")
	}
}

func TestRedisBackend_List(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Adicionar múltiplas entradas
	data1 := &ClientIPData{Count: 1, Time: time.Now()}
	data2 := &ClientIPData{Count: 2, Time: time.Now()}
	data3 := &ClientIPData{Count: 3, Time: time.Now()}

	backend.Set("192.168.1.1", data1)
	backend.Set("192.168.1.2", data2)
	backend.Set("192.168.1.3", data3)

	// Listar
	result, err := backend.List()
	if err != nil {
		t.Fatalf("List retornou erro: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("List deveria retornar 3 entradas, obtido %d", len(result))
	}

	if result["192.168.1.1"].Count != 1 {
		t.Error("Dados incorretos para 192.168.1.1")
	}

	if result["192.168.1.2"].Count != 2 {
		t.Error("Dados incorretos para 192.168.1.2")
	}

	if result["192.168.1.3"].Count != 3 {
		t.Error("Dados incorretos para 192.168.1.3")
	}
}

func TestRedisBackend_ListEmpty(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	result, err := backend.List()
	if err != nil {
		t.Fatalf("List retornou erro: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("List deveria estar vazio, obtido %d entradas", len(result))
	}
}

func TestRedisBackend_Clear(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Adicionar dados
	data1 := &ClientIPData{Count: 1, Time: time.Now()}
	data2 := &ClientIPData{Count: 2, Time: time.Now()}

	backend.Set("192.168.1.1", data1)
	backend.Set("192.168.1.2", data2)

	// Verificar que existem
	list, _ := backend.List()
	if len(list) != 2 {
		t.Fatalf("Deveria ter 2 entradas antes do Clear")
	}

	// Limpar
	err := backend.Clear()
	if err != nil {
		t.Fatalf("Clear retornou erro: %v", err)
	}

	// Verificar que está vazio
	list, _ = backend.List()
	if len(list) != 0 {
		t.Errorf("Após Clear deveria estar vazio, obtido %d entradas", len(list))
	}
}

func TestRedisBackend_ClearEmpty(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Clear em backend vazio não deve dar erro
	err := backend.Clear()
	if err != nil {
		t.Fatalf("Clear em backend vazio retornou erro: %v", err)
	}
}

func TestRedisBackend_SetUpdateExisting(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Criar entrada inicial
	data1 := &ClientIPData{Count: 5, Time: time.Now()}
	backend.Set("192.168.1.1", data1)

	// Atualizar
	data2 := &ClientIPData{Count: 10, Time: time.Now()}
	err := backend.Set("192.168.1.1", data2)
	if err != nil {
		t.Fatalf("Set (update) retornou erro: %v", err)
	}

	// Verificar atualização
	retrieved, _ := backend.Get("192.168.1.1")
	if retrieved.Count != 10 {
		t.Errorf("Count deveria ser 10 após update, obtido %d", retrieved.Count)
	}
}

func TestRedisBackend_ConcurrentAccess(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Teste de concorrência básico
	done := make(chan bool)

	// Múltiplas goroutines escrevendo
	for i := 0; i < 10; i++ {
		go func(id int) {
			data := &ClientIPData{
				Count: id,
				Time:  time.Now(),
			}
			backend.Set("ip-"+string(rune(id)), data)
			done <- true
		}(i)
	}

	// Aguardar todas as goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verificar que todas foram escritas
	list, err := backend.List()
	if err != nil {
		t.Fatalf("List retornou erro: %v", err)
	}

	if len(list) < 10 {
		t.Errorf("Esperado pelo menos 10 entradas, obtido %d", len(list))
	}
}

func TestRedisBackend_DeleteNonExistent(t *testing.T) {
	backend, mr := setupTestRedis(t)
	defer mr.Close()

	// Delete de chave inexistente não deve dar erro
	err := backend.Delete("nao-existe")
	if err != nil {
		t.Errorf("Delete de chave inexistente não deveria dar erro: %v", err)
	}
}
