package limiter

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestTokenBucketAllow(t *testing.T) {
	ctx := context.Background()

	redisAddr := getEnv("REDIS_ADDR", "127.0.0.1:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "tiktokRedis")

	client := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect Redis failed: %v", err)
	}

	bucket := NewTokenBucket(client, "test:rate_limit:")

	// 每次测试使用单独的key，避免受到以前测试数据影响。
	key := "user:" + time.Now().Format("20060102150405.000000000")
	redisKey := "test:rate_limit:" + key

	t.Cleanup(func() {
		if err := client.Del(ctx, redisKey).Err(); err != nil {
			t.Logf("delete test key failed: %v", err)
		}
	})

	// 每秒补充2个令牌，桶容量为3，每次请求消耗1个。
	const (
		ratePerSecond = 2.0
		capacity      = int64(3)
		cost          = int64(1)
	)

	// 初始状态是满桶，因此前3次应该成功。
	for requestNumber := 1; requestNumber <= 3; requestNumber++ {
		result, err := bucket.Allow(
			ctx,
			key,
			ratePerSecond,
			capacity,
			cost,
		)
		if err != nil {
			t.Fatalf(
				"request %d returned error: %v",
				requestNumber,
				err,
			)
		}

		if !result.Allowed {
			t.Fatalf(
				"request %d should be allowed, remaining=%d",
				requestNumber,
				result.Remaining,
			)
		}

		t.Logf(
			"request %d allowed, remaining=%d",
			requestNumber,
			result.Remaining,
		)
	}

	// 令牌已经用完，第4次应该被拒绝。
	result, err := bucket.Allow(
		ctx,
		key,
		ratePerSecond,
		capacity,
		cost,
	)
	if err != nil {
		t.Fatalf("fourth request returned error: %v", err)
	}

	if result.Allowed {
		t.Fatalf(
			"fourth request should be rejected, remaining=%d",
			result.Remaining,
		)
	}

	t.Logf(
		"fourth request rejected, remaining=%d",
		result.Remaining,
	)

	// 每秒补充2个令牌，等待700毫秒可以恢复至少1个令牌。
	time.Sleep(700 * time.Millisecond)

	result, err = bucket.Allow(
		ctx,
		key,
		ratePerSecond,
		capacity,
		cost,
	)
	if err != nil {
		t.Fatalf("request after refill returned error: %v", err)
	}

	if !result.Allowed {
		t.Fatalf(
			"request after refill should be allowed, remaining=%d",
			result.Remaining,
		)
	}

	t.Logf(
		"request after refill allowed, remaining=%d",
		result.Remaining,
	)
}

func TestTokenBucketInvalidArguments(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{
		Addr: "127.0.0.1:6379",
	})
	defer client.Close()

	bucket := NewTokenBucket(client, "test:rate_limit:")

	tests := []struct {
		name     string
		key      string
		rate     float64
		capacity int64
		cost     int64
		wantErr  error
	}{
		{
			name:     "empty key",
			key:      "",
			rate:     1,
			capacity: 10,
			cost:     1,
			wantErr:  ErrEmptyKey,
		},
		{
			name:     "invalid rate",
			key:      "user:1",
			rate:     0,
			capacity: 10,
			cost:     1,
			wantErr:  ErrInvalidRate,
		},
		{
			name:     "invalid capacity",
			key:      "user:1",
			rate:     1,
			capacity: 0,
			cost:     1,
			wantErr:  ErrInvalidCapacity,
		},
		{
			name:     "invalid cost",
			key:      "user:1",
			rate:     1,
			capacity: 10,
			cost:     0,
			wantErr:  ErrInvalidCost,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := bucket.Allow(
				context.Background(),
				test.key,
				test.rate,
				test.capacity,
				test.cost,
			)

			if err != test.wantErr {
				t.Fatalf(
					"expected error %v, got %v",
					test.wantErr,
					err,
				)
			}
		})
	}
}

func getEnv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}

	return value
}
