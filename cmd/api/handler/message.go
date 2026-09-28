package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	messagepb "douyin/kitex/kitex_gen/message"

	"github.com/cloudwego/hertz/pkg/app"
)

// MessageAction 发送一条私信。
func MessageAction(ctx context.Context, c *app.RequestContext) {
	toUserID, err := strconv.ParseInt(c.Query("to_user_id"), 10, 64)
	if err != nil || toUserID <= 0 {
		c.JSON(http.StatusBadRequest, response.MessageAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "to_user_id is invalid",
			},
		})
		return
	}

	actionType, err := strconv.ParseInt(c.Query("action_type"), 10, 32)
	if err != nil || actionType != 1 {
		c.JSON(http.StatusBadRequest, response.MessageAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "action_type must be 1",
			},
		})
		return
	}

	content := strings.TrimSpace(c.Query("content"))
	if content == "" {
		content = strings.TrimSpace(c.PostForm("content"))
	}
	if content == "" {
		c.JSON(http.StatusBadRequest, response.MessageAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "message content cannot be empty",
			},
		})
		return
	}

	rpcResponse, err := rpc.MessageAction(
		ctx,
		&messagepb.MessageActionRequest{
			Token:      c.GetString("authenticated_token"),
			ToUserId:   toUserID,
			ActionType: int32(actionType),
			Content:    content,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.MessageAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "message RPC request failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.MessageAction{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
	})
}

// MessageChat 查询当前用户和指定好友之间的聊天记录。
func MessageChat(ctx context.Context, c *app.RequestContext) {
	toUserID, err := strconv.ParseInt(c.Query("to_user_id"), 10, 64)
	if err != nil || toUserID <= 0 {
		c.JSON(http.StatusBadRequest, response.MessageChat{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "to_user_id is invalid",
			},
			MessageList: make([]*messagepb.Message, 0),
		})
		return
	}

	var preMsgTime int64
	preMsgTimeText := c.Query("pre_msg_time")
	if preMsgTimeText != "" {
		preMsgTime, err = strconv.ParseInt(preMsgTimeText, 10, 64)
		if err != nil || preMsgTime < 0 {
			c.JSON(http.StatusBadRequest, response.MessageChat{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "pre_msg_time is invalid",
				},
				MessageList: make([]*messagepb.Message, 0),
			})
			return
		}
	}

	rpcResponse, err := rpc.MessageChat(
		ctx,
		&messagepb.MessageChatRequest{
			Token:      c.GetString("authenticated_token"),
			ToUserId:   toUserID,
			PreMsgTime: preMsgTime,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.MessageChat{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "message RPC request failed",
			},
			MessageList: make([]*messagepb.Message, 0),
		})
		return
	}

	c.JSON(http.StatusOK, response.MessageChat{
		Base: response.Base{
			StatusCode: int(rpcResponse.StatusCode),
			StatusMsg:  rpcResponse.StatusMsg,
		},
		MessageList: rpcResponse.MessageList,
	})
}
