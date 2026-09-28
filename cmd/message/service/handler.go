package service

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	messagepb "douyin/kitex/kitex_gen/message"
	appjwt "douyin/pkg/jwt"
	apprabbitmq "douyin/pkg/rabbitmq"
)

const maxMessageLength = 255

type MessageServiceImpl struct {
	jwtManager *appjwt.JWT
}

func NewMessageServiceImpl(signingKey []byte) *MessageServiceImpl {
	return &MessageServiceImpl{
		jwtManager: appjwt.NewJWT(signingKey),
	}
}

// MessageAction 发送一条私信。
func (s *MessageServiceImpl) MessageAction(
	ctx context.Context,
	req *messagepb.MessageActionRequest,
) (*messagepb.MessageActionResponse, error) {
	if req == nil {
		return messageActionError("request cannot be nil"), nil
	}
	if req.GetToken() == "" {
		return messageActionError("token cannot be empty"), nil
	}
	if req.GetToUserId() <= 0 {
		return messageActionError("to_user_id must be greater than zero"), nil
	}
	if req.GetActionType() != 1 {
		return messageActionError("action_type must be 1"), nil
	}

	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return messageActionError("message content cannot be empty"), nil
	}
	if utf8.RuneCountInString(content) > maxMessageLength {
		return messageActionError(
			"message content cannot exceed 255 characters",
		), nil
	}

	claims, ok := s.parseToken(req.GetToken())
	if !ok {
		return messageActionError("token is invalid or expired"), nil
	}
	if claims.Id == req.GetToUserId() {
		return messageActionError("cannot send message to yourself"), nil
	}

	friends, err := areFriends(
		ctx,
		uint(claims.Id),
		uint(req.GetToUserId()),
	)
	if err != nil {
		return messageActionError("query friend relation failed"), nil
	}
	if !friends {
		return messageActionError(
			"only mutual followers can send messages",
		), nil
	}

	messageRecord, err := db.CreateMessage(
		ctx,
		uint(claims.Id),
		uint(req.GetToUserId()),
		content,
	)
	if err != nil {
		return messageActionError("create message failed"), nil
	}

	event := apprabbitmq.NewMessageCreatedEvent(
		messageRecord.ID,
		messageRecord.FromUserID,
		messageRecord.ToUserID,
		messageRecord.Content,
		messageRecord.CreatedAt,
	)
	publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	publishErr := apprabbitmq.PublishMessageCreated(publishCtx, event)
	cancel()
	if publishErr != nil {
		log.Printf("publish message event failed, fallback to direct cache update: %v", publishErr)
		if cacheErr := appcache.SetLatestMessage(ctx, messageRecord.FromUserID, messageRecord.ToUserID, messageRecord); cacheErr != nil {
			log.Printf("update latest-message cache failed: %v", cacheErr)
		}
	}

	return &messagepb.MessageActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
	}, nil
}

// MessageChat 查询当前用户和指定好友之间的新消息。
func (s *MessageServiceImpl) MessageChat(
	ctx context.Context,
	req *messagepb.MessageChatRequest,
) (*messagepb.MessageChatResponse, error) {
	if req == nil {
		return messageChatError("request cannot be nil"), nil
	}
	if req.GetToken() == "" {
		return messageChatError("token cannot be empty"), nil
	}
	if req.GetToUserId() <= 0 {
		return messageChatError("to_user_id must be greater than zero"), nil
	}
	if req.GetPreMsgTime() < 0 {
		return messageChatError("pre_msg_time cannot be negative"), nil
	}

	claims, ok := s.parseToken(req.GetToken())
	if !ok {
		return messageChatError("token is invalid or expired"), nil
	}
	if claims.Id == req.GetToUserId() {
		return messageChatError("cannot query chat with yourself"), nil
	}

	friends, err := areFriends(
		ctx,
		uint(claims.Id),
		uint(req.GetToUserId()),
	)
	if err != nil {
		return messageChatError("query friend relation failed"), nil
	}
	if !friends {
		return messageChatError(
			"only mutual followers can view messages",
		), nil
	}

	records, err := db.GetMessagesBetweenUsers(
		ctx,
		uint(claims.Id),
		uint(req.GetToUserId()),
		req.GetPreMsgTime(),
	)
	if err != nil {
		return messageChatError("query messages failed"), nil
	}

	messageList := make(
		[]*messagepb.Message,
		0,
		len(records),
	)
	for _, record := range records {
		messageList = append(messageList, &messagepb.Message{
			Id:         int64(record.ID),
			ToUserId:   int64(record.ToUserID),
			FromUserId: int64(record.FromUserID),
			Content:    record.Content,
			CreateTime: record.CreatedAt.UnixMilli(),
		})
	}

	return &messagepb.MessageChatResponse{
		StatusCode:  0,
		StatusMsg:   "success",
		MessageList: messageList,
	}, nil
}

func (s *MessageServiceImpl) parseToken(
	token string,
) (*appjwt.CustomClaims, bool) {
	if s.jwtManager == nil || token == "" {
		return nil, false
	}

	claims, err := s.jwtManager.ParseToken(token)
	if err != nil || claims.Id <= 0 {
		return nil, false
	}

	return claims, true
}

func areFriends(
	ctx context.Context,
	userID uint,
	toUserID uint,
) (bool, error) {
	userFollowsTarget, err := appcache.IsFollowing(ctx, userID, toUserID)
	if err != nil {
		return false, err
	}
	if !userFollowsTarget {
		return false, nil
	}

	targetFollowsUser, err := appcache.IsFollowing(ctx, toUserID, userID)
	if err != nil {
		return false, err
	}

	return targetFollowsUser, nil
}

func messageActionError(message string) *messagepb.MessageActionResponse {
	return &messagepb.MessageActionResponse{
		StatusCode: -1,
		StatusMsg:  message,
	}
}

func messageChatError(message string) *messagepb.MessageChatResponse {
	return &messagepb.MessageChatResponse{
		StatusCode:  -1,
		StatusMsg:   message,
		MessageList: make([]*messagepb.Message, 0),
	}
}
