package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	appredis "douyin/dal/redis"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func TestHandleVideoEvent(t *testing.T) {
	ctx := context.Background()
	authorID := uint(time.Now().UnixNano())
	event := apprabbitmq.NewVideoPublishedEvent(1, authorID, time.Now())
	body, _ := json.Marshal(event)
	key := fmt.Sprintf("user:info:%d", authorID)

	defer appredis.GetClient().Del(ctx, key)
	defer appredis.DeleteProcessedEvent(ctx, event.EventID)

	if err := appredis.GetClient().Set(ctx, key, "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if requeue, err := handleVideoEvent(ctx, body); err != nil || requeue {
		t.Fatalf("handle video event failed: requeue=%v err=%v", requeue, err)
	}
	if exists := appredis.GetClient().Exists(ctx, key).Val(); exists != 0 {
		t.Fatal("author cache was not deleted")
	}
}
