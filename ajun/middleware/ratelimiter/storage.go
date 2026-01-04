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
}

func NewStorage(ctx context.Context, backend Backend, timeCleanIn time.Duration, ttl time.Duration) *Storage {
	s := &Storage{
		backend:     backend,
		timeCleanIn: timeCleanIn,
		ttl:         ttl,
	}

	go s.StartCleanupWorker(ctx)

	return s
}

func (s *Storage) AddClientIP(clientIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.backend.Get(clientIP)
	if err != nil {
		data = &ClientIPData{}
	}
	data.Count++
	data.Time = time.Now()

	s.backend.Set(clientIP, data)
}

func (s *Storage) DisableClientIP(clientIP string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.backend.Get(clientIP)
	if err != nil {
		data = &ClientIPData{}
	}
	data.Count++
	data.Time = time.Now()
	data.DisableUntil = time.Now().Add(duration)

	s.backend.Set(clientIP, data)
}

func (s *Storage) GetTimeDisabledClientIP(clientIP string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

	data, err := s.backend.List()
	if err != nil {
		return nil
	}

	clientIPs := make(map[string]int)
	for ip, d := range data {
		clientIPs[ip] = d.Count
	}
	return clientIPs
}

func (s *Storage) ResetClientIP(clientIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.backend.Delete(clientIP)
}

func (s *Storage) ResetDataClientIPs() {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	data, err := s.backend.List()
	if err != nil {
		log.Printf("Erro ao listar dados para limpeza: %v\n", err)
		return
	}

	for ip, d := range data {
		if d.DisableUntil.Before(now) && now.Sub(d.Time) > ttl {
			s.backend.Delete(ip)
			count++
		}
	}

	if count > 0 {
		log.Printf("Cleanup complete. Removed %d old entries.\n", count)
	}
}
