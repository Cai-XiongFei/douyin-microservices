package cache

import (
	"context"
	"douyin/dal/db"
	appredis "douyin/dal/redis"
	applock "douyin/internal/lock"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	redisSDK "github.com/redis/go-redis/v9"
)

const (
	userTTL              = 5 * time.Minute
	userLockTTL          = 5 * time.Second
	userLockWaitInterval = 50 * time.Millisecond
	userLockWaitAttempts = 20
)

func userKey(userID int64) string {
	return fmt.Sprintf("user:info:%d", userID)
}

func getCachedUser(
	ctx context.Context,
	key string,
) (*db.User, bool, error) {
	data, err := appredis.GetClient().
		Get(ctx, key).
		Bytes()

	if errors.Is(err, redisSDK.Nil) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	var user db.User

	if err := json.Unmarshal(data, &user); err != nil {
		// 缓存内容损坏时删除，后续重新查询MySQL。
		_ = appredis.GetClient().
			Del(ctx, key).
			Err()

		return nil, false, nil
	}

	return &user, true, nil
}

func loadUserAndSetCache(
	ctx context.Context,
	key string,
	userID int64,
) (*db.User, error) {
	user, err := db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return user, err
	}

	data, err := json.Marshal(user)
	if err != nil {
		return user, nil
	}

	if err := appredis.GetClient().
		Set(ctx, key, data, userTTL).
		Err(); err != nil {
		log.Printf("set user cache failed: %v", err)
	}

	return user, nil
}

func GetUserByID(
	ctx context.Context,
	userID int64,
) (*db.User, error) {
	key := userKey(userID)

	// 第一次查询Redis。
	user, found, err := getCachedUser(ctx, key)
	if err == nil && found {
		return user, nil
	}

	if err != nil {
		log.Printf(
			"query user cache failed, fallback to MySQL: %v",
			err,
		)

		return loadUserAndSetCache(ctx, key, userID)
	}

	lockKey := "lock:" + key

	redisLock, acquired, err := applock.TryAcquire(
		ctx,
		lockKey,
		userLockTTL,
	)
	if err != nil {
		// Redis锁不可用时不能让整个接口不可用，降级查询MySQL。
		log.Printf(
			"acquire user cache lock failed, fallback to MySQL: %v",
			err,
		)

		return loadUserAndSetCache(ctx, key, userID)
	}

	if !acquired {
		// 其他服务实例正在重建缓存，当前请求暂时等待。
		ticker := time.NewTicker(userLockWaitInterval)
		defer ticker.Stop()

		for attempt := 0; attempt < userLockWaitAttempts; attempt++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case <-ticker.C:
				user, found, err := getCachedUser(ctx, key)
				if err != nil {
					log.Printf(
						"wait for user cache failed: %v",
						err,
					)

					return loadUserAndSetCache(
						ctx,
						key,
						userID,
					)
				}

				if found {
					return user, nil
				}
			}
		}

		// 等待1秒后缓存仍未生成，为保证接口可用，降级查询MySQL。
		return loadUserAndSetCache(ctx, key, userID)
	}

	// 当前请求成功获得锁，结束时安全释放。
	defer func() {
		releaseCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Second,
		)
		defer cancel()

		if err := redisLock.Release(releaseCtx); err != nil &&
			!errors.Is(err, applock.ErrLockNotOwned) {
			log.Printf(
				"release user cache lock failed: %v",
				err,
			)
		}
	}()

	// 双重检查：获得锁之前，其他请求可能刚刚完成缓存重建。
	user, found, err = getCachedUser(ctx, key)
	if err == nil && found {
		return user, nil
	}

	if err != nil {
		log.Printf(
			"recheck user cache failed: %v",
			err,
		)
	}

	// 当前请求负责查询MySQL并重建缓存。
	return loadUserAndSetCache(ctx, key, userID)
}

func DeleteUser(ctx context.Context, userID uint) error {
	return appredis.GetClient().Del(ctx, userKey(int64(userID))).Err()
}

func InvalidateUser(ctx context.Context, userID uint) {
	if err := DeleteUser(ctx, userID); err != nil {
		log.Printf("delete user cache failed: %v", err)
	}
}
