package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	redisSDK "github.com/redis/go-redis/v9"
)

const favoriteStatusTTL = 10 * time.Minute

func favoriteStatusKey(userID uint, videoID uint) string {
	return fmt.Sprintf(
		"favorite:status:%d:%d",
		userID,
		videoID,
	)
}

// GetFavoriteStatus 查询缓存中的收藏状态。
//
// 第二个返回值 found：
// true  表示 Redis 中有缓存；
// false 表示 Redis 中没有缓存，需要查询 MySQL。
func GetFavoriteStatus(
	ctx context.Context,
	userID uint,
	videoID uint,
) (favorite bool, found bool, err error) {
	value, err := Client.Get(
		ctx,
		favoriteStatusKey(userID, videoID),
	).Result()

	if errors.Is(err, redisSDK.Nil) {
		return false, false, nil
	}

	if err != nil {
		return false, false, err
	}

	favorite, err = strconv.ParseBool(value)
	if err != nil {
		return false, false, err
	}

	return favorite, true, nil
}

// SetFavoriteStatus 把收藏状态写入 Redis。
func SetFavoriteStatus(
	ctx context.Context,
	userID uint,
	videoID uint,
	favorite bool,
) error {
	return Client.Set(
		ctx,
		favoriteStatusKey(userID, videoID),
		strconv.FormatBool(favorite),
		favoriteStatusTTL,
	).Err()
}

// DeleteFavoriteStatus 删除收藏状态缓存。
func DeleteFavoriteStatus(
	ctx context.Context,
	userID uint,
	videoID uint,
) error {
	return Client.Del(
		ctx,
		favoriteStatusKey(userID, videoID),
	).Err()
}
