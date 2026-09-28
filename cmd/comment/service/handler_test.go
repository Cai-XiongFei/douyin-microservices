package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/dal/db"
	commentpb "douyin/kitex/kitex_gen/comment"
	appjwt "douyin/pkg/jwt"
)

func TestCommentActionAndList(t *testing.T) {
	ctx := context.Background()
	database := db.GetDB()

	unique := time.Now().UnixNano()

	author := &db.User{
		UserName: fmt.Sprintf(
			"comment_rpc_author_%d",
			unique,
		),
		Password: "test_password",
	}

	if err := database.Create(author).Error; err != nil {
		t.Fatalf("create author failed: %v", err)
	}

	commentUser := &db.User{
		UserName: fmt.Sprintf(
			"comment_rpc_user_%d",
			unique,
		),
		Password: "test_password",
	}

	if err := database.Create(commentUser).Error; err != nil {
		t.Fatalf("create comment user failed: %v", err)
	}

	otherUser := &db.User{
		UserName: fmt.Sprintf(
			"comment_rpc_other_%d",
			unique,
		),
		Password: "test_password",
	}

	if err := database.Create(otherUser).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}

	video := &db.Video{
		AuthorID: author.ID,
		PlayUrl:  "tests/comment-rpc-video.mp4",
		CoverUrl: "tests/comment-rpc-cover.jpg",
		Title:    "comment RPC test video",
	}

	if err := database.Create(video).Error; err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	t.Cleanup(func() {
		database.Unscoped().
			Where("video_id = ?", video.ID).
			Delete(&db.Comment{})

		database.Unscoped().
			Delete(&db.Video{}, video.ID)

		database.Unscoped().
			Where(
				"id IN ?",
				[]uint{
					author.ID,
					commentUser.ID,
					otherUser.ID,
				},
			).
			Delete(&db.User{})
	})

	signingKey := []byte("comment-test-signing-key")
	jwtManager := appjwt.NewJWT(signingKey)

	commentUserToken, err := jwtManager.CreateToken(
		appjwt.CustomClaims{
			Id: int64(commentUser.ID),
		},
	)
	if err != nil {
		t.Fatalf(
			"create comment user token failed: %v",
			err,
		)
	}

	otherUserToken, err := jwtManager.CreateToken(
		appjwt.CustomClaims{
			Id: int64(otherUser.ID),
		},
	)
	if err != nil {
		t.Fatalf(
			"create other user token failed: %v",
			err,
		)
	}

	handler := NewCommentServiceImpl(signingKey)
	commentText := "这是通过 RPC 发布的测试评论"

	// 1. 发布评论。
	createResponse, err := handler.CommentAction(
		ctx,
		&commentpb.CommentActionRequest{
			Token:       commentUserToken,
			VideoId:     int64(video.ID),
			ActionType:  1,
			CommentText: commentText,
		},
	)
	if err != nil {
		t.Fatalf(
			"create comment returned error: %v",
			err,
		)
	}

	if createResponse.StatusCode != 0 {
		t.Fatalf(
			"create comment failed: code=%d message=%s",
			createResponse.StatusCode,
			createResponse.StatusMsg,
		)
	}

	if createResponse.Comment == nil {
		t.Fatal("created comment is nil")
	}

	if createResponse.Comment.Content != commentText {
		t.Fatalf(
			"comment content: got %q, want %q",
			createResponse.Comment.Content,
			commentText,
		)
	}

	commentID := createResponse.Comment.Id

	// 2. 未登录用户查询评论列表。
	listResponse, err := handler.CommentList(
		ctx,
		&commentpb.CommentListRequest{
			VideoId: int64(video.ID),
			Token:   "",
		},
	)
	if err != nil {
		t.Fatalf(
			"comment list returned error: %v",
			err,
		)
	}

	if listResponse.StatusCode != 0 {
		t.Fatalf(
			"comment list failed: code=%d message=%s",
			listResponse.StatusCode,
			listResponse.StatusMsg,
		)
	}

	if len(listResponse.CommentList) != 1 {
		t.Fatalf(
			"comment list length: got %d, want 1",
			len(listResponse.CommentList),
		)
	}

	if listResponse.CommentList[0].Id != commentID {
		t.Fatalf(
			"comment ID: got %d, want %d",
			listResponse.CommentList[0].Id,
			commentID,
		)
	}

	if listResponse.CommentList[0].User == nil {
		t.Fatal("comment user is nil")
	}

	if listResponse.CommentList[0].User.Id !=
		int64(commentUser.ID) {
		t.Fatalf(
			"comment user ID: got %d, want %d",
			listResponse.CommentList[0].User.Id,
			commentUser.ID,
		)
	}

	// 3. 其他用户不能删除这条评论。
	otherDeleteResponse, err := handler.CommentAction(
		ctx,
		&commentpb.CommentActionRequest{
			Token:      otherUserToken,
			VideoId:    int64(video.ID),
			ActionType: 2,
			CommentId:  commentID,
		},
	)
	if err != nil {
		t.Fatalf(
			"other user delete returned error: %v",
			err,
		)
	}

	if otherDeleteResponse.StatusCode == 0 {
		t.Fatal("other user deleted the comment")
	}

	// 4. 评论创建者删除自己的评论。
	deleteResponse, err := handler.CommentAction(
		ctx,
		&commentpb.CommentActionRequest{
			Token:      commentUserToken,
			VideoId:    int64(video.ID),
			ActionType: 2,
			CommentId:  commentID,
		},
	)
	if err != nil {
		t.Fatalf(
			"delete comment returned error: %v",
			err,
		)
	}

	if deleteResponse.StatusCode != 0 {
		t.Fatalf(
			"delete comment failed: code=%d message=%s",
			deleteResponse.StatusCode,
			deleteResponse.StatusMsg,
		)
	}

	// 5. 删除后列表应为空。
	listResponse, err = handler.CommentList(
		ctx,
		&commentpb.CommentListRequest{
			VideoId: int64(video.ID),
		},
	)
	if err != nil {
		t.Fatalf(
			"comment list after delete returned error: %v",
			err,
		)
	}

	if len(listResponse.CommentList) != 0 {
		t.Fatalf(
			"comment list after delete: got %d, want 0",
			len(listResponse.CommentList),
		)
	}
}
