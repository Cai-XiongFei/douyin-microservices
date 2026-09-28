package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/dal/db"
	favoritepb "douyin/kitex/kitex_gen/favorite"
	appjwt "douyin/pkg/jwt"
)

func TestFavoriteAction(t *testing.T) {
	ctx := context.Background()
	database := db.GetDB()

	unique := time.Now().UnixNano()

	author := &db.User{
		UserName: fmt.Sprintf("rpc_favorite_author_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(author).Error; err != nil {
		t.Fatalf("create author failed: %v", err)
	}

	currentUser := &db.User{
		UserName: fmt.Sprintf("rpc_favorite_user_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(currentUser).Error; err != nil {
		t.Fatalf("create current user failed: %v", err)
	}

	currentVideo := &db.Video{
		AuthorID: author.ID,
		PlayUrl:  "tests/rpc-favorite-video.mp4",
		CoverUrl: "tests/rpc-favorite-cover.jpg",
		Title:    "rpc favorite test",
	}

	if err := database.Create(currentVideo).Error; err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	t.Cleanup(func() {
		database.Unscoped().
			Where(
				"user_id = ? AND video_id = ?",
				currentUser.ID,
				currentVideo.ID,
			).
			Delete(&db.Favorite{})

		database.Unscoped().Delete(&db.Video{}, currentVideo.ID)

		database.Unscoped().
			Where("id IN ?", []uint{author.ID, currentUser.ID}).
			Delete(&db.User{})
	})

	signingKey := []byte("favorite-test-signing-key")

	jwtManager := appjwt.NewJWT(signingKey)

	claims := appjwt.CustomClaims{
		Id: int64(currentUser.ID),
	}

	token, err := jwtManager.CreateToken(claims)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	handler := NewFavoriteServiceImpl(signingKey)

	// 点赞。
	likeResponse, err := handler.FavoriteAction(
		ctx,
		&favoritepb.FavoriteActionRequest{
			Token:      token,
			VideoId:    int64(currentVideo.ID),
			ActionType: 1,
		},
	)
	if err != nil {
		t.Fatalf("favorite RPC returned error: %v", err)
	}

	if likeResponse.StatusCode != 0 {
		t.Fatalf(
			"favorite failed: code=%d message=%s",
			likeResponse.StatusCode,
			likeResponse.StatusMsg,
		)
	}

	// 取消点赞。
	cancelResponse, err := handler.FavoriteAction(
		ctx,
		&favoritepb.FavoriteActionRequest{
			Token:      token,
			VideoId:    int64(currentVideo.ID),
			ActionType: 2,
		},
	)
	if err != nil {
		t.Fatalf("cancel favorite RPC returned error: %v", err)
	}

	if cancelResponse.StatusCode != 0 {
		t.Fatalf(
			"cancel favorite failed: code=%d message=%s",
			cancelResponse.StatusCode,
			cancelResponse.StatusMsg,
		)
	}
}

func TestFavoriteActionRejectsInvalidToken(t *testing.T) {
	handler := NewFavoriteServiceImpl(
		[]byte("favorite-test-signing-key"),
	)

	response, err := handler.FavoriteAction(
		context.Background(),
		&favoritepb.FavoriteActionRequest{
			Token:      "invalid-token",
			VideoId:    1,
			ActionType: 1,
		},
	)
	if err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}

	if response.StatusCode == 0 {
		t.Fatal("invalid token was accepted")
	}
}
func TestFavoriteList(t *testing.T) {
	ctx := context.Background()
	database := db.GetDB()

	unique := time.Now().UnixNano()

	author := &db.User{
		UserName:        fmt.Sprintf("favorite_list_author_%d", unique),
		Password:        "test_password",
		Avatar:          "",
		BackgroundImage: "",
	}

	if err := database.Create(author).Error; err != nil {
		t.Fatalf("create author failed: %v", err)
	}

	currentUser := &db.User{
		UserName: fmt.Sprintf("favorite_list_user_%d", unique),
		Password: "test_password",
	}

	if err := database.Create(currentUser).Error; err != nil {
		t.Fatalf("create current user failed: %v", err)
	}

	currentVideo := &db.Video{
		AuthorID: author.ID,
		PlayUrl:  "tests/favorite-list-video.mp4",
		CoverUrl: "tests/favorite-list-cover.jpg",
		Title:    "favorite list test video",
	}

	if err := database.Create(currentVideo).Error; err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	t.Cleanup(func() {
		database.Unscoped().
			Where(
				"user_id = ? AND video_id = ?",
				currentUser.ID,
				currentVideo.ID,
			).
			Delete(&db.Favorite{})

		database.Unscoped().
			Delete(&db.Video{}, currentVideo.ID)

		database.Unscoped().
			Where("id IN ?", []uint{author.ID, currentUser.ID}).
			Delete(&db.User{})
	})

	signingKey := []byte("favorite-list-test-key")

	jwtManager := appjwt.NewJWT(signingKey)

	claims := appjwt.CustomClaims{
		Id: int64(currentUser.ID),
	}

	token, err := jwtManager.CreateToken(claims)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}

	handler := NewFavoriteServiceImpl(signingKey)

	// 先点赞视频。
	actionResponse, err := handler.FavoriteAction(
		ctx,
		&favoritepb.FavoriteActionRequest{
			Token:      token,
			VideoId:    int64(currentVideo.ID),
			ActionType: 1,
		},
	)
	if err != nil {
		t.Fatalf("favorite action returned error: %v", err)
	}

	if actionResponse.StatusCode != 0 {
		t.Fatalf(
			"favorite action failed: code=%d message=%s",
			actionResponse.StatusCode,
			actionResponse.StatusMsg,
		)
	}

	// 再查询当前用户的点赞列表。
	listResponse, err := handler.FavoriteList(
		ctx,
		&favoritepb.FavoriteListRequest{
			UserId: int64(currentUser.ID),
			Token:  token,
		},
	)
	if err != nil {
		t.Fatalf("favorite list returned error: %v", err)
	}

	if listResponse.StatusCode != 0 {
		t.Fatalf(
			"favorite list failed: code=%d message=%s",
			listResponse.StatusCode,
			listResponse.StatusMsg,
		)
	}

	if len(listResponse.VideoList) != 1 {
		t.Fatalf(
			"video list length: got %d, want 1",
			len(listResponse.VideoList),
		)
	}

	resultVideo := listResponse.VideoList[0]

	if resultVideo.Id != int64(currentVideo.ID) {
		t.Fatalf(
			"video id: got %d, want %d",
			resultVideo.Id,
			currentVideo.ID,
		)
	}

	if resultVideo.Title != currentVideo.Title {
		t.Fatalf(
			"video title: got %q, want %q",
			resultVideo.Title,
			currentVideo.Title,
		)
	}

	if resultVideo.Author == nil {
		t.Fatal("video author is nil")
	}

	if resultVideo.Author.Id != int64(author.ID) {
		t.Fatalf(
			"author id: got %d, want %d",
			resultVideo.Author.Id,
			author.ID,
		)
	}

	if !resultVideo.IsFavorite {
		t.Fatal("is_favorite: got false, want true")
	}

	if resultVideo.PlayUrl == "" {
		t.Fatal("play_url is empty")
	}

	if resultVideo.CoverUrl == "" {
		t.Fatal("cover_url is empty")
	}
}
