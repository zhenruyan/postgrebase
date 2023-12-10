package rediscache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache[T any] struct {
	client *redis.Client
}

func New[T any](client *redis.Client) *RedisCache[T] {
	return &RedisCache[T]{
		client: client,
	}
}

func (r *RedisCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	bytesvalue, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, bytesvalue, ttl).Err()
}

func (r *RedisCache[T]) Get(ctx context.Context, key string) (T, error) {
	var obj T
	cmd := r.client.Get(ctx, key)
	if err := cmd.Err(); err != nil {
		return obj, err
	}
	bytevalue, err := cmd.Bytes()
	if err != nil {
		return obj, err
	}
	err = json.Unmarshal(bytevalue, &obj)
	return obj, err
}

func (r *RedisCache[T]) Delete(ctx context.Context, keys ...string) error {
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return err
	}
	return nil
}
