package service

import (
	"context"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	favoritepb "douyin/kitex/kitex_gen/favorite"
	userpb "douyin/kitex/kitex_gen/user"
	videopb "douyin/kitex/kitex_gen/video"
	appjwt "douyin/pkg/jwt"
	appminio "douyin/pkg/minio"
	apprabbitmq "douyin/pkg/rabbitmq"
)

type FavoriteServiceImpl struct {
	jwtManager *appjwt.JWT
}

func NewFavoriteServiceImpl(signingKey []byte) *FavoriteServiceImpl {
	return &FavoriteServiceImpl{
		jwtManager: appjwt.NewJWT(signingKey),
	}
}

// FavoriteAction 处理点赞和取消点赞。
// action_type=1 表示点赞，action_type=2 表示取消点赞。
func (s *FavoriteServiceImpl) FavoriteAction(
	ctx context.Context,
	req *favoritepb.FavoriteActionRequest,
) (*favoritepb.FavoriteActionResponse, error) {
	if req == nil {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "request cannot be nil",
		}, nil
	}

	if req.Token == "" {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "token cannot be empty",
		}, nil
	}

	if req.VideoId <= 0 {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "video_id must be greater than zero",
		}, nil
	}

	if req.ActionType != 1 && req.ActionType != 2 {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "action_type must be 1 or 2",
		}, nil
	}

	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "token is invalid or expired",
		}, nil
	}

	if claims.Id <= 0 {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "invalid user id in token",
		}, nil
	}

	event := apprabbitmq.NewFavoriteChangedEvent(
		uint(claims.Id),
		uint(req.VideoId),
		req.ActionType,
	)

	outboxEvent, err := db.NewOutboxEvent(
		event.EventID,
		"favorite",
		req.VideoId,
		apprabbitmq.FavoriteChangedRoutingKey,
		event,
	)
	if err != nil {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "create favorite event failed",
		}, nil
	}

	err = db.FavoriteActionWithOutbox(
		ctx,
		uint(claims.Id),
		uint(req.VideoId),
		req.ActionType,
		outboxEvent,
	)
	if err != nil {
		return &favoritepb.FavoriteActionResponse{
			StatusCode: -1,
			StatusMsg:  "favorite action failed",
		}, nil
	}

	return &favoritepb.FavoriteActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
	}, nil
}

// FavoriteList 查询指定用户点赞过的视频列表。
func (s *FavoriteServiceImpl) FavoriteList(
	ctx context.Context,
	req *favoritepb.FavoriteListRequest,
) (*favoritepb.FavoriteListResponse, error) {
	if req == nil {
		return favoriteListError("request cannot be nil"), nil
	}

	if req.UserId <= 0 {
		return favoriteListError(
			"user_id must be greater than zero",
		), nil
	}

	if req.Token == "" {
		return favoriteListError("token cannot be empty"), nil
	}

	// token 代表当前正在查看列表的登录用户。
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		return favoriteListError(
			"token is invalid or expired",
		), nil
	}

	if claims.Id <= 0 {
		return favoriteListError(
			"invalid user id in token",
		), nil
	}

	// user_id 代表要查询谁的点赞列表。
	_, err = appcache.GetUserByID(ctx, req.UserId)
	if err != nil {
		return favoriteListError("user does not exist"), nil
	}

	videos, err := db.GetFavoriteVideosByUserID(
		ctx,
		uint(req.UserId),
	)
	if err != nil {
		return favoriteListError(
			"query favorite videos failed",
		), nil
	}

	videoList := make(
		[]*videopb.Video,
		0,
		len(videos),
	)

	for _, currentVideo := range videos {
		videoItem, err := buildFavoriteVideo(
			ctx,
			currentVideo,
			uint(claims.Id),
		)
		if err != nil {
			return favoriteListError(
				"build favorite video failed",
			), nil
		}

		videoList = append(videoList, videoItem)
	}

	return &favoritepb.FavoriteListResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		VideoList:  videoList,
	}, nil
}

// buildFavoriteVideo 把数据库 Video 转成 RPC 返回的 Video。
func buildFavoriteVideo(
	ctx context.Context,
	currentVideo *db.Video,
	currentUserID uint,
) (*videopb.Video, error) {
	author, err := appcache.GetUserByID(
		ctx,
		int64(currentVideo.AuthorID),
	)
	if err != nil {
		return nil, err
	}

	isFavorite, err := appcache.IsFavorite(
		ctx,
		currentUserID,
		currentVideo.ID,
	)
	if err != nil {
		return nil, err
	}

	playURL, err := getTemporaryURL(
		appminio.VideoBucketName,
		currentVideo.PlayUrl,
	)
	if err != nil {
		return nil, err
	}

	coverURL, err := getTemporaryURL(
		appminio.CoverBucketName,
		currentVideo.CoverUrl,
	)
	if err != nil {
		return nil, err
	}

	avatarURL, err := getTemporaryURL(
		appminio.AvatarBucketName,
		author.Avatar,
	)
	if err != nil {
		return nil, err
	}

	backgroundURL, err := getTemporaryURL(
		appminio.BackgroundImageBucketName,
		author.BackgroundImage,
	)
	if err != nil {
		return nil, err
	}

	return &videopb.Video{
		Id: int64(currentVideo.ID),

		Author: &userpb.User{
			Id:              int64(author.ID),
			Name:            author.UserName,
			FollowCount:     int64(author.FollowingCount),
			FollowerCount:   int64(author.FollowerCount),
			IsFollow:        false,
			Avatar:          avatarURL,
			BackgroundImage: backgroundURL,
			Signature:       author.Signature,
			TotalFavorited:  int64(author.TotalFavorited),
			WorkCount:       int64(author.WorkCount),
			FavoriteCount:   int64(author.FavoriteCount),
		},

		PlayUrl:       playURL,
		CoverUrl:      coverURL,
		FavoriteCount: int64(currentVideo.FavoriteCount),
		CommentCount:  int64(currentVideo.CommentCount),
		IsFavorite:    isFavorite,
		Title:         currentVideo.Title,
	}, nil
}

// getTemporaryURL 对空对象名直接返回空字符串，避免请求 MinIO。
func getTemporaryURL(
	bucketName string,
	objectName string,
) (string, error) {
	if objectName == "" {
		return "", nil
	}

	return appminio.GetFileTemporaryURL(
		bucketName,
		objectName,
	)
}

func favoriteListError(
	message string,
) *favoritepb.FavoriteListResponse {
	return &favoritepb.FavoriteListResponse{
		StatusCode: -1,
		StatusMsg:  message,
		VideoList:  nil,
	}
}
