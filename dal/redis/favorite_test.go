package redis

import (
	"context"
	"testing"
	"time"
)

func TestFavoriteStatusCache(t *testing.T) {
	ctx := context.Background()

	userID := uint(time.Now().UnixNano())
	videoID := uint(10001)

	defer func() {
		_ = DeleteFavoriteStatus(ctx, userID, videoID)
	}()

	// 第一次查询时，缓存应该不存在。
	_, found, err := GetFavoriteStatus(
		ctx,
		userID,
		videoID,
	)
	if err != nil {
		t.Fatalf("查询收藏缓存失败：%v", err)
	}

	if found {
		t.Fatal("期望缓存不存在，但实际存在")
	}

	// 写入“已收藏”状态。
	err = SetFavoriteStatus(
		ctx,
		userID,
		videoID,
		true,
	)
	if err != nil {
		t.Fatalf("写入收藏缓存失败：%v", err)
	}

	// 再次查询，应该命中缓存。
	favorite, found, err := GetFavoriteStatus(
		ctx,
		userID,
		videoID,
	)
	if err != nil {
		t.Fatalf("查询收藏缓存失败：%v", err)
	}

	if !found {
		t.Fatal("期望命中缓存，但缓存不存在")
	}

	if !favorite {
		t.Fatal("期望收藏状态为 true，实际为 false")
	}
}
