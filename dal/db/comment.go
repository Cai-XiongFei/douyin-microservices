package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Comment 表示一条视频评论。
type Comment struct {
	gorm.Model

	VideoID uint   `gorm:"not null;index"`
	UserID  uint   `gorm:"not null;index"`
	Content string `gorm:"type:varchar(255);not null"`
}

// CreateComment 创建评论，并把视频评论数加一。
func CreateComment(
	ctx context.Context,
	userID uint,
	videoID uint,
	content string,
) (*Comment, error) {
	comment := &Comment{
		UserID:  userID,
		VideoID: videoID,
		Content: content,
	}

	err := GetDB().
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			// 确认发表评论的用户存在。
			var currentUser User
			if err := tx.First(
				&currentUser,
				userID,
			).Error; err != nil {
				return fmt.Errorf(
					"query comment user failed: %w",
					err,
				)
			}

			// 确认被评论的视频存在。
			var currentVideo Video
			if err := tx.First(
				&currentVideo,
				videoID,
			).Error; err != nil {
				return fmt.Errorf(
					"query comment video failed: %w",
					err,
				)
			}

			if err := tx.Create(comment).Error; err != nil {
				return fmt.Errorf(
					"create comment failed: %w",
					err,
				)
			}

			// 视频评论数加一。
			if err := tx.
				Model(&Video{}).
				Where("id = ?", videoID).
				UpdateColumn(
					"comment_count",
					gorm.Expr("comment_count + ?", 1),
				).
				Error; err != nil {
				return fmt.Errorf(
					"increase video comment count failed: %w",
					err,
				)
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return comment, nil
}

// DeleteComment 删除评论，并把视频评论数减一。
// 只有评论创建者才能删除自己的评论。
func DeleteComment(
	ctx context.Context,
	userID uint,
	videoID uint,
	commentID uint,
) error {
	return GetDB().
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			var comment Comment

			// 同时使用评论ID、用户ID和视频ID进行查询，
			// 从数据库层保证只能删除自己的评论。
			if err := tx.
				Where(
					"id = ? AND user_id = ? AND video_id = ?",
					commentID,
					userID,
					videoID,
				).
				First(&comment).
				Error; err != nil {
				return fmt.Errorf(
					"query comment failed: %w",
					err,
				)
			}

			if err := tx.Delete(&comment).Error; err != nil {
				return fmt.Errorf(
					"delete comment failed: %w",
					err,
				)
			}

			// 防止异常情况下评论数减成负数。
			if err := tx.
				Model(&Video{}).
				Where(
					"id = ? AND comment_count > 0",
					videoID,
				).
				UpdateColumn(
					"comment_count",
					gorm.Expr("comment_count - ?", 1),
				).
				Error; err != nil {
				return fmt.Errorf(
					"decrease video comment count failed: %w",
					err,
				)
			}

			return nil
		})
}

// GetCommentsByVideoID 查询视频评论，最新评论排在前面。
func GetCommentsByVideoID(
	ctx context.Context,
	videoID uint,
) ([]*Comment, error) {
	comments := make([]*Comment, 0)

	err := GetDB().
		WithContext(ctx).
		Where("video_id = ?", videoID).
		Order("created_at DESC").
		Find(&comments).
		Error
	if err != nil {
		return nil, err
	}

	return comments, nil
}
