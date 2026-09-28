package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/dal/db"
	videopb "douyin/kitex/kitex_gen/video"
	appjwt "douyin/pkg/jwt"

	"gorm.io/plugin/dbresolver"
)

func TestPublishList(t *testing.T) {
	ctx := context.Background()

	testUser := &db.User{
		UserName: fmt.Sprintf(
			"publish_list_user_%d",
			time.Now().UnixNano(),
		),
		Password:        "test_password",
		Avatar:          "default0.png",
		BackgroundImage: "default_background.jpg",
	}

	if err := db.CreateUser(ctx, testUser); err != nil {
		t.Fatalf("create test user failed: %v", err)
	}

	defer func() {
		db.GetDB().
			Clauses(dbresolver.Write).
			Unscoped().
			Delete(&db.User{}, testUser.ID)
	}()

	testVideo := &db.Video{
		AuthorID: testUser.ID,
		PlayUrl:  "publish-list-test.mp4",
		CoverUrl: "publish-list-test.png",
		Title:    "publish list test",
	}

	if err := db.CreateVideo(ctx, testVideo); err != nil {
		t.Fatalf("create test video failed: %v", err)
	}

	defer func() {
		db.GetDB().
			Clauses(dbresolver.Write).
			Unscoped().
			Delete(&db.Video{}, testVideo.ID)
	}()

	signingKey := []byte("video-test-signing-key")
	jwtManager := appjwt.NewJWT(signingKey)
	token, err := jwtManager.CreateToken(appjwt.CustomClaims{Id: int64(testUser.ID)})
	if err != nil {
		t.Fatalf("create test token failed: %v", err)
	}

	service := NewVideoServiceImpl(signingKey, 50)

	response, err := service.PublishList(
		ctx,
		&videopb.PublishListRequest{
			UserId: int64(testUser.ID),
			Token:  token,
		},
	)
	if err != nil {
		t.Fatalf("publish list failed: %v", err)
	}

	if response.StatusCode != 0 {
		t.Fatalf(
			"publish list returned failure: %+v",
			response,
		)
	}

	if len(response.VideoList) != 1 {
		t.Fatalf(
			"expected one video, got %d",
			len(response.VideoList),
		)
	}

	result := response.VideoList[0]

	if result.Id != int64(testVideo.ID) {
		t.Fatalf(
			"video ID mismatch: want=%d got=%d",
			testVideo.ID,
			result.Id,
		)
	}

	if result.Title != testVideo.Title {
		t.Fatalf(
			"title mismatch: want=%s got=%s",
			testVideo.Title,
			result.Title,
		)
	}

	if result.Author == nil {
		t.Fatal("video author is nil")
	}

	if result.Author.Id != int64(testUser.ID) {
		t.Fatalf(
			"author ID mismatch: want=%d got=%d",
			testUser.ID,
			result.Author.Id,
		)
	}

	if result.PlayUrl == "" {
		t.Fatal("video play URL is empty")
	}

	if result.CoverUrl == "" {
		t.Fatal("video cover URL is empty")
	}
}
