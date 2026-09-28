package cache

import (
	"context"
	"log"

	"douyin/dal/db"
	appredis "douyin/dal/redis"
)

// IsFavorite 先查询 Redis，未命中时再查询 MySQL。
func IsFavorite(
	ctx context.Context,
	userID uint,
	videoID uint,
) (bool, error) {
	favorite, found, err := appredis.GetFavoriteStatus(
		ctx,
		userID,
		videoID,
	)

	// Redis 查询成功并且缓存存在，直接返回。
	if err == nil && found {
		return favorite, nil
	}

	// Redis 临时出错不应该导致接口失败，继续查询 MySQL。
	if err != nil {
		log.Printf(
			"query favorite cache failed, fallback to MySQL: %v",
			err,
		)
	}

	favorite, err = db.IsFavorite(
		ctx,
		userID,
		videoID,
	)
	if err != nil {
		return false, err
	}

	// 回填缓存失败不影响本次请求。
	if err := appredis.SetFavoriteStatus(
		ctx,
		userID,
		videoID,
		favorite,
	); err != nil {
		log.Printf("set favorite cache failed: %v", err)
	}

	return favorite, nil
}

// InvalidateFavoriteStatus 删除已经过期的收藏缓存。
func InvalidateFavoriteStatus(
	ctx context.Context,
	userID uint,
	videoID uint,
) {
	if err := appredis.DeleteFavoriteStatus(
		ctx,
		userID,
		videoID,
	); err != nil {
		log.Printf("delete favorite cache failed: %v", err)
	}
}
