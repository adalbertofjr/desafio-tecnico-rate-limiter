package local

import (
	"context"
	"log"
	"sync"
	"time"
)

type Storage interface {
	AddClientIP(clientIP string)
	DisableClientIP(clientIP string, duration time.Duration)
	GetTimeDisabledClientIP(clientIP string) (time.Time, bool)
	GetClientIPCount(clientIP string) int
	ListClientIPs() map[string]int
	ResetClientIP(clientIP string)
	ResetDataClientIPs()
}

type LocalStorage struct {
	mu          sync.RWMutex
	timeCleanIn time.Duration
	ttl         time.Duration
	clients     map[string]*ClientIPData
}

type ClientIPData struct {
	count        int
	time         time.Time
	disableUntil time.Time
}

func NewLocalStorage(ctx context.Context, timeCleanIn time.Duration, ttl time.Duration) Storage {
	dataSource := &LocalStorage{
		timeCleanIn: timeCleanIn,
		ttl:         ttl,
		clients:     make(map[string]*ClientIPData),
	}

	go dataSource.StartCleanupWorker(ctx)

	return dataSource
}

func (ds *LocalStorage) AddClientIP(clientIP string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	data, exists := ds.clients[clientIP]
	if !exists {
		data = &ClientIPData{}
		ds.clients[clientIP] = data
	}
	data.count++
	data.time = time.Now()
}

func (ds *LocalStorage) DisableClientIP(clientIP string, duration time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	data, exists := ds.clients[clientIP]
	if !exists {
		data = &ClientIPData{time: time.Now()}
		ds.clients[clientIP] = data
	}

	data.disableUntil = time.Now().Add(duration)
}

func (ds *LocalStorage) GetTimeDisabledClientIP(clientIP string) (time.Time, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	data, exists := ds.clients[clientIP]
	if !exists {
		return time.Time{}, false
	}
	return data.disableUntil, true
}

func (ds *LocalStorage) GetClientIPCount(clientIP string) int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if data, exists := ds.clients[clientIP]; exists {
		return data.count
	}
	return 0
}

func (ds *LocalStorage) ListClientIPs() map[string]int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	clientIPs := make(map[string]int)
	for ip, data := range ds.clients {
		clientIPs[ip] = data.count
	}
	return clientIPs
}

func (ds *LocalStorage) ResetClientIP(clientIP string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	delete(ds.clients, clientIP)
}

func (ds *LocalStorage) ResetDataClientIPs() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.clients = make(map[string]*ClientIPData)
}

func (ds *LocalStorage) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(ds.timeCleanIn)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ds.cleanupOldData(ds.ttl)
		case <-ctx.Done():
			log.Println("Cleanup worker stopped")
			return
		}
	}
}

func (ds *LocalStorage) cleanupOldData(ttl time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	count := 0

	for ip, data := range ds.clients {
		if data.disableUntil.Before(now) && now.Sub(data.time) > ttl {
			delete(ds.clients, ip)
			count++
		}
	}

	if count > 0 {
		log.Printf("Cleanup complete. Removed %d old entries.\n", count)
	}
}
