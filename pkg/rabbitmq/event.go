package rabbitmq

import (
	"fmt"
	"time"
)

const (
	EventsExchange            = "douyin.events"
	FavoriteChangedRoutingKey = "favorite.changed"
	FavoriteCacheQueue        = "favorite.cache"
)

type FavoriteChangedEvent struct {
	EventID    string `json:"event_id"`
	UserID     uint   `json:"user_id"`
	VideoID    uint   `json:"video_id"`
	ActionType int32  `json:"action_type"`
	OccurredAt int64  `json:"occurred_at"`
}

func NewFavoriteChangedEvent(
	userID uint,
	videoID uint,
	actionType int32,
) FavoriteChangedEvent {
	now := time.Now()

	return FavoriteChangedEvent{
		EventID: fmt.Sprintf(
			"%d-%d-%d",
			now.UnixNano(),
			userID,
			videoID,
		),
		UserID:     userID,
		VideoID:    videoID,
		ActionType: actionType,
		OccurredAt: now.UnixMilli(),
	}
}
