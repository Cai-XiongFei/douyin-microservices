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

const latestMessageTTL = 24 * time.Hour

func latestMessageKey(userID uint, toUserID uint) string {
	if userID > toUserID {
		userID, toUserID = toUserID, userID
	}
	return fmt.Sprintf("message:latest:%d:%d", userID, toUserID)
}

func GetLatestMessageBetweenUsers(ctx context.Context, userID uint, toUserID uint) (*db.Message, error) {
	key := latestMessageKey(userID, toUserID)
	data, err := appredis.GetClient().Get(ctx, key).Bytes()
	if err == nil {
		if string(data) == "null" {
			return nil, nil
		}
		var message db.Message
		if unmarshalErr := json.Unmarshal(data, &message); unmarshalErr == nil {
			return &message, nil
		}
		_ = appredis.GetClient().Del(ctx, key).Err()
	} else if !errors.Is(err, redisSDK.Nil) {
		log.Printf("query latest-message cache failed, fallback to MySQL: %v", err)
	}

	message, err := db.GetLatestMessageBetweenUsers(ctx, userID, toUserID)
	if err != nil {
		return nil, err
	}
	if err := SetLatestMessage(ctx, userID, toUserID, message); err != nil {
		log.Printf("set latest-message cache failed: %v", err)
	}
	return message, nil
}

func SetLatestMessage(ctx context.Context, userID uint, toUserID uint, message *db.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return appredis.GetClient().Set(ctx, latestMessageKey(userID, toUserID), data, latestMessageTTL).Err()
}
