package service

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	commentpb "douyin/kitex/kitex_gen/comment"
	userpb "douyin/kitex/kitex_gen/user"
	appjwt "douyin/pkg/jwt"
	storage "douyin/pkg/minio"
	apprabbitmq "douyin/pkg/rabbitmq"
)

const maxCommentLength = 255

type CommentServiceImpl struct {
	jwtManager *appjwt.JWT
}

func NewCommentServiceImpl(
	signingKey []byte,
) *CommentServiceImpl {
	return &CommentServiceImpl{
		jwtManager: appjwt.NewJWT(signingKey),
	}
}

// CommentAction 处理发布评论和删除评论。
func (s *CommentServiceImpl) CommentAction(
	ctx context.Context,
	req *commentpb.CommentActionRequest,
) (*commentpb.CommentActionResponse, error) {
	if req == nil {
		return commentActionError(
			"request cannot be nil",
		), nil
	}

	if req.GetToken() == "" {
		return commentActionError(
			"token cannot be empty",
		), nil
	}

	if req.GetVideoId() <= 0 {
		return commentActionError(
			"video_id must be greater than zero",
		), nil
	}

	if req.GetActionType() != 1 &&
		req.GetActionType() != 2 {
		return commentActionError(
			"action_type must be 1 or 2",
		), nil
	}

	if s.jwtManager == nil {
		return commentActionError(
			"JWT manager is not initialized",
		), nil
	}

	claims, err := s.jwtManager.ParseToken(
		req.GetToken(),
	)
	if err != nil || claims.Id <= 0 {
		return commentActionError(
			"token is invalid or expired",
		), nil
	}

	switch req.GetActionType() {
	case 1:
		return s.createComment(
			ctx,
			uint(claims.Id),
			uint(req.GetVideoId()),
			req.GetCommentText(),
		)

	case 2:
		return s.deleteComment(
			ctx,
			uint(claims.Id),
			uint(req.GetVideoId()),
			uint(req.GetCommentId()),
		)
	}

	return commentActionError(
		"invalid comment action",
	), nil
}

func (s *CommentServiceImpl) createComment(
	ctx context.Context,
	userID uint,
	videoID uint,
	commentText string,
) (*commentpb.CommentActionResponse, error) {
	commentText = strings.TrimSpace(commentText)

	if commentText == "" {
		return commentActionError(
			"comment text cannot be empty",
		), nil
	}

	if utf8.RuneCountInString(commentText) >
		maxCommentLength {
		return commentActionError(
			"comment text cannot exceed 255 characters",
		), nil
	}

	commentRecord, err := db.CreateComment(
		ctx,
		userID,
		videoID,
		commentText,
	)
	if err != nil {
		return commentActionError(
			"create comment failed",
		), nil
	}

	event := apprabbitmq.NewCommentChangedEvent(userID, videoID, commentRecord.ID, 1)
	publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	publishErr := apprabbitmq.PublishCommentChanged(publishCtx, event)
	cancel()
	if publishErr != nil {
		log.Printf("publish comment event failed, fallback to direct invalidation: %v", publishErr)
		appcache.InvalidateCommentList(ctx, videoID)
	}

	result, err := buildCommentResponse(
		ctx,
		commentRecord,
		userID,
	)
	if err != nil {
		return commentActionError(
			"build comment response failed",
		), nil
	}

	return &commentpb.CommentActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		Comment:    result,
	}, nil
}

func (s *CommentServiceImpl) deleteComment(
	ctx context.Context,
	userID uint,
	videoID uint,
	commentID uint,
) (*commentpb.CommentActionResponse, error) {
	if commentID == 0 {
		return commentActionError(
			"comment_id must be greater than zero",
		), nil
	}

	err := db.DeleteComment(
		ctx,
		userID,
		videoID,
		commentID,
	)
	if err != nil {
		return commentActionError(
			"delete comment failed",
		), nil
	}

	event := apprabbitmq.NewCommentChangedEvent(userID, videoID, commentID, 2)
	publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	publishErr := apprabbitmq.PublishCommentChanged(publishCtx, event)
	cancel()
	if publishErr != nil {
		log.Printf("publish comment event failed, fallback to direct invalidation: %v", publishErr)
		appcache.InvalidateCommentList(ctx, videoID)
	}

	return &commentpb.CommentActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		Comment:    nil,
	}, nil
}

