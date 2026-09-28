package redis

import (
	"context"
	"fmt"
	"time"
)

const processedEventTTL = 24 * time.Hour

func processedEventKey(eventID string) string {
	return fmt.Sprintf("event:processed:%s", eventID)
}

func IsEventProcessed(
	ctx context.Context,
	eventID string,
) (bool, error) {
	count, err := Client.Exists(
		ctx,
		processedEventKey(eventID),
	).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func MarkEventProcessed(
	ctx context.Context,
	eventID string,
) error {
	return Client.Set(
		ctx,
		processedEventKey(eventID),
		"1",
		processedEventTTL,
	).Err()
}

// 主要供测试结束时清理数据使用。
func DeleteProcessedEvent(
	ctx context.Context,
	eventID string,
) error {
	return Client.Del(
		ctx,
		processedEventKey(eventID),
	).Err()
}
