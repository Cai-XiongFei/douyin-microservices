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

func TestHandleCommentEvent(t *testing.T) {
	ctx := context.Background()
	videoID := uint(time.Now().UnixNano())
	event := apprabbitmq.NewCommentChangedEvent(1, videoID, 1, 1)
	body, _ := json.Marshal(event)
	key := fmt.Sprintf("comment:list:%d", videoID)

	defer appredis.GetClient().Del(ctx, key)
	defer appredis.DeleteProcessedEvent(ctx, event.EventID)

	if err := appredis.GetClient().Set(ctx, key, "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if requeue, err := handleCommentEvent(ctx, body); err != nil || requeue {
		t.Fatalf("handle comment event failed: requeue=%v err=%v", requeue, err)
	}
	if exists := appredis.GetClient().Exists(ctx, key).Val(); exists != 0 {
		t.Fatal("comment cache was not deleted")
	}
}
