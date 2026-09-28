package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	commentpb "douyin/kitex/kitex_gen/comment"

	"github.com/cloudwego/hertz/pkg/app"
)

// CommentAction 处理发布和删除评论。
func CommentAction(
	ctx context.Context,
	c *app.RequestContext,
) {
	token := c.GetString("authenticated_token")

	videoID, err := strconv.ParseInt(
		c.Query("video_id"),
		10,
		64,
	)
	if err != nil || videoID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			response.CommentAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "video_id is invalid",
				},
			},
		)
		return
	}

	actionType, err := strconv.ParseInt(
		c.Query("action_type"),
		10,
		32,
	)
	if err != nil ||
		(actionType != 1 && actionType != 2) {
		c.JSON(
			http.StatusBadRequest,
			response.CommentAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "action_type must be 1 or 2",
				},
			},
		)
		return
	}

	request := &commentpb.CommentActionRequest{
		Token:      token,
		VideoId:    videoID,
		ActionType: int32(actionType),
	}

	switch actionType {
	case 1:
		commentText := strings.TrimSpace(
			c.Query("comment_text"),
		)

		if commentText == "" {
			commentText = strings.TrimSpace(
				c.PostForm("comment_text"),
			)
		}

		if commentText == "" {
			c.JSON(
				http.StatusBadRequest,
				response.CommentAction{
					Base: response.Base{
						StatusCode: -1,
						StatusMsg:  "comment_text cannot be empty",
					},
				},
			)
			return
		}

		request.CommentText = commentText

	case 2:
		commentID, err := strconv.ParseInt(
			c.Query("comment_id"),
			10,
			64,
		)
		if err != nil || commentID <= 0 {
			c.JSON(
				http.StatusBadRequest,
				response.CommentAction{
					Base: response.Base{
						StatusCode: -1,
						StatusMsg:  "comment_id is invalid",
					},
				},
			)
			return
		}

		request.CommentId = commentID
	}

	rpcResponse, err := rpc.CommentAction(
		ctx,
		request,
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			response.CommentAction{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "comment RPC request failed",
				},
			},
		)
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(
			http.StatusOK,
			response.CommentAction{
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
		response.CommentAction{
			Base: response.Base{
				StatusCode: 0,
				StatusMsg:  rpcResponse.StatusMsg,
			},
			Comment: rpcResponse.Comment,
		},
	)
}

// CommentList 查询视频评论列表。
// 这是公开接口，token 可以不传。
func CommentList(
	ctx context.Context,
	c *app.RequestContext,
) {
	videoID, err := strconv.ParseInt(
		c.Query("video_id"),
		10,
		64,
	)
	if err != nil || videoID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			response.CommentList{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "video_id is invalid",
				},
				CommentList: make(
					[]*commentpb.Comment,
					0,
				),
			},
		)
		return
	}

	token := c.Query("token")

	rpcResponse, err := rpc.CommentList(
		ctx,
		&commentpb.CommentListRequest{
			Token:   token,
			VideoId: videoID,
		},
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			response.CommentList{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "comment RPC request failed",
				},
				CommentList: make(
					[]*commentpb.Comment,
					0,
				),
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		response.CommentList{
			Base: response.Base{
				StatusCode: int(
					rpcResponse.StatusCode,
				),
				StatusMsg: rpcResponse.StatusMsg,
			},
			CommentList: rpcResponse.CommentList,
		},
	)
}
