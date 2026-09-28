package service

import (
	"context"
	storage "douyin/pkg/minio"
	"fmt"
	"math/rand"
	"time"

	"douyin/dal/db"
	appcache "douyin/internal/cache"
	"douyin/internal/tool"
	"douyin/kitex/kitex_gen/user"
	appjwt "douyin/pkg/jwt"

	jwtlib "github.com/golang-jwt/jwt"
)

type UserServiceImpl struct {
	tokenManager *appjwt.JWT
}

func NewUserServiceImpl(signingKey string) *UserServiceImpl {
	return &UserServiceImpl{
		tokenManager: appjwt.NewJWT([]byte(signingKey)),
	}
}

func (s *UserServiceImpl) Register(
	ctx context.Context,
	req *user.UserRegisterRequest,
) (*user.UserRegisterResponse, error) {
	// 检查用户名是否已经存在
	existingUser, err := db.GetUserByName(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		msg := "用户已存在"
		return &user.UserRegisterResponse{
			StatusCode: 1,
			StatusMsg:  msg,
		}, nil
	}

	// 创建用户，数据库只保存密码的 MD5 结果
	newUser := &db.User{
		UserName: req.GetUsername(),
		Password: tool.Md5Encrypt(req.GetPassword()),
		Avatar:   fmt.Sprintf("default%d.png", rand.Intn(10)),
	}

	if err := db.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	// 注册成功后生成一个有效期为 5 分钟的 JWT
	token, err := s.createToken(int64(newUser.ID), 5*time.Minute)
	if err != nil {
		return nil, err
	}

	return &user.UserRegisterResponse{
		StatusCode: 0,
		UserId:     int64(newUser.ID),
		Token:      token,
	}, nil
}

func (s *UserServiceImpl) Login(
	ctx context.Context,
	req *user.UserLoginRequest,
) (*user.UserLoginResponse, error) {
	// 根据用户名查询数据库
	dbUser, err := db.GetUserByName(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}

	// 用户不存在，或者密码的 MD5 不一致
	if dbUser == nil ||
		dbUser.Password != tool.Md5Encrypt(req.GetPassword()) {
		msg := "用户名或密码错误"

		return &user.UserLoginResponse{
			StatusCode: 1,
			StatusMsg:  msg,
		}, nil
	}

	// 登录成功后生成一个有效期为 24 小时的 JWT
	token, err := s.createToken(int64(dbUser.ID), 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &user.UserLoginResponse{
		StatusCode: 0,
		UserId:     int64(dbUser.ID),
		Token:      token,
	}, nil
}

func (s *UserServiceImpl) createToken(
	userID int64,
	duration time.Duration,
) (string, error) {
	claims := appjwt.CustomClaims{
		Id: userID,
		StandardClaims: jwtlib.StandardClaims{
			ExpiresAt: time.Now().Add(duration).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "douyin-user-service",
		},
	}

	return s.tokenManager.CreateToken(claims)
}

func (s *UserServiceImpl) UserInfo(
	ctx context.Context,
	req *user.UserInfoRequest,
) (*user.UserInfoResponse, error) {
	if req == nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "request cannot be nil",
		}, nil
	}

	if s.tokenManager == nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "JWT manager is not initialized",
		}, nil
	}

	claims, err := s.tokenManager.ParseToken(req.GetToken())
	if err != nil || claims.Id <= 0 {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "token is invalid or expired",
		}, nil
	}
	// 根据请求中的用户 ID 查询数据库
	dbUser, err := appcache.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	if dbUser == nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "user does not exist",
		}, nil
	}

	// 根据数据库中的头像文件名生成临时访问地址
	avatarURL, err := storage.GetFileTemporaryURL(
		storage.AvatarBucketName,
		dbUser.Avatar,
	)
	if err != nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "failed to get avatar URL",
		}, nil
	}

	// 根据背景图文件名生成临时访问地址
	backgroundURL, err := storage.GetFileTemporaryURL(
		storage.BackgroundImageBucketName,
		dbUser.BackgroundImage,
	)
	if err != nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "failed to get background image URL",
		}, nil
	}

	isFollow, err := appcache.IsFollowing(
		ctx,
		uint(claims.Id),
		dbUser.ID,
	)
	if err != nil {
		return &user.UserInfoResponse{
			StatusCode: -1,
			StatusMsg:  "failed to query follow relation",
		}, nil
	}

	return &user.UserInfoResponse{
		StatusCode: 0,
		StatusMsg:  "success",
		User: &user.User{
			Id:              int64(dbUser.ID),
			Name:            dbUser.UserName,
			FollowCount:     int64(dbUser.FollowingCount),
			FollowerCount:   int64(dbUser.FollowerCount),
			IsFollow:        isFollow,
			Avatar:          avatarURL,
			BackgroundImage: backgroundURL,
			Signature:       dbUser.Signature,
			TotalFavorited:  int64(dbUser.TotalFavorited),
			WorkCount:       int64(dbUser.WorkCount),
			FavoriteCount:   int64(dbUser.FavoriteCount),
		},
	}, nil
}
