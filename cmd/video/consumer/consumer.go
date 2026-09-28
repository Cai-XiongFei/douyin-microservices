package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func ConsumeVideoEvents(ctx context.Context) error {
	return apprabbitmq.ConsumeEvents(
		ctx,
		apprabbitmq.VideoCacheQueue,
		apprabbitmq.VideoPublishedRoutingKey,
		handleVideoEvent,
	)
}

func handleVideoEvent(ctx context.Context, body []byte) (bool, error) {
	var event apprabbitmq.VideoPublishedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return false, fmt.Errorf("invalid video event JSON: %w", err)
	}
	if event.EventID == "" || event.VideoID == 0 || event.AuthorID == 0 {
		return false, fmt.Errorf("video event contains invalid identifiers")
	}

	processed, err := appredis.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return true, err
	}
	if processed {
		return false, nil
	}
	if err := appcache.DeleteUser(ctx, event.AuthorID); err != nil {
		return true, err
	}
	if err := appredis.MarkEventProcessed(ctx, event.EventID); err != nil {
		return true, err
	}
	return false, nil
}
