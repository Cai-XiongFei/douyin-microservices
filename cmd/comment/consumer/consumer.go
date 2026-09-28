package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func ConsumeCommentEvents(ctx context.Context) error {
	return apprabbitmq.ConsumeEvents(
		ctx,
		apprabbitmq.CommentCacheQueue,
		apprabbitmq.CommentChangedRoutingKey,
		handleCommentEvent,
	)
}

func handleCommentEvent(ctx context.Context, body []byte) (bool, error) {
	var event apprabbitmq.CommentChangedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return false, fmt.Errorf("invalid comment event JSON: %w", err)
	}
	if event.EventID == "" || event.UserID == 0 || event.VideoID == 0 || event.CommentID == 0 {
		return false, fmt.Errorf("comment event contains invalid identifiers")
	}
	if event.ActionType != 1 && event.ActionType != 2 {
		return false, fmt.Errorf("invalid comment action type")
	}

	processed, err := appredis.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return true, err
	}
	if processed {
		return false, nil
	}
	if err := appcache.DeleteCommentList(ctx, event.VideoID); err != nil {
		return true, err
	}
	if err := appredis.MarkEventProcessed(ctx, event.EventID); err != nil {
		return true, err
	}
	return false, nil
}
