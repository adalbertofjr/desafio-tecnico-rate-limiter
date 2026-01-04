package ratelimiter

import "sync"

type MemoryBackend struct {
	mu   sync.RWMutex
	data map[string]*ClientIPData
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		data: make(map[string]*ClientIPData),
	}
}

func (mb *MemoryBackend) Get(clientIP string) (*ClientIPData, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	data, exists := mb.data[clientIP]
	if !exists {
		return nil, ErrNotFound
	}
	return data, nil
}

func (mb *MemoryBackend) Set(clientIP string, data *ClientIPData) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.data[clientIP] = data
	return nil
}

func (mb *MemoryBackend) Delete(clientIP string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	delete(mb.data, clientIP)
	return nil
}

func (mb *MemoryBackend) List() (map[string]*ClientIPData, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	// Create a deep copy to avoid race conditions
	copyData := make(map[string]*ClientIPData)
	for k, v := range mb.data {
		dataCopy := *v
		copyData[k] = &dataCopy
	}
	return copyData, nil
}

func (mb *MemoryBackend) Clear() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.data = make(map[string]*ClientIPData)
	return nil
}
