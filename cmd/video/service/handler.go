package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	userpb "douyin/kitex/kitex_gen/user"
	videopb "douyin/kitex/kitex_gen/video"
	appjwt "douyin/pkg/jwt"
	storage "douyin/pkg/minio"
	apprabbitmq "douyin/pkg/rabbitmq"
)

const defaultMaxVideoSizeMB int64 = 50

type VideoServiceImpl struct {
	jwtManager        *appjwt.JWT
	maxVideoSizeBytes int64
}

func NewVideoServiceImpl(
	signingKey []byte,
	maxVideoSizeMB int64,
) *VideoServiceImpl {
	if maxVideoSizeMB <= 0 {
		maxVideoSizeMB = defaultMaxVideoSizeMB
	}

	return &VideoServiceImpl{
		jwtManager: appjwt.NewJWT(signingKey),
		maxVideoSizeBytes: maxVideoSizeMB *
			1024 *
			1024,
	}
}

func (s *VideoServiceImpl) PublishList(
	ctx context.Context,
	req *videopb.PublishListRequest,
) (*videopb.PublishListResponse, error) {
	if req == nil {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "request cannot be nil",
			VideoList:  make([]*videopb.Video, 0),
		}, nil
	}

	if s.jwtManager == nil {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "JWT manager is not initialized",
			VideoList:  make([]*videopb.Video, 0),
		}, nil
	}

	claims, err := s.jwtManager.ParseToken(req.GetToken())
	if err != nil || claims.Id <= 0 {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "token is invalid or expired",
			VideoList:  make([]*videopb.Video, 0),
		}, nil
	}
	// 查询目标用户。
	author, err := appcache.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	if author == nil {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "user does not exist",
			VideoList:  make([]*videopb.Video, 0),
		}, nil
	}

	// 查询该用户发布的视频记录。
	videoRecords, err := db.GetVideosByUserID(
		ctx,
		req.GetUserId(),
	)
	if err != nil {
		return nil, err
	}

	// 生成作者头像和背景图临时地址。
	avatarURL, err := storage.GetFileTemporaryURL(
		storage.AvatarBucketName,
		author.Avatar,
	)
	if err != nil {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "failed to get avatar URL",
		}, nil
	}

	backgroundURL, err := storage.GetFileTemporaryURL(
		storage.BackgroundImageBucketName,
		author.BackgroundImage,
	)
	if err != nil {
		return &videopb.PublishListResponse{
			StatusCode: -1,
			StatusMsg:  "failed to get background image URL",
		}, nil
	}

	isFollow, err := appcache.IsFollowing(
		ctx,
		uint(claims.Id),
		author.ID,
	)
	if err != nil {
		return nil, err
	}

	videoList := make([]*videopb.Video, 0, len(videoRecords))

	for _, record := range videoRecords {
		playURL, err := storage.GetFileTemporaryURL(
			storage.VideoBucketName,
			record.PlayUrl,
		)
		if err != nil {
			return &videopb.PublishListResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get video URL",
			}, nil
		}

		coverURL, err := storage.GetFileTemporaryURL(
			storage.CoverBucketName,
			record.CoverUrl,
		)
		if err != nil {
			return &videopb.PublishListResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get cover URL",
			}, nil
		}

		isFavorite, err := appcache.IsFavorite(
			ctx,
			uint(claims.Id),
			record.ID,
		)
		if err != nil {
			return nil, err
		}

		videoList = append(videoList, &videopb.Video{
			Id: int64(record.ID),
			Author: &userpb.User{
				Id:              int64(author.ID),
				Name:            author.UserName,
				FollowCount:     int64(author.FollowingCount),
				FollowerCount:   int64(author.FollowerCount),
				IsFollow:        isFollow,
				Avatar:          avatarURL,
				BackgroundImage: backgroundURL,
				Signature:       author.Signature,
				TotalFavorited:  int64(author.TotalFavorited),
				WorkCount:       int64(author.WorkCount),
				FavoriteCount:   int64(author.FavoriteCount),
			},
			PlayUrl:       playURL,
			CoverUrl:      coverURL,
			FavoriteCount: int64(record.FavoriteCount),
			CommentCount:  int64(record.CommentCount),
			IsFavorite:    isFavorite,
			Title:         record.Title,
			ShareCount:    0,
		})
	}

	return &videopb.PublishListResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		VideoList:  videoList,
	}, nil
}

