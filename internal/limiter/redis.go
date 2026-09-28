package limiter

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrNilClient       = errors.New("redis client cannot be nil")
	ErrEmptyKey        = errors.New("limiter key cannot be empty")
	ErrInvalidRate     = errors.New("rate must be greater than zero")
	ErrInvalidCapacity = errors.New("capacity must be greater than zero")
	ErrInvalidCost     = errors.New("cost must be greater than zero")
)

// tokenBucketScript 原子完成：
// 1. 读取当前令牌数和上次补充时间。
// 2. 根据经过的时间补充令牌。
// 3. 判断令牌是否足够。
// 4. 扣除令牌。
// 5. 保存最新状态。
var tokenBucketScript = goredis.NewScript(`
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])

-- 使用 Redis 服务器时间，避免多个 API 实例的系统时间不一致。
local redis_time = redis.call("TIME")
local now_ms = tonumber(redis_time[1]) * 1000
	+ math.floor(tonumber(redis_time[2]) / 1000)

local values = redis.call(
	"HMGET",
	KEYS[1],
	"tokens",
	"last_refill_ms"
)

local tokens
local last_refill_ms

-- 第一次请求时，把令牌桶初始化为满桶。
if values[1] == false or values[2] == false then
	tokens = capacity
	last_refill_ms = now_ms
else
	tokens = tonumber(values[1])
	last_refill_ms = tonumber(values[2])
end

-- 防止时间异常造成负数。
local elapsed_ms = math.max(0, now_ms - last_refill_ms)

-- 根据经过的时间补充令牌，但不能超过桶容量。
local refill_tokens = (elapsed_ms / 1000) * rate
tokens = math.min(capacity, tokens + refill_tokens)

local allowed = 0

-- 令牌足够时才允许请求，并扣除对应令牌。
if tokens >= cost then
	allowed = 1
	tokens = tokens - cost
end

redis.call(
	"HSET",
	KEYS[1],
	"tokens",
	tokens,
	"last_refill_ms",
	now_ms
)

-- 长时间没有请求时自动删除限流记录。
-- 这里设置为“从空桶恢复到满桶时间”的两倍。
local ttl_ms = math.ceil((capacity / rate) * 2000)

if ttl_ms < 1000 then
	ttl_ms = 1000
end

redis.call("PEXPIRE", KEYS[1], ttl_ms)

return {
	allowed,
	math.floor(tokens)
}
`)

// Result 表示一次限流判断结果。
type Result struct {
	Allowed   bool
	Remaining int64
}

// TokenBucket 是基于 Redis 的分布式令牌桶。
type TokenBucket struct {
	client    goredis.UniversalClient
	keyPrefix string
}

// NewTokenBucket 创建令牌桶。
func NewTokenBucket(
	client goredis.UniversalClient,
	keyPrefix string,
) *TokenBucket {
	if keyPrefix == "" {
		keyPrefix = "rate_limit:"
	}

	return &TokenBucket{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Allow 判断本次请求是否可以消耗令牌。
func (b *TokenBucket) Allow(
	ctx context.Context,
	key string,
	ratePerSecond float64,
	capacity int64,
	cost int64,
) (Result, error) {
	if b == nil || b.client == nil {
		return Result{}, ErrNilClient
	}

	if key == "" {
		return Result{}, ErrEmptyKey
	}

	if ratePerSecond <= 0 {
		return Result{}, ErrInvalidRate
	}

	if capacity <= 0 {
		return Result{}, ErrInvalidCapacity
	}

	if cost <= 0 {
		return Result{}, ErrInvalidCost
	}

	redisKey := b.keyPrefix + key

	values, err := tokenBucketScript.Run(
		ctx,
		b.client,
		[]string{redisKey},
		ratePerSecond,
		capacity,
		cost,
	).Slice()
	if err != nil {
		return Result{}, fmt.Errorf("execute token bucket script: %w", err)
	}

	if len(values) != 2 {
		return Result{}, fmt.Errorf(
			"unexpected token bucket result length: %d",
			len(values),
		)
	}

	allowedValue, ok := values[0].(int64)
	if !ok {
		return Result{}, fmt.Errorf(
			"unexpected allowed value type: %T",
			values[0],
		)
	}

	remainingValue, ok := values[1].(int64)
	if !ok {
		return Result{}, fmt.Errorf(
			"unexpected remaining value type: %T",
			values[1],
		)
	}

	return Result{
		Allowed:   allowedValue == 1,
		Remaining: remainingValue,
	}, nil
}
