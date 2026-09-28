package consumer

import (
	"context"
	"testing"

	appredis "douyin/dal/redis"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func TestHandleFavoriteChangedEvent(t *testing.T) {
	ctx := context.Background()

	event := apprabbitmq.NewFavoriteChangedEvent(
		50001,
		60001,
		1,
	)

	defer func() {
		_ = appredis.DeleteFavoriteStatus(
			ctx,
			event.UserID,
			event.VideoID,
		)

		_ = appredis.DeleteProcessedEvent(
			ctx,
			event.EventID,
		)
	}()

	// 模拟 Redis 中存在旧缓存。
	if err := appredis.SetFavoriteStatus(
		ctx,
		event.UserID,
		event.VideoID,
		true,
	); err != nil {
		t.Fatalf("写入测试缓存失败：%v", err)
	}

	if err := HandleFavoriteChangedEvent(
		ctx,
		event,
	); err != nil {
		t.Fatalf("处理收藏事件失败：%v", err)
	}

	// 消费完成后，旧缓存应被删除。
	_, found, err := appredis.GetFavoriteStatus(
		ctx,
		event.UserID,
		event.VideoID,
	)
	if err != nil {
		t.Fatalf("查询收藏缓存失败：%v", err)
	}

	if found {
		t.Fatal("收藏缓存没有被删除")
	}

	processed, err := appredis.IsEventProcessed(
		ctx,
		event.EventID,
	)
	if err != nil {
		t.Fatalf("查询消息处理记录失败：%v", err)
	}

	if !processed {
		t.Fatal("消息没有被记录为已处理")
	}

	// 再处理一次相同消息，应该直接安全返回。
	if err := HandleFavoriteChangedEvent(
		ctx,
		event,
	); err != nil {
		t.Fatalf("重复处理消息失败：%v", err)
	}
}