func (s *VideoServiceImpl) Feed(
	ctx context.Context,
	req *videopb.FeedRequest,
) (*videopb.FeedResponse, error) {
	if req == nil {
		return &videopb.FeedResponse{
			StatusCode: -1,
			StatusMsg:  "request cannot be nil",
			VideoList:  make([]*videopb.Video, 0),
		}, nil
	}
	const feedLimit = 30

	var currentUserID uint

	// Feed 允许未登录用户访问。
	// 但如果传了 token，就检查 token 是否有效。
	if req.GetToken() != "" {
		if s.jwtManager == nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "JWT manager is not initialized",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}

		claims, err := s.jwtManager.ParseToken(
			req.GetToken(),
		)
		if err != nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "token is invalid or expired",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}

		currentUserID = uint(claims.Id)
	}

	latestTime := req.GetLatestTime()

	if latestTime <= 0 {
		latestTime = time.Now().UnixMilli()
	}

	videoRecords, err := db.MGetVideos(
		ctx,
		feedLimit,
		&latestTime,
	)
	if err != nil {
		return nil, err
	}

	videoList := make(
		[]*videopb.Video,
		0,
		len(videoRecords),
	)

	for _, record := range videoRecords {
		author, err := appcache.GetUserByID(
			ctx,
			int64(record.AuthorID),
		)
		if err != nil {
			return nil, err
		}

		if author == nil {
			continue
		}

		playURL, err := storage.GetFileTemporaryURL(
			storage.VideoBucketName,
			record.PlayUrl,
		)
		if err != nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get video URL",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}

		coverURL, err := storage.GetFileTemporaryURL(
			storage.CoverBucketName,
			record.CoverUrl,
		)
		if err != nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get cover URL",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}

		avatarURL, err := storage.GetFileTemporaryURL(
			storage.AvatarBucketName,
			author.Avatar,
		)
		if err != nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get avatar URL",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}

		backgroundURL, err := storage.GetFileTemporaryURL(
			storage.BackgroundImageBucketName,
			author.BackgroundImage,
		)
		if err != nil {
			return &videopb.FeedResponse{
				StatusCode: -1,
				StatusMsg:  "failed to get background URL",
				VideoList:  make([]*videopb.Video, 0),
			}, nil
		}
		isFollow := false
		isFavorite := false

		if currentUserID > 0 {
			isFollow, err = appcache.IsFollowing(
				ctx,
				currentUserID,
				author.ID,
			)
			if err != nil {
				return nil, err
			}

			isFavorite, err = appcache.IsFavorite(
				ctx,
				currentUserID,
				record.ID,
			)
			if err != nil {
				return nil, err
			}
		}

		videoList = append(videoList, &videopb.Video{
			Id: int64(record.ID),
			Author: &userpb.User{
				Id:              int64(author.ID),
				Name:            author.UserName,
				FollowCount:     int64(author.FollowingCount),
				FollowerCount:   int64(author.FollowerCount),
				IsFollow:        isFollow,
				Avatar:          avatarURL,
				BackgroundImage: backgroundURL,
				Signature:       author.Signature,
				TotalFavorited:  int64(author.TotalFavorited),
				WorkCount:       int64(author.WorkCount),
				FavoriteCount:   int64(author.FavoriteCount),
			},
			PlayUrl:       playURL,
			CoverUrl:      coverURL,
			FavoriteCount: int64(record.FavoriteCount),
			CommentCount:  int64(record.CommentCount),
			IsFavorite:    isFavorite,
			Title:         record.Title,
			ShareCount:    0,
		})
	}

	nextTime := latestTime

	if len(videoRecords) > 0 {
		nextTime = videoRecords[len(videoRecords)-1].
			CreatedAt.UnixMilli()
	}

	return &videopb.FeedResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		VideoList:  videoList,
		NextTime:   nextTime,
	}, nil
}

func (s *VideoServiceImpl) PublishAction(
	ctx context.Context,
	req *videopb.PublishActionRequest,
) (*videopb.PublishActionResponse, error) {
	if req == nil {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "request cannot be nil",
		}, nil
	}

	if s.jwtManager == nil {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "JWT manager is not initialized",
		}, nil
	}

	claims, err := s.jwtManager.ParseToken(req.GetToken())
	if err != nil || claims.Id <= 0 {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "token is invalid or expired",
		}, nil
	}

	title := strings.TrimSpace(req.GetTitle())

	if title == "" {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "video title cannot be empty",
		}, nil
	}

	if utf8.RuneCountInString(title) > 32 {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "video title cannot exceed 32 characters",
		}, nil
	}

	videoData := req.GetData()

	if len(videoData) == 0 {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "video data cannot be empty",
		}, nil
	}

	if int64(len(videoData)) > s.maxVideoSizeBytes {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg: fmt.Sprintf(
				"video size cannot exceed %d MB",
				s.maxVideoSizeBytes/(1024*1024),
			),
		}, nil
	}

	// 提前确认 token 中的用户确实存在。
	author, err := appcache.GetUserByID(ctx, claims.Id)
	if err != nil {
		return nil, err
	}

	if author == nil {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "video author does not exist",
		}, nil
	}

	// 对象名称不包含用户输入的标题，避免标题中的特殊字符成为对象路径。
	timestamp := time.Now().UnixNano()

	videoName := fmt.Sprintf(
		"%d_%d.mp4",
		claims.Id,
		timestamp,
	)

	coverName := fmt.Sprintf(
		"%d_%d.jpg",
		claims.Id,
		timestamp,
	)

	// 上传视频，并通过 FFmpeg 生成、上传封面。
	if err := VideoPublish(
		videoData,
		videoName,
		coverName,
	); err != nil {
		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "upload video or cover failed",
		}, nil
	}

	videoRecord := &db.Video{
		AuthorID: uint(claims.Id),
		PlayUrl:  videoName,
		CoverUrl: coverName,
		Title:    title,
	}

	// CreateVideo 内部使用事务：
	// 创建视频记录 + 用户作品数量加一。
	if err := db.CreateVideo(ctx, videoRecord); err != nil {
		// 数据库失败时清理已经上传的 MinIO 文件。
		_ = storage.DeleteFile(
			storage.VideoBucketName,
			videoName,
		)

		_ = storage.DeleteFile(
			storage.CoverBucketName,
			coverName,
		)

		return &videopb.PublishActionResponse{
			StatusCode: -1,
			StatusMsg:  "create video record failed",
		}, nil
	}

	event := apprabbitmq.NewVideoPublishedEvent(videoRecord.ID, videoRecord.AuthorID, videoRecord.CreatedAt)
	publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	publishErr := apprabbitmq.PublishVideoPublished(publishCtx, event)
	cancel()
	if publishErr != nil {
		log.Printf("publish video event failed, fallback to direct user-cache invalidation: %v", publishErr)
		appcache.InvalidateUser(ctx, videoRecord.AuthorID)
	}

	return &videopb.PublishActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
	}, nil
}
