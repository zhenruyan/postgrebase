package rediscache

import (
	"context"
	"testing"
	"time"
)

type testStruct struct {
	A int
	B string
}

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
	val, err := IntTest.Get(context.Background(), "int")
	if err != nil {
		t.Error(err)
	}
	t.Error(val)

	StringTest := New[string](conn)
	StringTestCal := "aaa"
	StringTest.Set(context.Background(), "str", StringTestCal, 10*time.Second)
	val2, err := StringTest.Get(context.Background(), "str")
	if err != nil {
		t.Error(err)
	}
	t.Error(val2)

	StructTest := New[testStruct](conn)

	st := testStruct{A: 1, B: "aaaaaaa"}

	err = StructTest.Set(context.TODO(), "st", st, 10*time.Second)

	if err != nil {
		t.Error(err)
	}

	val3, err := StructTest.Get(context.Background(), "st")
	if err != nil {
		t.Error(err)
	}
	t.Error(val3)
}
