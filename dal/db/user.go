package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type User struct {
	gorm.Model

	UserName string `gorm:"index:idx_username,unique;type:varchar(40);not null"`
	Password string `gorm:"type:varchar(256);not null"`

	FollowingCount uint `gorm:"default:0;not null"`
	FollowerCount  uint `gorm:"default:0;not null"`

	Avatar          string `gorm:"type:varchar(256)"`
	BackgroundImage string `gorm:"type:varchar(256);default:default_background.jpg"`

	WorkCount      uint `gorm:"default:0;not null"`
	FavoriteCount  uint `gorm:"default:0;not null"`
	TotalFavorited uint `gorm:"default:0;not null"`

	Signature string `gorm:"type:varchar(256)"`
}

func (User) TableName() string {
	return "users"
}

func GetUserByID(ctx context.Context, userID int64) (*User, error) {
	result := new(User)

	err := GetDB().
		Clauses(dbresolver.Read).
		WithContext(ctx).
		First(result, userID).
		Error

	if err == nil {
		return result, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return nil, err
}

func GetUserByName(ctx context.Context, username string) (*User, error) {
	result := new(User)

	err := GetDB().
		Clauses(dbresolver.Read).
		WithContext(ctx).
		Where("user_name = ?", username).
		First(result).
		Error

	if err == nil {
		return result, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return nil, err
}

func CreateUser(ctx context.Context, user *User) error {
	return GetDB().
		Clauses(dbresolver.Write).
		WithContext(ctx).
		Create(user).
		Error
}
