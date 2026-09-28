package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func TestHandleMessageEvent(t *testing.T) {
	ctx := context.Background()
	userID := uint(time.Now().UnixNano())
	toUserID := userID + 1
	event := apprabbitmq.NewMessageCreatedEvent(1, userID, toUserID, "hello", time.Now())
	body, _ := json.Marshal(event)
	key := fmt.Sprintf("message:latest:%d:%d", userID, toUserID)

	defer appredis.GetClient().Del(ctx, key)
	defer appredis.DeleteProcessedEvent(ctx, event.EventID)

	if requeue, err := handleMessageEvent(ctx, body); err != nil || requeue {
		t.Fatalf("handle message event failed: requeue=%v err=%v", requeue, err)
	}
	message, err := appcache.GetLatestMessageBetweenUsers(ctx, userID, toUserID)
	if err != nil {
		t.Fatal(err)
	}
	if message == nil || message.Content != "hello" {
		t.Fatalf("unexpected latest message: %+v", message)
	}
}
