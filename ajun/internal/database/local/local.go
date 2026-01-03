package local

import (
	"sync"
	"time"
)

type DataSource struct {
	mu           sync.RWMutex
	ClientIPData map[string]*ClientIPData
}

type ClientIPData struct {
	Count        int
	Time         time.Time
	DisableUntil time.Time
}

func InitDataSource() *DataSource {
	return &DataSource{
		ClientIPData: make(map[string]*ClientIPData),
	}
}

func (ds *DataSource) AddClientIP(clientIP string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	count := 0
	if data, exists := ds.ClientIPData[clientIP]; exists {
		count = data.Count
	}

	timeDisableUntil := time.Time{}
	if data, exists := ds.ClientIPData[clientIP]; exists {
		timeDisableUntil = data.DisableUntil
	}

	clientData := &ClientIPData{
		Count:        count + 1,
		Time:         time.Now(),
		DisableUntil: timeDisableUntil,
	}

	// todo: verificar se existe e inicializar se necessário
	ds.ClientIPData[clientIP] = clientData
}

func (ds *DataSource) DisableClientIP(clientIP string, duration time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if data, exists := ds.ClientIPData[clientIP]; exists {
		data.DisableUntil = time.Now().Add(duration)
	} else {
		ds.ClientIPData[clientIP] = &ClientIPData{
			Count:        0,
			Time:         time.Now(),
			DisableUntil: time.Now().Add(duration),
		}
	}
}

func (ds *DataSource) GetTimeDisabledClientIP(clientIP string) (time.Time, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	data, exists := ds.ClientIPData[clientIP]
	if !exists {
		return time.Time{}, false
	}
	return data.DisableUntil, true
}

func (ds *DataSource) GetClientIPCount(clientIP string) int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if data, exists := ds.ClientIPData[clientIP]; exists {
		return data.Count
	}
	return 0
}

func (ds *DataSource) ListClientIPs() map[string]int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	clientIPs := make(map[string]int)
	for ip, data := range ds.ClientIPData {
		clientIPs[ip] = data.Count
	}
	return clientIPs
}

func (ds *DataSource) ResetClientIP(clientIP string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	delete(ds.ClientIPData, clientIP)
}

func (ds *DataSource) ResetDataClientIPs() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.ClientIPData = make(map[string]*ClientIPData)
}
