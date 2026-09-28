package handler

import (
	"context"
	"net/http"
	"strconv"

	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	favoritepb "douyin/kitex/kitex_gen/favorite"
	videopb "douyin/kitex/kitex_gen/video"

	"github.com/cloudwego/hertz/pkg/app"
)

// FavoriteAction 处理点赞和取消点赞。
func FavoriteAction(
	ctx context.Context,
	c *app.RequestContext,
) {
	token := c.GetString("authenticated_token")

	videoIDText := c.Query("video_id")
	videoID, err := strconv.ParseInt(
		videoIDText,
		10,
		64,
	)
	if err != nil || videoID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			response.FavoriteAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "video_id is invalid",
				},
			},
		)
		return
	}

	actionTypeText := c.Query("action_type")
	actionType, err := strconv.ParseInt(
		actionTypeText,
		10,
		32,
	)
	if err != nil ||
		(actionType != 1 && actionType != 2) {
		c.JSON(
			http.StatusBadRequest,
			response.FavoriteAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "action_type must be 1 or 2",
				},
			},
		)
		return
	}

	rpcResponse, err := rpc.FavoriteAction(
		ctx,
		&favoritepb.FavoriteActionRequest{
			Token:      token,
			VideoId:    videoID,
			ActionType: int32(actionType),
		},
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			response.FavoriteAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "favorite RPC request failed",
				},
			},
		)
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(
			http.StatusOK,
			response.FavoriteAction{
				Base: response.Base{
					StatusCode: int(
						rpcResponse.StatusCode,
					),
					StatusMsg: rpcResponse.StatusMsg,
				},
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		response.FavoriteAction{
			Base: response.Base{
				StatusCode: 0,
				StatusMsg:  rpcResponse.StatusMsg,
			},
		},
	)
}

// FavoriteList 查询指定用户点赞过的视频。
func FavoriteList(
	ctx context.Context,
	c *app.RequestContext,
) {
	token := c.GetString("authenticated_token")

	userIDText := c.Query("user_id")
	userID, err := strconv.ParseInt(
		userIDText,
		10,
		64,
	)
	if err != nil || userID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			response.FavoriteList{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "user_id is invalid",
				},
				VideoList: make([]*videopb.Video, 0),
			},
		)
		return
	}

	rpcResponse, err := rpc.FavoriteList(
		ctx,
		&favoritepb.FavoriteListRequest{
			UserId: userID,
			Token:  token,
		},
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			response.FavoriteList{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "favorite RPC request failed",
				},
				VideoList: make([]*videopb.Video, 0),
			},
		)
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(
			http.StatusOK,
			response.FavoriteList{
				Base: response.Base{
					StatusCode: int(
						rpcResponse.StatusCode,
					),
					StatusMsg: rpcResponse.StatusMsg,
				},
				VideoList: make([]*videopb.Video, 0),
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		response.FavoriteList{
			Base: response.Base{
				StatusCode: 0,
				StatusMsg:  rpcResponse.StatusMsg,
			},
			VideoList: rpcResponse.VideoList,
		},
	)
}
