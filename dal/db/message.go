package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Message 表示一条用户私信。
type Message struct {
	gorm.Model

	FromUserID uint   `gorm:"not null;index:idx_message_from_to;index"`
	ToUserID   uint   `gorm:"not null;index:idx_message_from_to;index"`
	Content    string `gorm:"type:varchar(255);not null"`
}

func (Message) TableName() string {
	return "messages"
}

// CreateMessage 保存一条消息。
func CreateMessage(
	ctx context.Context,
	fromUserID uint,
	toUserID uint,
	content string,
) (*Message, error) {
	content = strings.TrimSpace(content)

	if fromUserID == 0 || toUserID == 0 {
		return nil, errors.New("user id must be greater than zero")
	}
	if fromUserID == toUserID {
		return nil, errors.New("cannot send message to yourself")
	}
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	var fromUser User
	if err := GetDB().WithContext(ctx).
		First(&fromUser, fromUserID).Error; err != nil {
		return nil, fmt.Errorf("query sender failed: %w", err)
	}

	var toUser User
	if err := GetDB().WithContext(ctx).
		First(&toUser, toUserID).Error; err != nil {
		return nil, fmt.Errorf("query receiver failed: %w", err)
	}

	message := &Message{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Content:    content,
	}

	if err := GetDB().WithContext(ctx).
		Create(message).Error; err != nil {
		return nil, fmt.Errorf("create message failed: %w", err)
	}

	return message, nil
}

// GetMessagesBetweenUsers 查询两个用户之间、指定时间之后的聊天记录。
func GetMessagesBetweenUsers(
	ctx context.Context,
	userID uint,
	toUserID uint,
	preMsgTime int64,
) ([]*Message, error) {
	messages := make([]*Message, 0)

	query := GetDB().WithContext(ctx).
		Where(
			"(from_user_id = ? AND to_user_id = ?) OR "+
				"(from_user_id = ? AND to_user_id = ?)",
			userID,
			toUserID,
			toUserID,
			userID,
		)

	if preMsgTime > 0 {
		query = query.Where(
			"created_at > ?",
			time.UnixMilli(preMsgTime),
		)
	}

	if err := query.
		Order("created_at ASC").
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// GetLatestMessageBetweenUsers 查询两个人之间的最后一条消息。
func GetLatestMessageBetweenUsers(
	ctx context.Context,
	userID uint,
	toUserID uint,
) (*Message, error) {
	var message Message

	err := GetDB().WithContext(ctx).
		Where(
			"(from_user_id = ? AND to_user_id = ?) OR "+
				"(from_user_id = ? AND to_user_id = ?)",
			userID,
			toUserID,
			toUserID,
			userID,
		).
		Order("created_at DESC").
		Order("id DESC").
		First(&message).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &message, nil
}