// CommentList 查询指定视频的评论列表。
// 评论列表允许未登录用户访问，token 是可选的。
func (s *CommentServiceImpl) CommentList(
	ctx context.Context,
	req *commentpb.CommentListRequest,
) (*commentpb.CommentListResponse, error) {
	if req == nil {
		return commentListError(
			"request cannot be nil",
		), nil
	}

	if req.GetVideoId() <= 0 {
		return commentListError(
			"video_id must be greater than zero",
		), nil
	}

	// 未登录时 token 可以为空；
	// 如果传了 token，就检查它是否有效。
	var currentUserID uint

	if req.GetToken() != "" {
		if s.jwtManager == nil {
			return commentListError(
				"JWT manager is not initialized",
			), nil
		}

		claims, err := s.jwtManager.ParseToken(
			req.GetToken(),
		)
		if err != nil || claims.Id <= 0 {
			return commentListError(
				"token is invalid or expired",
			), nil
		}

		currentUserID = uint(claims.Id)
	}

	commentRecords, err := appcache.GetCommentsByVideoID(
		ctx,
		uint(req.GetVideoId()),
	)
	if err != nil {
		return commentListError(
			"query comments failed",
		), nil
	}

	commentList := make(
		[]*commentpb.Comment,
		0,
		len(commentRecords),
	)

	for _, record := range commentRecords {
		result, err := buildCommentResponse(
			ctx,
			record,
			currentUserID,
		)
		if err != nil {
			return commentListError(
				"build comment response failed",
			), nil
		}

		commentList = append(
			commentList,
			result,
		)
	}

	return &commentpb.CommentListResponse{
		StatusCode:  0,
		StatusMsg:   "success",
		CommentList: commentList,
	}, nil
}

// buildCommentResponse 把数据库评论转换成 RPC 评论结构。
func buildCommentResponse(
	ctx context.Context,
	record *db.Comment,
	currentUserID uint,
) (*commentpb.Comment, error) {
	currentUser, err := appcache.GetUserByID(
		ctx,
		int64(record.UserID),
	)
	if err != nil {
		return nil, err
	}

	avatarURL := ""

	if currentUser.Avatar != "" {
		avatarURL, err = storage.GetFileTemporaryURL(
			storage.AvatarBucketName,
			currentUser.Avatar,
		)
		if err != nil {
			return nil, err
		}
	}

	backgroundURL := ""

	if currentUser.BackgroundImage != "" {
		backgroundURL, err =
			storage.GetFileTemporaryURL(
				storage.BackgroundImageBucketName,
				currentUser.BackgroundImage,
			)
		if err != nil {
			return nil, err
		}
	}

	isFollow := false
	if currentUserID > 0 {
		isFollow, err = appcache.IsFollowing(
			ctx,
			currentUserID,
			currentUser.ID,
		)
		if err != nil {
			return nil, err
		}
	}

	return &commentpb.Comment{
		Id: int64(record.ID),
		User: &userpb.User{
			Id:              int64(currentUser.ID),
			Name:            currentUser.UserName,
			FollowCount:     int64(currentUser.FollowingCount),
			FollowerCount:   int64(currentUser.FollowerCount),
			IsFollow:        isFollow,
			Avatar:          avatarURL,
			BackgroundImage: backgroundURL,
			Signature:       currentUser.Signature,
			TotalFavorited:  int64(currentUser.TotalFavorited),
			WorkCount:       int64(currentUser.WorkCount),
			FavoriteCount:   int64(currentUser.FavoriteCount),
		},
		Content: record.Content,
		CreateDate: record.CreatedAt.Format(
			"01-02",
		),
	}, nil
}

func commentActionError(
	message string,
) *commentpb.CommentActionResponse {
	return &commentpb.CommentActionResponse{
		StatusCode: -1,
		StatusMsg:  message,
		Comment:    nil,
	}
}

func commentListError(
	message string,
) *commentpb.CommentListResponse {
	return &commentpb.CommentListResponse{
		StatusCode:  -1,
		StatusMsg:   message,
		CommentList: make([]*commentpb.Comment, 0),
	}
}
