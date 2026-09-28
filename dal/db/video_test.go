package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/plugin/dbresolver"
)

func TestCreateAndQueryVideo(t *testing.T) {
	ctx := context.Background()

	testUser := &User{
		UserName: fmt.Sprintf(
			"video_test_user_%d",
			time.Now().UnixNano(),
		),
		Password: "test_password",
		Avatar:   "default0.png",
	}

	if err := CreateUser(ctx, testUser); err != nil {
		t.Fatalf("create test user failed: %v", err)
	}

	defer func() {
		GetDB().
			Clauses(dbresolver.Write).
			Unscoped().
			Delete(&User{}, testUser.ID)
	}()

	testVideo := &Video{
		AuthorID: testUser.ID,
		PlayUrl:  "test-video.mp4",
		CoverUrl: "test-cover.png",
		Title:    "test video",
	}

	if err := CreateVideo(ctx, testVideo); err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	defer func() {
		GetDB().
			Clauses(dbresolver.Write).
			Unscoped().
			Delete(&Video{}, testVideo.ID)
	}()

	if testVideo.ID == 0 {
		t.Fatal("video ID was not generated")
	}

	result, err := GetVideoByID(
		ctx,
		int64(testVideo.ID),
	)
	if err != nil {
		t.Fatalf("get video by ID failed: %v", err)
	}

	if result == nil {
		t.Fatal("created video was not found")
	}

	if result.Title != testVideo.Title {
		t.Fatalf(
			"title mismatch: want=%s got=%s",
			testVideo.Title,
			result.Title,
		)
	}

	videos, err := GetVideosByUserID(
		ctx,
		int64(testUser.ID),
	)
	if err != nil {
		t.Fatalf("get videos by user ID failed: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf(
			"expected one video, got %d",
			len(videos),
		)
	}

	updatedUser, err := GetUserByID(
		ctx,
		int64(testUser.ID),
	)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}

	if updatedUser.WorkCount != 1 {
		t.Fatalf(
			"expected work count 1, got %d",
			updatedUser.WorkCount,
		)
	}
}
