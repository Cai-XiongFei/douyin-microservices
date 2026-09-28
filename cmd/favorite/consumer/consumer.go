package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"douyin/dal/db"
	appredis "douyin/dal/redis"
	appcache "douyin/internal/cache"
	apprabbitmq "douyin/pkg/rabbitmq"
)

func HandleFavoriteChangedEvent(
	ctx context.Context,
	event apprabbitmq.FavoriteChangedEvent,
) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id cannot be empty")
	}

	if event.UserID == 0 {
		return fmt.Errorf("user_id cannot be zero")
	}

	if event.VideoID == 0 {
		return fmt.Errorf("video_id cannot be zero")
	}

	if event.ActionType != 1 && event.ActionType != 2 {
		return fmt.Errorf("action_type must be 1 or 2")
	}

	processed, err := appredis.IsEventProcessed(
		ctx,
		event.EventID,
	)
	if err != nil {
		return fmt.Errorf(
			"query processed event failed: %w",
			err,
		)
	}

	// 相同事件已经处理过，直接返回成功。
	if processed {
		return nil
	}

	// 删除旧的收藏状态缓存。
	if err := appredis.DeleteFavoriteStatus(
		ctx,
		event.UserID,
		event.VideoID,
	); err != nil {
		return fmt.Errorf(
			"delete favorite cache failed: %w",
			err,
		)
	}

	// 业务处理完成后再记录消息已经处理。
	if err := appcache.DeleteUser(ctx, event.UserID); err != nil {
		return fmt.Errorf("delete favorite-user cache failed: %w", err)
	}

	video, err := db.GetVideoByID(ctx, int64(event.VideoID))
	if err != nil {
		return fmt.Errorf("query favorite video failed: %w", err)
	}
	if video != nil {
		if err := appcache.DeleteUser(ctx, video.AuthorID); err != nil {
			return fmt.Errorf("delete video-author cache failed: %w", err)
		}
	}

	if err := appredis.MarkEventProcessed(
		ctx,
		event.EventID,
	); err != nil {
		return fmt.Errorf(
			"mark event processed failed: %w",
			err,
		)
	}

	return nil
}

func ConsumeFavoriteEvents(ctx context.Context) error {
	return apprabbitmq.ConsumeEvents(
		ctx,
		apprabbitmq.FavoriteCacheQueue,
		apprabbitmq.FavoriteChangedRoutingKey,
		handleFavoriteEvent,
	)
}

func handleFavoriteEvent(ctx context.Context, body []byte) (bool, error) {
	var event apprabbitmq.FavoriteChangedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return false, fmt.Errorf("invalid favorite event JSON: %w", err)
	}
	if event.EventID == "" || event.UserID == 0 || event.VideoID == 0 {
		return false, fmt.Errorf("favorite event contains invalid identifiers")
	}
	if event.ActionType != 1 && event.ActionType != 2 {
		return false, fmt.Errorf("invalid favorite action type")
	}
	if err := HandleFavoriteChangedEvent(ctx, event); err != nil {
		return true, err
	}
	return false, nil
}
