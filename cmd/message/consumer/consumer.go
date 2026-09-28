package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"douyin/dal/db"
	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"

	"gorm.io/gorm"
)

func ConsumeMessageEvents(ctx context.Context) error {
	return apprabbitmq.ConsumeEvents(
		ctx,
		apprabbitmq.MessageCacheQueue,
		apprabbitmq.MessageCreatedRoutingKey,
		handleMessageEvent,
	)
}

func handleMessageEvent(ctx context.Context, body []byte) (bool, error) {
	var event apprabbitmq.MessageCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return false, fmt.Errorf("invalid message event JSON: %w", err)
	}
	if event.EventID == "" || event.MessageID == 0 || event.FromUserID == 0 || event.ToUserID == 0 {
		return false, fmt.Errorf("message event contains invalid identifiers")
	}

	processed, err := appredis.IsEventProcessed(ctx, event.EventID)
	if err != nil {
		return true, err
	}
	if processed {
		return false, nil
	}

	message := &db.Message{
		Model: gorm.Model{
			ID:        event.MessageID,
			CreatedAt: time.UnixMilli(event.CreatedAt),
		},
		FromUserID: event.FromUserID,
		ToUserID:   event.ToUserID,
		Content:    event.Content,
	}
	if err := appcache.SetLatestMessage(ctx, event.FromUserID, event.ToUserID, message); err != nil {
		return true, err
	}
	if err := appredis.MarkEventProcessed(ctx, event.EventID); err != nil {
		return true, err
	}
	return false, nil
}
