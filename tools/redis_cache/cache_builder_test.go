package rediscache

import (
	"context"
	"testing"
)

func TestMain(m *testing.M) {
	conn, cb := NewRedisParseUrl("")
	defer cb()
	conn.Ping(context.TODO())

}
