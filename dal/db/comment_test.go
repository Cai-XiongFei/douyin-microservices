package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCommentCreateListAndDelete(t *testing.T) {
	ctx := context.Background()
	database := GetDB()

	unique := time.Now().UnixNano()

	author := &User{
		UserName: fmt.Sprintf("comment_author_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(author).Error; err != nil {
		t.Fatalf("create author failed: %v", err)
	}

	commentUser := &User{
		UserName: fmt.Sprintf("comment_user_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(commentUser).Error; err != nil {
		t.Fatalf("create comment user failed: %v", err)
	}

	otherUser := &User{
		UserName: fmt.Sprintf("comment_other_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(otherUser).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}

	video := &Video{
		AuthorID: author.ID,
		PlayUrl:  "tests/comment-video.mp4",
		CoverUrl: "tests/comment-cover.jpg",
		Title:    "comment test video",
	}

	if err := database.Create(video).Error; err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	t.Cleanup(func() {
		database.Unscoped().
			Where("video_id = ?", video.ID).
			Delete(&Comment{})

		database.Unscoped().
			Delete(&Video{}, video.ID)

		database.Unscoped().
			Where(
				"id IN ?",
				[]uint{
					author.ID,
					commentUser.ID,
					otherUser.ID,
				},
			).
			Delete(&User{})
	})

	// 1. 发布评论。
	comment, err := CreateComment(
		ctx,
		commentUser.ID,
		video.ID,
		"这是一条测试评论",
	)
	if err != nil {
		t.Fatalf("create comment failed: %v", err)
	}

	if comment.ID == 0 {
		t.Fatal("comment ID was not generated")
	}

	if comment.Content != "这是一条测试评论" {
		t.Fatalf(
			"comment content: got %q",
			comment.Content,
		)
	}

	checkVideoCommentCount := func(want uint) {
		t.Helper()

		var updatedVideo Video
		if err := database.
			First(&updatedVideo, video.ID).
			Error; err != nil {
			t.Fatalf("query video failed: %v", err)
		}

		if updatedVideo.CommentCount != want {
			t.Fatalf(
				"video comment count: got %d, want %d",
				updatedVideo.CommentCount,
				want,
			)
		}
	}

	checkVideoCommentCount(1)

	// 2. 查询评论列表。
	comments, err := GetCommentsByVideoID(
		ctx,
		video.ID,
	)
	if err != nil {
		t.Fatalf("query comments failed: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf(
			"comment list length: got %d, want 1",
			len(comments),
		)
	}

	if comments[0].ID != comment.ID {
		t.Fatalf(
			"comment ID: got %d, want %d",
			comments[0].ID,
			comment.ID,
		)
	}

	// 3. 其他用户不能删除这条评论。
	err = DeleteComment(
		ctx,
		otherUser.ID,
		video.ID,
		comment.ID,
	)
	if err == nil {
		t.Fatal("other user deleted the comment")
	}

	checkVideoCommentCount(1)

	// 4. 评论创建者删除自己的评论。
	err = DeleteComment(
		ctx,
		commentUser.ID,
		video.ID,
		comment.ID,
	)
	if err != nil {
		t.Fatalf("delete comment failed: %v", err)
	}

	checkVideoCommentCount(0)

	// 5. 软删除后的评论不应出现在列表中。
	comments, err = GetCommentsByVideoID(
		ctx,
		video.ID,
	)
	if err != nil {
		t.Fatalf(
			"query comments after delete failed: %v",
			err,
		)
	}

	if len(comments) != 0 {
		t.Fatalf(
			"comment list after delete: got %d, want 0",
			len(comments),
		)
	}
}
