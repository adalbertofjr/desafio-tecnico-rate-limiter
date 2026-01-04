package ratelimiter

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/redis/go-redis/v9"
)

type RedisBackend struct {
	mu     sync.RWMutex
	ctx    context.Context
	client *redis.Client
}

func NewRedisBackend(ctx context.Context) *RedisBackend {
	return &RedisBackend{
		mu:  sync.RWMutex{},
		ctx: ctx,
		client: redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		}),
	}
}

func (rb *RedisBackend) Get(clientIP string) (*ClientIPData, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result, err := rb.client.Get(rb.ctx, clientIP).Result()
	if err != nil {
		return nil, err
	}

	var data ClientIPData
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (rb *RedisBackend) Set(clientIP string, data *ClientIPData) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return rb.client.Set(rb.ctx, clientIP, jsonData, 0).Err()
}

func (rb *RedisBackend) Delete(clientIP string) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	_, err := rb.client.Del(rb.ctx, clientIP).Result()
	return err
}

func (rb *RedisBackend) List() (map[string]*ClientIPData, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	keys, err := rb.client.Keys(rb.ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ClientIPData)
	for _, key := range keys {
		val, err := rb.client.Get(rb.ctx, key).Result()
		if err != nil {
			return nil, err
		}

		var data ClientIPData
		err = json.Unmarshal([]byte(val), &data)
		if err != nil {
			return nil, err
		}

		result[key] = &data
	}

	return result, nil
}

func (rb *RedisBackend) Clear() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	keys, err := rb.client.Keys(rb.ctx, "*").Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		_, err = rb.client.Del(rb.ctx, keys...).Result()
		return err
	}

	return nil
}
