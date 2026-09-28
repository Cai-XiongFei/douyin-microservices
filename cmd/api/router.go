package main

import (
	"douyin/cmd/api/handler"
	redisdb "douyin/dal/redis"
	"douyin/internal/limiter"
	"douyin/pkg/jwt"
	"douyin/pkg/middleware"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func registerRoutes(h *server.Hertz,
	jwtManager *jwt.JWT,
) {
	rateLimiter := limiter.NewTokenBucket(
		redisdb.Client,
		"rate_limit:",
	)

	favoriteRateLimit := middleware.UserRateLimit(
		rateLimiter,
		"favorite_action",
		0.1,
		3,
	)
	douyin := h.Group("/douyin")
	user := douyin.Group("/user")
	publish := douyin.Group("/publish")
	favorite := douyin.Group("/favorite")
	comment := douyin.Group("/comment")
	relation := douyin.Group("/relation")
	message := douyin.Group("/message")

	user.POST("/register/", handler.Register)
	user.POST("/login/", handler.Login)
	user.GET("/",
		middleware.TokenAuth(jwtManager),
		handler.UserInfo)
	publish.GET("/list/",
		middleware.TokenAuth(jwtManager),
		handler.PublishList)
	publish.POST(
		"/action/",
		middleware.TokenAuth(jwtManager),

		handler.PublishAction,
	)
	douyin.GET("/feed/", handler.Feed)
	favorite.POST(
		"/action/",
		middleware.TokenAuth(jwtManager),
		favoriteRateLimit,
		handler.FavoriteAction,
	)

	favorite.GET(
		"/list/",
		middleware.TokenAuth(jwtManager),
		handler.FavoriteList,
	)
	comment.POST(
		"/action/",
		middleware.TokenAuth(jwtManager),
		handler.CommentAction,
	)
	comment.GET(
		"/list/",
		handler.CommentList,
	)
	relation.POST(
		"/action/",
		middleware.TokenAuth(jwtManager),
		handler.RelationAction,
	)
	relation.GET(
		"/follow/list/",
		middleware.TokenAuth(jwtManager),
		handler.RelationFollowList,
	)
	relation.GET(
		"/follower/list/",
		middleware.TokenAuth(jwtManager),
		handler.RelationFollowerList,
	)
	relation.GET(
		"/friend/list/",
		middleware.TokenAuth(jwtManager),
		handler.RelationFriendList,
	)
	message.POST(
		"/action/",
		middleware.TokenAuth(jwtManager),
		handler.MessageAction,
	)
	message.GET(
		"/chat/",
		middleware.TokenAuth(jwtManager),
		handler.MessageChat,
	)
}
