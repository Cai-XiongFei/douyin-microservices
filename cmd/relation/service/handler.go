package service

import (
	"context"
	"errors"
	"log"
	"time"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	relationpb "douyin/kitex/kitex_gen/relation"
	userpb "douyin/kitex/kitex_gen/user"
	appjwt "douyin/pkg/jwt"
	storage "douyin/pkg/minio"
	apprabbitmq "douyin/pkg/rabbitmq"
)

type RelationServiceImpl struct {
	jwtManager *appjwt.JWT
}

func NewRelationServiceImpl(signingKey []byte) *RelationServiceImpl {
	return &RelationServiceImpl{
		jwtManager: appjwt.NewJWT(signingKey),
	}
}

// RelationAction 处理关注和取消关注。
func (s *RelationServiceImpl) RelationAction(
	ctx context.Context,
	req *relationpb.RelationActionRequest,
) (*relationpb.RelationActionResponse, error) {
	if req == nil {
		return relationActionError("request cannot be nil"), nil
	}
	if req.GetToken() == "" {
		return relationActionError("token cannot be empty"), nil
	}
	if req.GetToUserId() <= 0 {
		return relationActionError("to_user_id must be greater than zero"), nil
	}
	if req.GetActionType() != 1 && req.GetActionType() != 2 {
		return relationActionError("action_type must be 1 or 2"), nil
	}

	claims, err := s.parseToken(req.GetToken())
	if err != nil {
		return relationActionError("token is invalid or expired"), nil
	}
	if claims.Id == req.GetToUserId() {
		return relationActionError("cannot follow yourself"), nil
	}

	err = db.RelationAction(
		ctx,
		uint(claims.Id),
		uint(req.GetToUserId()),
		req.GetActionType(),
	)
	if err != nil {
		return relationActionError("relation action failed"), nil
	}

	event := apprabbitmq.NewRelationChangedEvent(
		uint(claims.Id),
		uint(req.GetToUserId()),
		req.GetActionType(),
	)
	publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	publishErr := apprabbitmq.PublishRelationChanged(publishCtx, event)
	cancel()
	if publishErr != nil {
		log.Printf("publish relation event failed, fallback to direct invalidation: %v", publishErr)
		appcache.InvalidateRelationStatus(ctx, uint(claims.Id), uint(req.GetToUserId()))
		appcache.InvalidateUser(ctx, uint(claims.Id))
		appcache.InvalidateUser(ctx, uint(req.GetToUserId()))
	}

	return &relationpb.RelationActionResponse{
		StatusCode: 0,
		StatusMsg:  "success",
	}, nil
}

// RelationFollowList 查询指定用户关注的人。
func (s *RelationServiceImpl) RelationFollowList(
	ctx context.Context,
	req *relationpb.RelationFollowListRequest,
) (*relationpb.RelationFollowListResponse, error) {
	if req == nil {
		return relationFollowListError("request cannot be nil"), nil
	}
	if req.GetUserId() <= 0 {
		return relationFollowListError("user_id must be greater than zero"), nil
	}

	claims, err := s.parseToken(req.GetToken())
	if err != nil {
		return relationFollowListError("token is invalid or expired"), nil
	}

	users, err := db.GetFollowingUsersByUserID(
		ctx,
		uint(req.GetUserId()),
	)
	if err != nil {
		return relationFollowListError("query following list failed"), nil
	}

	userList, err := buildUserList(ctx, uint(claims.Id), users)
	if err != nil {
		return relationFollowListError("build following list failed"), nil
	}

	return &relationpb.RelationFollowListResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		UserList:   userList,
	}, nil
}

// RelationFollowerList 查询指定用户的粉丝。
func (s *RelationServiceImpl) RelationFollowerList(
	ctx context.Context,
	req *relationpb.RelationFollowerListRequest,
) (*relationpb.RelationFollowerListResponse, error) {
	if req == nil {
		return relationFollowerListError("request cannot be nil"), nil
	}
	if req.GetUserId() <= 0 {
		return relationFollowerListError("user_id must be greater than zero"), nil
	}

	claims, err := s.parseToken(req.GetToken())
	if err != nil {
		return relationFollowerListError("token is invalid or expired"), nil
	}

	users, err := db.GetFollowerUsersByUserID(
		ctx,
		uint(req.GetUserId()),
	)
	if err != nil {
		return relationFollowerListError("query follower list failed"), nil
	}

	userList, err := buildUserList(ctx, uint(claims.Id), users)
	if err != nil {
		return relationFollowerListError("build follower list failed"), nil
	}

	return &relationpb.RelationFollowerListResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		UserList:   userList,
	}, nil
}

