package handler

import (
	"context"
	"net/http"
	"strconv"

	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	relationpb "douyin/kitex/kitex_gen/relation"
	userpb "douyin/kitex/kitex_gen/user"

	"github.com/cloudwego/hertz/pkg/app"
)

func RelationAction(ctx context.Context, c *app.RequestContext) {
	token := c.GetString("authenticated_token")

	toUserID, err := strconv.ParseInt(c.Query("to_user_id"), 10, 64)
	if err != nil || toUserID <= 0 {
		c.JSON(http.StatusBadRequest, response.RelationAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "to_user_id is invalid",
			},
		})
		return
	}

	actionType, err := strconv.ParseInt(c.Query("action_type"), 10, 32)
	if err != nil || (actionType != 1 && actionType != 2) {
		c.JSON(http.StatusBadRequest, response.RelationAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "action_type must be 1 or 2",
			},
		})
		return
	}

	rpcResponse, err := rpc.RelationAction(
		ctx,
		&relationpb.RelationActionRequest{
			Token:      token,
			ToUserId:   toUserID,
			ActionType: int32(actionType),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.RelationAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "relation RPC request failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.RelationAction{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
	})
}

func RelationFollowList(ctx context.Context, c *app.RequestContext) {
	userID, ok := parseRelationUserID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, response.RelationFollowList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user_id is invalid",
			},
			UserList: make([]*userpb.User, 0),
		})
		return
	}

	rpcResponse, err := rpc.RelationFollowList(
		ctx,
		&relationpb.RelationFollowListRequest{
			UserId: userID,
			Token:  c.GetString("authenticated_token"),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.RelationFollowList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "relation RPC request failed",
			},
			UserList: make([]*userpb.User, 0),
		})
		return
	}

	c.JSON(http.StatusOK, response.RelationFollowList{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
		UserList: rpcResponse.UserList,
	})
}

func RelationFollowerList(ctx context.Context, c *app.RequestContext) {
	userID, ok := parseRelationUserID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, response.RelationFollowerList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user_id is invalid",
			},
			UserList: make([]*userpb.User, 0),
		})
		return
	}

	rpcResponse, err := rpc.RelationFollowerList(
		ctx,
		&relationpb.RelationFollowerListRequest{
			UserId: userID,
			Token:  c.GetString("authenticated_token"),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.RelationFollowerList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "relation RPC request failed",
			},
			UserList: make([]*userpb.User, 0),
		})
		return
	}

	c.JSON(http.StatusOK, response.RelationFollowerList{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
		UserList: rpcResponse.UserList,
	})
}

func RelationFriendList(ctx context.Context, c *app.RequestContext) {
	userID, ok := parseRelationUserID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, response.RelationFriendList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user_id is invalid",
			},
			UserList: make([]*relationpb.FriendUser, 0),
		})
		return
	}

	rpcResponse, err := rpc.RelationFriendList(
		ctx,
		&relationpb.RelationFriendListRequest{
			UserId: userID,
			Token:  c.GetString("authenticated_token"),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.RelationFriendList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "relation RPC request failed",
			},
			UserList: make([]*relationpb.FriendUser, 0),
		})
		return
	}

	c.JSON(http.StatusOK, response.RelationFriendList{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
		UserList: rpcResponse.UserList,
	})
}

func parseRelationUserID(c *app.RequestContext) (int64, bool) {
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}

	return userID, true
}
