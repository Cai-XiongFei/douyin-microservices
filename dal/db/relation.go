package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Relation 表示一条用户关注关系：UserID 关注了 ToUserID。
type Relation struct {
	gorm.Model

	UserID   uint `gorm:"not null;uniqueIndex:idx_user_to_user;index"`
	ToUserID uint `gorm:"not null;uniqueIndex:idx_user_to_user;index"`
}

func (Relation) TableName() string {
	return "relations"
}

// RelationAction 执行关注或取消关注。
// actionType=1 表示关注，actionType=2 表示取消关注。
func RelationAction(
	ctx context.Context,
	userID uint,
	toUserID uint,
	actionType int32,
) error {
	if userID == 0 || toUserID == 0 {
		return errors.New("user id must be greater than zero")
	}
	if userID == toUserID {
		return errors.New("cannot follow yourself")
	}
	if actionType != 1 && actionType != 2 {
		return errors.New("invalid relation action type")
	}

	return GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var currentUser User
		if err := tx.First(&currentUser, userID).Error; err != nil {
			return fmt.Errorf("query current user failed: %w", err)
		}

		var targetUser User
		if err := tx.First(&targetUser, toUserID).Error; err != nil {
			return fmt.Errorf("query target user failed: %w", err)
		}

		var relation Relation
		err := tx.Where(
			"user_id = ? AND to_user_id = ?",
			userID,
			toUserID,
		).First(&relation).Error

		switch actionType {
		case 1:
			// 已经关注时直接返回成功，避免重复增加计数。
			if err == nil {
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query relation failed: %w", err)
			}

			relation = Relation{UserID: userID, ToUserID: toUserID}
			if err := tx.Create(&relation).Error; err != nil {
				return fmt.Errorf("create relation failed: %w", err)
			}

			if err := tx.Model(&User{}).
				Where("id = ?", userID).
				UpdateColumn(
					"following_count",
					gorm.Expr("following_count + ?", 1),
				).Error; err != nil {
				return fmt.Errorf("increase following count failed: %w", err)
			}

			if err := tx.Model(&User{}).
				Where("id = ?", toUserID).
				UpdateColumn(
					"follower_count",
					gorm.Expr("follower_count + ?", 1),
				).Error; err != nil {
				return fmt.Errorf("increase follower count failed: %w", err)
			}

		case 2:
			// 本来就没有关注时，也按取消成功处理。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("query relation failed: %w", err)
			}

			// 使用硬删除，避免重新关注时与唯一索引冲突。
			if err := tx.Unscoped().Delete(&relation).Error; err != nil {
				return fmt.Errorf("delete relation failed: %w", err)
			}

			if err := tx.Model(&User{}).
				Where("id = ? AND following_count > 0", userID).
				UpdateColumn(
					"following_count",
					gorm.Expr("following_count - ?", 1),
				).Error; err != nil {
				return fmt.Errorf("decrease following count failed: %w", err)
			}

			if err := tx.Model(&User{}).
				Where("id = ? AND follower_count > 0", toUserID).
				UpdateColumn(
					"follower_count",
					gorm.Expr("follower_count - ?", 1),
				).Error; err != nil {
				return fmt.Errorf("decrease follower count failed: %w", err)
			}
		}

		return nil
	})
}

// IsFollowing 判断 userID 是否关注了 toUserID。
func IsFollowing(
	ctx context.Context,
	userID uint,
	toUserID uint,
) (bool, error) {
	if userID == 0 || toUserID == 0 {
		return false, nil
	}

	var count int64
	err := GetDB().WithContext(ctx).
		Model(&Relation{}).
		Where("user_id = ? AND to_user_id = ?", userID, toUserID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetFollowingUsersByUserID 查询指定用户关注的人。
func GetFollowingUsersByUserID(
	ctx context.Context,
	userID uint,
) ([]*User, error) {
	users := make([]*User, 0)
	err := GetDB().WithContext(ctx).
		Table("users").
		Select("users.*").
		Joins("JOIN relations ON relations.to_user_id = users.id").
		Where("relations.user_id = ?", userID).
		Where("relations.deleted_at IS NULL").
		Where("users.deleted_at IS NULL").
		Order("relations.created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetFollowerUsersByUserID 查询关注指定用户的人。
func GetFollowerUsersByUserID(
	ctx context.Context,
	userID uint,
) ([]*User, error) {
	users := make([]*User, 0)
	err := GetDB().WithContext(ctx).
		Table("users").
		Select("users.*").
		Joins("JOIN relations ON relations.user_id = users.id").
		Where("relations.to_user_id = ?", userID).
		Where("relations.deleted_at IS NULL").
		Where("users.deleted_at IS NULL").
		Order("relations.created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetFriendUsersByUserID 查询和指定用户互相关注的用户。
func GetFriendUsersByUserID(
	ctx context.Context,
	userID uint,
) ([]*User, error) {
	users := make([]*User, 0)
	err := GetDB().WithContext(ctx).
		Table("users").
		Select("users.*").
		Joins(
			"JOIN relations AS outgoing ON outgoing.to_user_id = users.id",
		).
		Joins(
			"JOIN relations AS incoming ON incoming.user_id = users.id",
		).
		Where("outgoing.user_id = ?", userID).
		Where("incoming.to_user_id = ?", userID).
		Where("outgoing.deleted_at IS NULL").
		Where("incoming.deleted_at IS NULL").
		Where("users.deleted_at IS NULL").
		Order("outgoing.created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}
