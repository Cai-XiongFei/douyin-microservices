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

func TestHandleRelationEvent(t *testing.T) {
	ctx := context.Background()
	userID := uint(time.Now().UnixNano())
	toUserID := userID + 1
	event := apprabbitmq.NewRelationChangedEvent(userID, toUserID, 1)
	body, _ := json.Marshal(event)

	defer appredis.DeleteRelationStatus(ctx, userID, toUserID)
	defer appredis.DeleteProcessedEvent(ctx, event.EventID)
	defer appredis.GetClient().Del(ctx, fmt.Sprintf("user:info:%d", userID), fmt.Sprintf("user:info:%d", toUserID))

	if err := appredis.SetRelationStatus(ctx, userID, toUserID, false); err != nil {
		t.Fatal(err)
	}
	if err := appredis.GetClient().Set(ctx, fmt.Sprintf("user:info:%d", userID), "stale", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if requeue, err := handleRelationEvent(ctx, body); err != nil || requeue {
		t.Fatalf("handle relation event failed: requeue=%v err=%v", requeue, err)
	}
	_, found, err := appredis.GetRelationStatus(ctx, userID, toUserID)
	if err != nil || found {
		t.Fatalf("relation cache was not deleted: found=%v err=%v", found, err)
	}
	if exists := appredis.GetClient().Exists(ctx, fmt.Sprintf("user:info:%d", userID)).Val(); exists != 0 {
		t.Fatal("user cache was not deleted")
	}
}
