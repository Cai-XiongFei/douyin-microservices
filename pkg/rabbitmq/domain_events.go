package rabbitmq

import (
	"fmt"
	"time"
)

const (
	RelationChangedRoutingKey = "relation.changed"
	RelationCacheQueue        = "relation.cache"
	CommentChangedRoutingKey  = "comment.changed"
	CommentCacheQueue         = "comment.cache"
	MessageCreatedRoutingKey  = "message.created"
	MessageCacheQueue         = "message.cache"
	VideoPublishedRoutingKey  = "video.published"
	VideoCacheQueue           = "video.cache"
)

type RelationChangedEvent struct {
	EventID    string `json:"event_id"`
	UserID     uint   `json:"user_id"`
	ToUserID   uint   `json:"to_user_id"`
	ActionType int32  `json:"action_type"`
	OccurredAt int64  `json:"occurred_at"`
}

type CommentChangedEvent struct {
	EventID    string `json:"event_id"`
	UserID     uint   `json:"user_id"`
	VideoID    uint   `json:"video_id"`
	CommentID  uint   `json:"comment_id"`
	ActionType int32  `json:"action_type"`
	OccurredAt int64  `json:"occurred_at"`
}

type MessageCreatedEvent struct {
	EventID    string `json:"event_id"`
	MessageID  uint   `json:"message_id"`
	FromUserID uint   `json:"from_user_id"`
	ToUserID   uint   `json:"to_user_id"`
	Content    string `json:"content"`
	CreatedAt  int64  `json:"created_at"`
}

type VideoPublishedEvent struct {
	EventID   string `json:"event_id"`
	VideoID   uint   `json:"video_id"`
	AuthorID  uint   `json:"author_id"`
	CreatedAt int64  `json:"created_at"`
}

func domainEventID(kind string, values ...uint) string {
	result := fmt.Sprintf("%s-%d", kind, time.Now().UnixNano())
	for _, value := range values {
		result += fmt.Sprintf("-%d", value)
	}
	return result
}

func NewRelationChangedEvent(userID uint, toUserID uint, actionType int32) RelationChangedEvent {
	return RelationChangedEvent{
		EventID:    domainEventID("relation", userID, toUserID),
		UserID:     userID,
		ToUserID:   toUserID,
		ActionType: actionType,
		OccurredAt: time.Now().UnixMilli(),
	}
}

func NewCommentChangedEvent(userID uint, videoID uint, commentID uint, actionType int32) CommentChangedEvent {
	return CommentChangedEvent{
		EventID:    domainEventID("comment", userID, videoID, commentID),
		UserID:     userID,
		VideoID:    videoID,
		CommentID:  commentID,
		ActionType: actionType,
		OccurredAt: time.Now().UnixMilli(),
	}
}

func NewMessageCreatedEvent(messageID uint, fromUserID uint, toUserID uint, content string, createdAt time.Time) MessageCreatedEvent {
	return MessageCreatedEvent{
		EventID:    domainEventID("message", messageID, fromUserID, toUserID),
		MessageID:  messageID,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Content:    content,
		CreatedAt:  createdAt.UnixMilli(),
	}
}

func NewVideoPublishedEvent(videoID uint, authorID uint, createdAt time.Time) VideoPublishedEvent {
	return VideoPublishedEvent{
		EventID:   domainEventID("video", videoID, authorID),
		VideoID:   videoID,
		AuthorID:  authorID,
		CreatedAt: createdAt.UnixMilli(),
	}
}
