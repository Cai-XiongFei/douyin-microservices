package lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testLockKey(testName string) string {
	return fmt.Sprintf(
		"lock:test:%s:%d",
		testName,
		time.Now().UnixNano(),
	)
}

func TestRedisLockAcquireAndRelease(t *testing.T) {
	ctx := context.Background()
	key := testLockKey("acquire-release")

	firstLock, acquired, err := TryAcquire(ctx, key, 5*time.Second)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("first request did not acquire lock")
	}

	// 使用错误 token 模拟其他请求释放当前请求持有的锁。
	fakeLock := &RedisLock{
		key:   key,
		token: "wrong-token",
		ttl:   5 * time.Second,
	}
	if err := fakeLock.Release(ctx); !errors.Is(err, ErrLockNotOwned) {
		t.Fatalf("fake release error = %v, want ErrLockNotOwned", err)
	}

	// 错误 token 不能删除真正持有者的锁。
	_, acquired, err = TryAcquire(ctx, key, 5*time.Second)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	if acquired {
		t.Fatal("lock was incorrectly deleted by wrong token")
	}

	if err := firstLock.Release(ctx); err != nil {
		t.Fatalf("owner release failed: %v", err)
	}

	// 正确释放后，新请求应该能够再次获得同名锁。
	thirdLock, acquired, err := TryAcquire(ctx, key, 5*time.Second)
	if err != nil {
		t.Fatalf("third acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("lock was not available after release")
	}
	if err := thirdLock.Release(ctx); err != nil {
		t.Fatalf("third lock release failed: %v", err)
	}
}

func TestRedisLockConcurrentAcquire(t *testing.T) {
	ctx := context.Background()
	key := testLockKey("concurrent")

	const workerCount = 20

	start := make(chan struct{})
	errorsChannel := make(chan error, workerCount)

	var waitGroup sync.WaitGroup
	var acquiredCount int32
	var winner *RedisLock
	var winnerMutex sync.Mutex

	for index := 0; index < workerCount; index++ {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()
			<-start

			redisLock, acquired, err := TryAcquire(
				ctx,
				key,
				5*time.Second,
			)
			if err != nil {
				errorsChannel <- err
				return
			}
			if !acquired {
				return
			}

			atomic.AddInt32(&acquiredCount, 1)

			winnerMutex.Lock()
			winner = redisLock
			winnerMutex.Unlock()
		}()
	}

	// 同时放行所有协程，让它们竞争同一个 Redis Key。
	close(start)
	waitGroup.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		t.Fatalf("acquire redis lock failed: %v", err)
	}
	if got := atomic.LoadInt32(&acquiredCount); got != 1 {
		t.Fatalf("acquired count = %d, want 1", got)
	}

	winnerMutex.Lock()
	winningLock := winner
	winnerMutex.Unlock()

	if winningLock == nil {
		t.Fatal("winning lock is nil")
	}
	if err := winningLock.Release(ctx); err != nil {
		t.Fatalf("release winning lock failed: %v", err)
	}
}

func TestRedisLockExpires(t *testing.T) {
	ctx := context.Background()
	key := testLockKey("expires")

	oldLock, acquired, err := TryAcquire(ctx, key, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire expiring lock failed: %v", err)
	}
	if !acquired {
		t.Fatal("expiring lock was not acquired")
	}

	time.Sleep(150 * time.Millisecond)

	newLock, acquired, err := TryAcquire(ctx, key, 5*time.Second)
	if err != nil {
		t.Fatalf("acquire lock after expiration failed: %v", err)
	}
	if !acquired {
		t.Fatal("lock was not available after TTL expiration")
	}

	// 旧持有者不能释放新持有者获得的同名锁。
	if err := oldLock.Release(ctx); !errors.Is(err, ErrLockNotOwned) {
		t.Fatalf("old owner release error = %v, want ErrLockNotOwned", err)
	}
	if err := newLock.Release(ctx); err != nil {
		t.Fatalf("new owner release failed: %v", err)
	}
}
