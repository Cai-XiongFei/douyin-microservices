package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func ConsumeRelationEvents(ctx context.Context) error {
	return apprabbitmq.ConsumeEvents(
		ctx,
		apprabbitmq.RelationCacheQueue,
		apprabbitmq.RelationChangedRoutingKey,
		handleRelationEvent,
	)
}

func handleRelationEvent(ctx context.Context, body []byte) (bool, error) {
	var event apprabbitmq.RelationChangedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return false, fmt.Errorf("invalid relation event JSON: %w", err)
	}
	if event.EventID == "" || event.UserID == 0 || event.ToUserID == 0 {
		return false, fmt.Errorf("relation event contains invalid identifiers")
	}
	if event.ActionType != 1 && event.ActionType != 2 {
		return false, fmt.Errorf("invalid relation action type")
	}

	processed, err := appredis.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return true, err
	}
	if processed {
		return false, nil
	}

	if err := appredis.DeleteRelationStatus(ctx, event.UserID, event.ToUserID); err != nil {
		return true, err
	}
	if err := appcache.DeleteUser(ctx, event.UserID); err != nil {
		return true, err
	}
	if err := appcache.DeleteUser(ctx, event.ToUserID); err != nil {
		return true, err
	}
	if err := appredis.MarkEventProcessed(ctx, event.EventID); err != nil {
		return true, err
	}
	return false, nil
}
