package rediscache

import (
	"context"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	conn, cb := NewRedisParseUrl("redis://localhost:6379/0")
	defer cb()
	cmd := conn.Ping(context.TODO())
	res := cmd.Val()
	t.Error(res)
	IntTest := New[int](conn)
	IntTestVal := 123
	err := IntTest.Set(context.Background(), "int", IntTestVal, 10*time.Second)
	if err != nil {
		t.Error(err)
	}

}