// RelationFriendList 查询互相关注的好友。
func (s *RelationServiceImpl) RelationFriendList(
	ctx context.Context,
	req *relationpb.RelationFriendListRequest,
) (*relationpb.RelationFriendListResponse, error) {
	if req == nil {
		return relationFriendListError("request cannot be nil"), nil
	}
	if req.GetUserId() <= 0 {
		return relationFriendListError("user_id must be greater than zero"), nil
	}

	claims, err := s.parseToken(req.GetToken())
	if err != nil {
		return relationFriendListError("token is invalid or expired"), nil
	}
	if claims.Id != req.GetUserId() {
		return relationFriendListError(
			"cannot query another user's friend list",
		), nil
	}

	users, err := db.GetFriendUsersByUserID(
		ctx,
		uint(req.GetUserId()),
	)
	if err != nil {
		return relationFriendListError("query friend list failed"), nil
	}

	friendList := make([]*relationpb.FriendUser, 0, len(users))
	for _, user := range users {
		result, err := buildFriendUser(
			ctx,
			uint(claims.Id),
			user,
		)
		if err != nil {
			return relationFriendListError("build friend list failed"), nil
		}
		friendList = append(friendList, result)
	}

	return &relationpb.RelationFriendListResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		UserList:   friendList,
	}, nil
}

func (s *RelationServiceImpl) parseToken(
	token string,
) (*appjwt.CustomClaims, error) {
	if s.jwtManager == nil || token == "" {
		return nil, errors.New("invalid token")
	}

	claims, err := s.jwtManager.ParseToken(token)
	if err != nil || claims.Id <= 0 {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func buildUserList(
	ctx context.Context,
	currentUserID uint,
	users []*db.User,
) ([]*userpb.User, error) {
	result := make([]*userpb.User, 0, len(users))

	for _, user := range users {
		isFollow, err := appcache.IsFollowing(ctx, currentUserID, user.ID)
		if err != nil {
			return nil, err
		}

		item, err := buildUser(user, isFollow)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, nil
}

func buildUser(user *db.User, isFollow bool) (*userpb.User, error) {
	avatarURL, backgroundURL, err := buildUserImageURLs(user)
	if err != nil {
		return nil, err
	}

	return &userpb.User{
		Id:              int64(user.ID),
		Name:            user.UserName,
		FollowCount:     int64(user.FollowingCount),
		FollowerCount:   int64(user.FollowerCount),
		IsFollow:        isFollow,
		Avatar:          avatarURL,
		BackgroundImage: backgroundURL,
		Signature:       user.Signature,
		TotalFavorited:  int64(user.TotalFavorited),
		WorkCount:       int64(user.WorkCount),
		FavoriteCount:   int64(user.FavoriteCount),
	}, nil
}

func buildFriendUser(
	ctx context.Context,
	currentUserID uint,
	user *db.User,
) (*relationpb.FriendUser, error) {
	avatarURL, backgroundURL, err := buildUserImageURLs(user)
	if err != nil {
		return nil, err
	}

	latestMessage, err := appcache.GetLatestMessageBetweenUsers(
		ctx,
		currentUserID,
		user.ID,
	)
	if err != nil {
		return nil, err
	}

	message := ""
	var msgType int64

	if latestMessage != nil {
		message = latestMessage.Content
		if latestMessage.FromUserID == currentUserID {
			msgType = 1
		}
	}

	return &relationpb.FriendUser{
		Message:         message,
		MsgType:         msgType,
		Id:              int64(user.ID),
		Name:            user.UserName,
		FollowCount:     int64(user.FollowingCount),
		FollowerCount:   int64(user.FollowerCount),
		IsFollow:        true,
		Avatar:          avatarURL,
		BackgroundImage: backgroundURL,
		Signature:       user.Signature,
		TotalFavorited:  int64(user.TotalFavorited),
		WorkCount:       int64(user.WorkCount),
		FavoriteCount:   int64(user.FavoriteCount),
	}, nil
}

func buildUserImageURLs(user *db.User) (string, string, error) {
	avatarURL := ""
	backgroundURL := ""

	if user.Avatar != "" {
		var err error
		avatarURL, err = storage.GetFileTemporaryURL(
			storage.AvatarBucketName,
			user.Avatar,
		)
		if err != nil {
			return "", "", err
		}
	}

	if user.BackgroundImage != "" {
		var err error
		backgroundURL, err = storage.GetFileTemporaryURL(
			storage.BackgroundImageBucketName,
			user.BackgroundImage,
		)
		if err != nil {
			return "", "", err
		}
	}

	return avatarURL, backgroundURL, nil
}

func relationActionError(message string) *relationpb.RelationActionResponse {
	return &relationpb.RelationActionResponse{
		StatusCode: -1,
		StatusMsg:  message,
	}
}

func relationFollowListError(
	message string,
) *relationpb.RelationFollowListResponse {
	return &relationpb.RelationFollowListResponse{
		StatusCode: -1,
		StatusMsg:  message,
		UserList:   make([]*userpb.User, 0),
	}
}

func relationFollowerListError(
	message string,
) *relationpb.RelationFollowerListResponse {
	return &relationpb.RelationFollowerListResponse{
		StatusCode: -1,
		StatusMsg:  message,
		UserList:   make([]*userpb.User, 0),
	}
}

func relationFriendListError(
	message string,
) *relationpb.RelationFriendListResponse {
	return &relationpb.RelationFriendListResponse{
		StatusCode: -1,
		StatusMsg:  message,
		UserList:   make([]*relationpb.FriendUser, 0),
	}
}
