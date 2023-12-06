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

// func (r *RedisCache[T]) Delete(ctx context.Context, keys ...string) error {
// 	if err := r.client.Del(ctx, keys...).Err(); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *RedisCache[T]) Batch(ctx context.Context, keys ...string) []T,error {
// 	cmd := r.client.MGet(ctx, keys...)
// 	if cmd.Err() != nil {
// 		return nil
// 	}

// 	results := make([][]byte, 0, len(keys))

// 	convertFun := func(in any) ([]byte, bool) {
// 		if in == nil {
// 			return nil, false
// 		}
// 		if cacheVal, ok := in.(string); ok {
// 			return []byte(cacheVal), true
// 		}
// 		return nil, false
// 	}

// 	for _, cacheVal := range cmd.Val() {
// 		if val, ok := convertFun(cacheVal); ok {
// 			results = append(results, val)
// 		}
// 	}
// 	return results
// }
