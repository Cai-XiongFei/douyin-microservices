package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	redisSDK "github.com/redis/go-redis/v9"
)

const relationStatusTTL = 10 * time.Minute

func relationStatusKey(userID uint, toUserID uint) string {
	return fmt.Sprintf("relation:status:%d:%d", userID, toUserID)
}

func GetRelationStatus(ctx context.Context, userID uint, toUserID uint) (following bool, found bool, err error) {
	value, err := Client.Get(ctx, relationStatusKey(userID, toUserID)).Result()
	if errors.Is(err, redisSDK.Nil) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}

	following, err = strconv.ParseBool(value)
	if err != nil {
		return false, false, err
	}
	return following, true, nil
}

func SetRelationStatus(ctx context.Context, userID uint, toUserID uint, following bool) error {
	return Client.Set(ctx, relationStatusKey(userID, toUserID), strconv.FormatBool(following), relationStatusTTL).Err()
}

func DeleteRelationStatus(ctx context.Context, userID uint, toUserID uint) error {
	return Client.Del(ctx, relationStatusKey(userID, toUserID)).Err()
}
