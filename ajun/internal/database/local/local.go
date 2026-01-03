package local

import (
	"context"
	"log"
	"sync"
	"time"
)

type DataSource struct {
	mu           sync.RWMutex
	timeCleanIn  time.Duration
	ttl          time.Duration
	ClientIPData map[string]*ClientIPData
}

type ClientIPData struct {
	Count        int
	Time         time.Time
	DisableUntil time.Time
}

func InitDataSource(ctx context.Context, timeCleanIn time.Duration, ttl time.Duration) *DataSource {
	dataSource := &DataSource{
		timeCleanIn:  timeCleanIn,
		ttl:          ttl,
		ClientIPData: make(map[string]*ClientIPData),
	}

	go dataSource.StartCleanupWorker(ctx, timeCleanIn, ttl)

	return dataSource
}

func (ds *DataSource) AddClientIP(clientIP string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	data, exists := ds.ClientIPData[clientIP]

	count := 0
	timeDisableUntil := time.Time{}
	if exists {
		count = data.Count
		timeDisableUntil = data.DisableUntil
	}

	clientData := &ClientIPData{
		Count:        count + 1,
		Time:         time.Now(),
		DisableUntil: timeDisableUntil,
	}

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

func (ds *DataSource) StartCleanupWorker(ctx context.Context, interval time.Duration, ttl time.Duration) {
	// interval := 30 * time.Second
	// ttl := 45 * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ds.cleanupOldData(ttl)
		case <-ctx.Done():
			log.Println("Cleanup worker stopped")
			return
		}
	}
}

func (ds *DataSource) cleanupOldData(ttl time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	count := 0

	for ip, data := range ds.ClientIPData {
		if data.DisableUntil.Before(now) && now.Sub(data.Time) > ttl {
			delete(ds.ClientIPData, ip)
			log.Printf("Cleaned up data for IP: %s\n", ip)
			count++
		}
	}

	if count > 0 {
		log.Printf("Cleanup complete. Removed %d old entries.\n", count)
	}
}
