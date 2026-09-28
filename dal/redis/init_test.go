package redis

import (
	"context"
	"testing"
)

func TestRedisConnection(t *testing.T) {
	result, err := Client.Ping(context.Background()).Result()
	if err != nil {
		t.Fatalf("Redis Ping 失败：%v", err)
	}

	if result != "PONG" {
		t.Fatalf("期望得到 PONG，实际得到：%s", result)
	}

	t.Logf("Redis 连接成功：%s", result)
}
