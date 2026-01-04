package ratelimiter

import (
	"context"
	"log"
	"sync"
	"time"
)

type Storage struct {
	mu          sync.RWMutex
	backend     Backend
	timeCleanIn time.Duration
	ttl         time.Duration
	clients     map[string]*ClientIPData
}

func NewStorage(ctx context.Context, backend Backend, timeCleanIn time.Duration, ttl time.Duration) *Storage {
	s := &Storage{
		backend:     backend,
		timeCleanIn: timeCleanIn,
		ttl:         ttl,
		clients:     make(map[string]*ClientIPData),
	}

	go s.StartCleanupWorker(ctx)

	return s
}

func (s *Storage) AddClientIP(clientIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.clients[clientIP]
	if !exists {
		data = &ClientIPData{}
		s.clients[clientIP] = data
	}
	data.Count++
	data.Time = time.Now()
	s.backend.Set(clientIP, data)
}

func (s *Storage) DisableClientIP(clientIP string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.clients[clientIP]
	if !exists {
		data = &ClientIPData{Time: time.Now()}
		s.clients[clientIP] = data
	}

	data.DisableUntil = time.Now().Add(duration)
	s.backend.Set(clientIP, data)
}

func (s *Storage) GetTimeDisabledClientIP(clientIP string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// data, exists := s.clients[clientIP]
	data, err := s.backend.Get(clientIP)
	if err != nil {
		return time.Time{}, false
	}
	return data.DisableUntil, true
}

func (s *Storage) GetClientIPCount(clientIP string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.backend.Get(clientIP)
	if err != nil {
		return 0
	}

	return data.Count
}

func (s *Storage) ListClientIPs() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clientIPs := make(map[string]int)
	for ip, data := range s.clients {
		clientIPs[ip] = data.Count
	}
	return clientIPs
}

func (s *Storage) ResetClientIP(clientIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// delete(s.clients, clientIP)
	delete(s.clients, clientIP)
	s.backend.Delete(clientIP)
}

func (s *Storage) ResetDataClientIPs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients = make(map[string]*ClientIPData)
	s.backend.Clear()
}

func (s *Storage) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(s.timeCleanIn)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupOldData(s.ttl)
		case <-ctx.Done():
			log.Println("Cleanup worker stopped")
			return
		}
	}
}

func (s *Storage) cleanupOldData(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0

	for ip, data := range s.clients {
		if data.DisableUntil.Before(now) && now.Sub(data.Time) > ttl {
			delete(s.clients, ip)
			count++
		}
	}

	if count > 0 {
		log.Printf("Cleanup complete. Removed %d old entries.\n", count)
	}
}
