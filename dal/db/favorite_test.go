package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestFavoriteAction(t *testing.T) {
	ctx := context.Background()
	database := GetDB()

	unique := time.Now().UnixNano()

	author := &User{
		UserName: fmt.Sprintf("favorite_author_%d", unique),
		Password: "test_password",
	}

	if err := database.WithContext(ctx).Create(author).Error; err != nil {
		t.Fatalf("create author failed: %v", err)
	}

	user := &User{
		UserName: fmt.Sprintf("favorite_user_%d", unique),
		Password: "test_password",
	}

	if err := database.WithContext(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	video := &Video{
		AuthorID: author.ID,
		PlayUrl:  "tests/favorite-video.mp4",
		CoverUrl: "tests/favorite-cover.jpg",
		Title:    "favorite test video",
	}

	if err := database.WithContext(ctx).Create(video).Error; err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	// 测试结束后清理本次创建的数据。
	t.Cleanup(func() {
		database.Unscoped().
			Where("user_id = ? AND video_id = ?", user.ID, video.ID).
			Delete(&Favorite{})

		database.Unscoped().Delete(&Video{}, video.ID)

		database.Unscoped().
			Where("id IN ?", []uint{author.ID, user.ID}).
			Delete(&User{})
	})

	// 第一次点赞。
	if err := FavoriteAction(ctx, user.ID, video.ID, 1); err != nil {
		t.Fatalf("favorite action failed: %v", err)
	}

	var relation Favorite
	if err := database.WithContext(ctx).
		Where("user_id = ? AND video_id = ?", user.ID, video.ID).
		First(&relation).
		Error; err != nil {
		t.Fatalf("favorite relation was not created: %v", err)
	}

	checkCounts := func(want uint) {
		t.Helper()

		var updatedVideo Video
		if err := database.WithContext(ctx).
			First(&updatedVideo, video.ID).
			Error; err != nil {
			t.Fatalf("query video failed: %v", err)
		}

		var updatedUser User
		if err := database.WithContext(ctx).
			First(&updatedUser, user.ID).
			Error; err != nil {
			t.Fatalf("query user failed: %v", err)
		}

		var updatedAuthor User
		if err := database.WithContext(ctx).
			First(&updatedAuthor, author.ID).
			Error; err != nil {
			t.Fatalf("query author failed: %v", err)
		}

		if updatedVideo.FavoriteCount != want {
			t.Fatalf(
				"video favorite count: got %d, want %d",
				updatedVideo.FavoriteCount,
				want,
			)
		}

		if updatedUser.FavoriteCount != want {
			t.Fatalf(
				"user favorite count: got %d, want %d",
				updatedUser.FavoriteCount,
				want,
			)
		}

		if updatedAuthor.TotalFavorited != want {
			t.Fatalf(
				"author total favorited: got %d, want %d",
				updatedAuthor.TotalFavorited,
				want,
			)
		}
	}

	checkCounts(1)

	// 重复点赞不应重复增加计数。
	if err := FavoriteAction(ctx, user.ID, video.ID, 1); err != nil {
		t.Fatalf("repeated favorite action failed: %v", err)
	}

	checkCounts(1)

	// 取消点赞。
	if err := FavoriteAction(ctx, user.ID, video.ID, 2); err != nil {
		t.Fatalf("cancel favorite failed: %v", err)
	}

	err := database.WithContext(ctx).
		Where("user_id = ? AND video_id = ?", user.ID, video.ID).
		First(&Favorite{}).
		Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("favorite relation still exists: %v", err)
	}

	checkCounts(0)
}
