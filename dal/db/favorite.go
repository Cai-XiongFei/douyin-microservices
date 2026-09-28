package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Favorite 表示一条用户点赞视频的关系。
type Favorite struct {
	gorm.Model

	UserID  uint `gorm:"not null;uniqueIndex:idx_user_video"`
	VideoID uint `gorm:"not null;uniqueIndex:idx_user_video;index"`
}

// FavoriteAction 保留给现有数据库测试使用。
func FavoriteAction(
	ctx context.Context,
	userID uint,
	videoID uint,
	actionType int32,
) error {
	return favoriteAction(
		ctx,
		userID,
		videoID,
		actionType,
		nil,
	)
}

// FavoriteActionWithOutbox 在业务事务中同时写入Outbox事件。
func FavoriteActionWithOutbox(
	ctx context.Context,
	userID uint,
	videoID uint,
	actionType int32,
	outboxEvent *OutboxEvent,
) error {
	if outboxEvent == nil {
		return errors.New("outbox event cannot be nil")
	}

	return favoriteAction(
		ctx,
		userID,
		videoID,
		actionType,
		outboxEvent,
	)
}

// FavoriteAction 执行点赞或者取消点赞。
// actionType：1 表示点赞，2 表示取消点赞。
func favoriteAction(
	ctx context.Context,
	userID uint,
	videoID uint,
	actionType int32,
	outboxEvent *OutboxEvent,
) error {
	if actionType != 1 && actionType != 2 {
		return errors.New("invalid favorite action type")
	}

	return GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 确认操作点赞的用户存在。
		var currentUser User
		if err := tx.First(&currentUser, userID).Error; err != nil {
			return fmt.Errorf("query current user failed: %w", err)
		}

		// 查询视频，同时获取视频作者 ID。
		var currentVideo Video
		if err := tx.First(&currentVideo, videoID).Error; err != nil {
			return fmt.Errorf("query video failed: %w", err)
		}

		var favorite Favorite
		err := tx.
			Where("user_id = ? AND video_id = ?", userID, videoID).
			First(&favorite).
			Error

		switch actionType {
		case 1:
			// 已经点赞过，直接返回成功，避免重复增加计数。
			if err == nil {
				return nil
			}

			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query favorite failed: %w", err)
			}

			favorite = Favorite{
				UserID:  userID,
				VideoID: videoID,
			}

			if err := tx.Create(&favorite).Error; err != nil {
				return fmt.Errorf("create favorite failed: %w", err)
			}

			// 视频点赞数 +1。
			if err := tx.
				Model(&Video{}).
				Where("id = ?", videoID).
				UpdateColumn(
					"favorite_count",
					gorm.Expr("favorite_count + ?", 1),
				).Error; err != nil {
				return fmt.Errorf("increase video favorite count failed: %w", err)
			}

			// 当前用户的点赞作品数 +1。
			if err := tx.
				Model(&User{}).
				Where("id = ?", userID).
				UpdateColumn(
					"favorite_count",
					gorm.Expr("favorite_count + ?", 1),
				).Error; err != nil {
				return fmt.Errorf("increase user favorite count failed: %w", err)
			}

			// 视频作者获得的总点赞数 +1。
			if err := tx.
				Model(&User{}).
				Where("id = ?", currentVideo.AuthorID).
				UpdateColumn(
					"total_favorited",
					gorm.Expr("total_favorited + ?", 1),
				).Error; err != nil {
				return fmt.Errorf("increase author total favorited failed: %w", err)
			}

		case 2:
			// 本来就没有点赞，直接返回成功。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			if err != nil {
				return fmt.Errorf("query favorite failed: %w", err)
			}

			// 使用硬删除，否则重新点赞时会与唯一索引冲突。
			if err := tx.Unscoped().Delete(&favorite).Error; err != nil {
				return fmt.Errorf("delete favorite failed: %w", err)
			}

			// 视频点赞数 -1。
			if err := tx.
				Model(&Video{}).
				Where("id = ? AND favorite_count > 0", videoID).
				UpdateColumn(
					"favorite_count",
					gorm.Expr("favorite_count - ?", 1),
				).Error; err != nil {
				return fmt.Errorf("decrease video favorite count failed: %w", err)
			}

			// 当前用户的点赞作品数 -1。
			if err := tx.
				Model(&User{}).
				Where("id = ? AND favorite_count > 0", userID).
				UpdateColumn(
					"favorite_count",
					gorm.Expr("favorite_count - ?", 1),
				).Error; err != nil {
				return fmt.Errorf("decrease user favorite count failed: %w", err)
			}

			// 视频作者获得的总点赞数 -1。
			if err := tx.
				Model(&User{}).
				Where(
					"id = ? AND total_favorited > 0",
					currentVideo.AuthorID,
				).
				UpdateColumn(
					"total_favorited",
					gorm.Expr("total_favorited - ?", 1),
				).Error; err != nil {
				return fmt.Errorf("decrease author total favorited failed: %w", err)
			}
		}
		if outboxEvent != nil {
			if err := CreateOutboxEvent(
				tx,
				outboxEvent,
			); err != nil {
				return fmt.Errorf(
					"create favorite outbox event failed: %w",
					err,
				)
			}
		}

		return nil
	})
}

func IsFavorite(
	ctx context.Context,
	userID uint,
	videoID uint,
) (bool, error) {
	var count int64

	err := GetDB().WithContext(ctx).
		Model(&Favorite{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func GetFavoriteVideosByUserID(
	ctx context.Context,
	userID uint,
) ([]*Video, error) {
	var videos []*Video

	err := GetDB().WithContext(ctx).
		Table("videos").
		Select("videos.*").
		Joins(
			"JOIN favorites ON favorites.video_id = videos.id",
		).
		Where("favorites.user_id = ?", userID).
		Where("favorites.deleted_at IS NULL").
		Where("videos.deleted_at IS NULL").
		Order("favorites.created_at DESC").
		Find(&videos).
		Error
	if err != nil {
		return nil, err
	}

	return videos, nil
}
