package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/dal/db"
	messagepb "douyin/kitex/kitex_gen/message"
	appjwt "douyin/pkg/jwt"

	jwtlib "github.com/golang-jwt/jwt"
)

func TestMessageActionAndChat(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	userA := &db.User{
		UserName: fmt.Sprintf("message_a_%d", suffix),
		Password: "test-password",
		Avatar:   "default1.png",
	}
	userB := &db.User{
		UserName: fmt.Sprintf("message_b_%d", suffix),
		Password: "test-password",
		Avatar:   "default2.png",
	}

	if err := db.CreateUser(ctx, userA); err != nil {
		t.Fatalf("create user A failed: %v", err)
	}
	if err := db.CreateUser(ctx, userB); err != nil {
		t.Fatalf("create user B failed: %v", err)
	}

	userIDs := []uint{userA.ID, userB.ID}
	defer func() {
		database := db.GetDB()
		database.Unscoped().
			Where("from_user_id IN ? OR to_user_id IN ?", userIDs, userIDs).
			Delete(&db.Message{})
		database.Unscoped().
			Where("user_id IN ? OR to_user_id IN ?", userIDs, userIDs).
			Delete(&db.Relation{})
		database.Unscoped().
			Where("id IN ?", userIDs).
			Delete(&db.User{})
	}()

	if err := db.RelationAction(ctx, userA.ID, userB.ID, 1); err != nil {
		t.Fatalf("A follows B failed: %v", err)
	}
	if err := db.RelationAction(ctx, userB.ID, userA.ID, 1); err != nil {
		t.Fatalf("B follows A failed: %v", err)
	}

	jwtManager := appjwt.NewJWT([]byte("signingKey"))
	createToken := func(userID uint) string {
		token, err := jwtManager.CreateToken(appjwt.CustomClaims{
			Id: int64(userID),
			StandardClaims: jwtlib.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
			},
		})
		if err != nil {
			t.Fatalf("create token failed: %v", err)
		}
		return token
	}

	service := NewMessageServiceImpl([]byte("signingKey"))

	actionResponse, err := service.MessageAction(
		ctx,
		&messagepb.MessageActionRequest{
			Token:      createToken(userA.ID),
			ToUserId:   int64(userB.ID),
			ActionType: 1,
			Content:    "hello from A",
		},
	)
	if err != nil {
		t.Fatalf("send message failed: %v", err)
	}
	if actionResponse.StatusCode != 0 {
		t.Fatalf("send message response failed: %s", actionResponse.StatusMsg)
	}

	chatResponse, err := service.MessageChat(
		ctx,
		&messagepb.MessageChatRequest{
			Token:      createToken(userB.ID),
			ToUserId:   int64(userA.ID),
			PreMsgTime: 0,
		},
	)
	if err != nil {
		t.Fatalf("query chat failed: %v", err)
	}
	if chatResponse.StatusCode != 0 {
		t.Fatalf("query chat response failed: %s", chatResponse.StatusMsg)
	}
	if len(chatResponse.MessageList) != 1 {
		t.Fatalf("expected one message, got %d", len(chatResponse.MessageList))
	}
	if chatResponse.MessageList[0].Content != "hello from A" {
		t.Fatalf(
			"unexpected message content: %s",
			chatResponse.MessageList[0].Content,
		)
	}
}
