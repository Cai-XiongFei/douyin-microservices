package cache

import (
	"context"
	"log"

	"douyin/dal/db"
	appredis "douyin/dal/redis"
)

func IsFollowing(ctx context.Context, userID uint, toUserID uint) (bool, error) {
	following, found, err := appredis.GetRelationStatus(ctx, userID, toUserID)
	if err == nil && found {
		return following, nil
	}
	if err != nil {
		log.Printf("query relation cache failed, fallback to MySQL: %v", err)
	}

	following, err = db.IsFollowing(ctx, userID, toUserID)
	if err != nil {
		return false, err
	}
	if err := appredis.SetRelationStatus(ctx, userID, toUserID, following); err != nil {
		log.Printf("set relation cache failed: %v", err)
	}
	return following, nil
}

func InvalidateRelationStatus(ctx context.Context, userID uint, toUserID uint) {
	if err := appredis.DeleteRelationStatus(ctx, userID, toUserID); err != nil {
		log.Printf("delete relation cache failed: %v", err)
	}
}
