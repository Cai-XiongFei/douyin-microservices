package db

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type Video struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"not null;index:idx_created_at"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Author   User `gorm:"foreignKey:AuthorID"`
	AuthorID uint `gorm:"index:idx_author_id;not null"`

	PlayUrl  string `gorm:"type:varchar(255);not null"`
	CoverUrl string `gorm:"type:varchar(255)"`

	FavoriteCount uint `gorm:"default:0;not null"`
	CommentCount  uint `gorm:"default:0;not null"`

	Title string `gorm:"type:varchar(50);not null"`
}

func (Video) TableName() string {
	return "videos"
}

func MGetVideos(
	ctx context.Context,
	limit int,
	latestTime *int64,
) ([]*Video, error) {
	videos := make([]*Video, 0)

	if latestTime == nil || *latestTime == 0 {
		now := time.Now().UnixMilli()
		latestTime = &now
	}

	err := GetDB().
		Clauses(dbresolver.Read).
		WithContext(ctx).
		Where(
			"created_at < ?",
			time.UnixMilli(*latestTime),
		).
		Order("created_at DESC").
		Limit(limit).
		Find(&videos).
		Error

	if err != nil {
		return nil, err
	}

	return videos, nil
}

func GetVideoByID(
	ctx context.Context,
	videoID int64,
) (*Video, error) {
	result := new(Video)

	err := GetDB().
		Clauses(dbresolver.Read).
		WithContext(ctx).
		First(result, videoID).
		Error

	if err == nil {
		return result, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return nil, err
}
