package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPublishFavoriteChanged(t *testing.T) {
	channel, err := NewChannel()
	if err != nil {
		t.Fatalf("创建 Channel 失败：%v", err)
	}
	defer channel.Close()

	if err := DeclareEventsExchange(channel); err != nil {
		t.Fatalf("创建 Exchange 失败：%v", err)
	}

	// 创建临时队列接收测试消息。
	queue, err := channel.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("创建测试队列失败：%v", err)
	}

	err = channel.QueueBind(
		queue.Name,
		FavoriteChangedRoutingKey,
		EventsExchange,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("绑定测试队列失败：%v", err)
	}

	deliveries, err := channel.Consume(
		queue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("创建测试消费者失败：%v", err)
	}

	expected := NewFavoriteChangedEvent(
		5,
		4,
		1,
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := PublishFavoriteChanged(
		ctx,
		expected,
	); err != nil {
		t.Fatalf("发送收藏事件失败：%v", err)
	}

	select {
	case delivery := <-deliveries:
		var actual FavoriteChangedEvent

		if err := json.Unmarshal(
			delivery.Body,
			&actual,
		); err != nil {
			t.Fatalf("解析收藏事件失败：%v", err)
		}

		if actual.EventID != expected.EventID {
			t.Fatalf(
				"event_id 不一致：期望 %s，实际 %s",
				expected.EventID,
				actual.EventID,
			)
		}

		if actual.UserID != expected.UserID {
			t.Fatalf("user_id 不一致")
		}

		if actual.VideoID != expected.VideoID {
			t.Fatalf("video_id 不一致")
		}

		if actual.ActionType != expected.ActionType {
			t.Fatalf("action_type 不一致")
		}

		t.Logf(
			"成功收到收藏事件：%+v",
			actual,
		)

	case <-ctx.Done():
		t.Fatal("等待收藏事件超时")
	}
}
