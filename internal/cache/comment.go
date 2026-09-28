package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"douyin/dal/db"
	appredis "douyin/dal/redis"

	redisSDK "github.com/redis/go-redis/v9"
)

const commentListTTL = 5 * time.Minute

func commentListKey(videoID uint) string {
	return fmt.Sprintf("comment:list:%d", videoID)
}

func GetCommentsByVideoID(ctx context.Context, videoID uint) ([]*db.Comment, error) {
	key := commentListKey(videoID)
	data, err := appredis.GetClient().Get(ctx, key).Bytes()
	if err == nil {
		comments := make([]*db.Comment, 0)
		if unmarshalErr := json.Unmarshal(data, &comments); unmarshalErr == nil {
			return comments, nil
		}
		_ = appredis.GetClient().Del(ctx, key).Err()
	} else if !errors.Is(err, redisSDK.Nil) {
		log.Printf("query comment cache failed, fallback to MySQL: %v", err)
	}

	comments, err := db.GetCommentsByVideoID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	data, marshalErr := json.Marshal(comments)
	if marshalErr == nil {
		if setErr := appredis.GetClient().Set(ctx, key, data, commentListTTL).Err(); setErr != nil {
			log.Printf("set comment cache failed: %v", setErr)
		}
	}
	return comments, nil
}

func DeleteCommentList(ctx context.Context, videoID uint) error {
	return appredis.GetClient().Del(ctx, commentListKey(videoID)).Err()
}

func InvalidateCommentList(ctx context.Context, videoID uint) {
	if err := DeleteCommentList(ctx, videoID); err != nil {
		log.Printf("delete comment cache failed: %v", err)
	}
}
