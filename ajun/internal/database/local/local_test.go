package local

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestCleanupOldData_RemovesStaleEntries(t *testing.T) {
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	// Adicionar IP com timestamp antigo
	oldTime := time.Now().Add(-2 * time.Second)
	ds.ClientIPData["192.168.1.1"] = &ClientIPData{
		Count:        5,
		Time:         oldTime,
		DisableUntil: time.Time{},
	}

	// Adicionar IP recente
	ds.ClientIPData["192.168.1.2"] = &ClientIPData{
		Count:        3,
		Time:         time.Now(),
		DisableUntil: time.Time{},
	}

	// Executar cleanup com TTL de 1 segundo
	ds.cleanupOldData(1 * time.Second)

	// Verificar que IP antigo foi removido
	if _, exists := ds.ClientIPData["192.168.1.1"]; exists {
		t.Error("IP antigo não foi removido pelo cleanup")
	}

	// Verificar que IP recente permanece
	if _, exists := ds.ClientIPData["192.168.1.2"]; !exists {
		t.Error("IP recente foi removido incorretamente")
	}
}

func TestCleanupOldData_RespectsActiveBlocks(t *testing.T) {
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	// IP antigo mas ainda bloqueado
	oldTime := time.Now().Add(-2 * time.Second)
	futureBlock := time.Now().Add(5 * time.Second)
	ds.ClientIPData["192.168.1.1"] = &ClientIPData{
		Count:        10,
		Time:         oldTime,
		DisableUntil: futureBlock,
	}

	// IP antigo com bloqueio expirado
	pastBlock := time.Now().Add(-1 * time.Second)
	ds.ClientIPData["192.168.1.2"] = &ClientIPData{
		Count:        8,
		Time:         oldTime,
		DisableUntil: pastBlock,
	}

	// Executar cleanup
	ds.cleanupOldData(1 * time.Second)

	// IP bloqueado não deve ser removido
	if _, exists := ds.ClientIPData["192.168.1.1"]; !exists {
		t.Error("IP bloqueado foi removido incorretamente")
	}

	// IP com bloqueio expirado deve ser removido
	if _, exists := ds.ClientIPData["192.168.1.2"]; exists {
		t.Error("IP com bloqueio expirado não foi removido")
	}
}

func TestCleanupWorker_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	done := make(chan bool)
	go func() {
		ds.StartCleanupWorker(ctx, 100*time.Millisecond, 1*time.Second)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Worker parou corretamente
	case <-time.After(500 * time.Millisecond):
		t.Error("Cleanup worker não parou após context cancelado")
	}
}

func TestCleanupWorker_RunsPeriodically(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	// Adicionar IP antigo
	oldTime := time.Now().Add(-2 * time.Second)
	ds.ClientIPData["192.168.1.1"] = &ClientIPData{
		Count:        5,
		Time:         oldTime,
		DisableUntil: time.Time{},
	}

	// Iniciar worker
	go ds.StartCleanupWorker(ctx, 200*time.Millisecond, 1*time.Second)

	// Aguardar pelo menos um ciclo
	time.Sleep(300 * time.Millisecond)

	// Verificar que IP foi removido
	ds.mu.RLock()
	_, exists := ds.ClientIPData["192.168.1.1"]
	ds.mu.RUnlock()

	if exists {
		t.Error("Cleanup worker não executou e removeu IP antigo")
	}
}

func TestCleanupOldData_ThreadSafe(t *testing.T) {
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	// Adicionar múltiplos IPs
	for i := 0; i < 100; i++ {
		ip := "192.168.1." + strconv.Itoa(i)
		ds.ClientIPData[ip] = &ClientIPData{
			Count:        i,
			Time:         time.Now().Add(-2 * time.Second),
			DisableUntil: time.Time{},
		}
	}

	// Executar cleanup concorrentemente com leituras
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			ds.cleanupOldData(1 * time.Second)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Ler dados concorrentemente
	for i := 0; i < 50; i++ {
		go func() {
			ds.ListClientIPs()
		}()
	}

	<-done
}

func TestCleanupOldData_EmptyMap(t *testing.T) {
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	// Executar cleanup em map vazio
	ds.cleanupOldData(1 * time.Second)

	if len(ds.ClientIPData) != 0 {
		t.Errorf("Map deveria estar vazio, mas tem %d entradas", len(ds.ClientIPData))
	}
}

func TestCleanupOldData_MultipleStaleEntries(t *testing.T) {
	ds := &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}

	oldTime := time.Now().Add(-5 * time.Second)

	// Adicionar 5 IPs antigos
	for i := 1; i <= 5; i++ {
		ip := "192.168.1." + strconv.Itoa(i)
		ds.ClientIPData[ip] = &ClientIPData{
			Count:        i,
			Time:         oldTime,
			DisableUntil: time.Time{},
		}
	}

	// Adicionar 3 IPs recentes
	for i := 6; i <= 8; i++ {
		ip := "192.168.1." + strconv.Itoa(i)
		ds.ClientIPData[ip] = &ClientIPData{
			Count:        i,
			Time:         time.Now(),
			DisableUntil: time.Time{},
		}
	}

	initialCount := len(ds.ClientIPData)
	if initialCount != 8 {
		t.Fatalf("Esperado 8 IPs iniciais, mas tem %d", initialCount)
	}

	// Executar cleanup
	ds.cleanupOldData(2 * time.Second)

	// Verificar que apenas 3 IPs permanecem
	if len(ds.ClientIPData) != 3 {
		t.Errorf("Esperado 3 IPs após cleanup, mas tem %d", len(ds.ClientIPData))
	}
}
