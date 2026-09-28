package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	appredis "douyin/dal/redis"
)

// RedisLock 表示当前请求成功获得的一把Redis分布式锁。
type RedisLock struct {
	key   string
	token string
	ttl   time.Duration
}

// TryAcquire 尝试获取分布式锁。
//
// 返回值：
// lock：成功获得的锁
// acquired：是否成功获得锁
// err：Redis操作或参数错误
func TryAcquire(
	ctx context.Context,
	key string,
	ttl time.Duration,
) (*RedisLock, bool, error) {
	if key == "" {
		return nil, false, errors.New("lock key cannot be empty")
	}

	if ttl <= 0 {
		return nil, false, errors.New("lock ttl must be greater than zero")
	}

	token, err := newToken()
	if err != nil {
		return nil, false, fmt.Errorf(
			"generate lock token failed: %w",
			err,
		)
	}

	acquired, err := appredis.GetClient().
		SetNX(ctx, key, token, ttl).
		Result()
	if err != nil {
		return nil, false, fmt.Errorf(
			"acquire redis lock failed: %w",
			err,
		)
	}

	if !acquired {
		return nil, false, nil
	}

	return &RedisLock{
		key:   key,
		token: token,
		ttl:   ttl,
	}, true, nil
}

// newToken 为每次加锁生成唯一令牌。
func newToken() (string, error) {
	data := make([]byte, 16)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return hex.EncodeToString(data), nil
}

// ErrLockNotOwned 表示锁已过期，或者当前请求不是锁的持有者。
var ErrLockNotOwned = errors.New(
	"redis lock is not owned by current caller or has expired",
)

// releaseLockScript 使用Lua保证“比较令牌”和“删除锁”原子执行。
const releaseLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end

return 0
`

// Release 释放当前请求持有的Redis分布式锁。
func (lock *RedisLock) Release(ctx context.Context) error {
	if lock == nil {
		return errors.New("redis lock cannot be nil")
	}

	result, err := appredis.GetClient().
		Eval(
			ctx,
			releaseLockScript,
			[]string{lock.key},
			lock.token,
		).
		Int64()
	if err != nil {
		return fmt.Errorf(
			"release redis lock failed: %w",
			err,
		)
	}

	if result == 0 {
		return ErrLockNotOwned
	}

	return nil
}
