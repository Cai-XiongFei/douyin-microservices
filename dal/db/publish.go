package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func CreateVideo(
	ctx context.Context,
	video *Video,
) error {
	return GetDB().
		Clauses(dbresolver.Write).
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			// 创建视频记录。
			if err := tx.Create(video).Error; err != nil {
				return err
			}

			// 用户作品数量加一。
			result := tx.
				Model(&User{}).
				Where("id = ?", video.AuthorID).
				Update(
					"work_count",
					gorm.Expr("work_count + ?", 1),
				)

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected != 1 {
				return errors.New("video author does not exist")
			}

			return nil
		})
}

func GetVideosByUserID(
	ctx context.Context,
	userID int64,
) ([]*Video, error) {
	videos := make([]*Video, 0)

	err := GetDB().
		Clauses(dbresolver.Read).
		WithContext(ctx).
		Where("author_id = ?", userID).
		Order("created_at DESC").
		Find(&videos).
		Error

	if err != nil {
		return nil, err
	}

	return videos, nil
}
