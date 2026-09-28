package handler

import (
	"context"
	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	"douyin/kitex/kitex_gen/user"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

func Register(ctx context.Context, c *app.RequestContext) {
	username := c.Query("username")
	password := c.Query("password")

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, response.Register{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "username and password cannot be empty",
			},
		})
		return
	}

	if len(username) > 32 || len(password) > 32 {
		c.JSON(http.StatusBadRequest, response.Register{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "username and password cannot exceed 32 characters",
			},
		})
		return
	}

	rpcResponse, err := rpc.Register(
		ctx,
		&user.UserRegisterRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Register{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user RPC request failed",
			},
		})
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(http.StatusOK, response.Register{
			Base: response.Base{
				StatusCode: int(rpcResponse.StatusCode),
				StatusMsg:  rpcResponse.StatusMsg,
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.Register{
		Base: response.Base{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		UserID: rpcResponse.UserId,
		Token:  rpcResponse.Token,
	})
}

func Login(ctx context.Context, c *app.RequestContext) {
	username := c.Query("username")
	password := c.Query("password")

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, response.Login{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "username and password cannot be empty",
			},
		})
		return
	}

	rpcResponse, err := rpc.Login(
		ctx,
		&user.UserLoginRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Login{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user RPC request failed",
			},
		})
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(http.StatusOK, response.Login{
			Base: response.Base{
				StatusCode: int(rpcResponse.StatusCode),
				StatusMsg:  rpcResponse.StatusMsg,
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.Login{
		Base: response.Base{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		UserID: rpcResponse.UserId,
		Token:  rpcResponse.Token,
	})
}

func UserInfo(ctx context.Context, c *app.RequestContext) {
	userIDText := c.Query("user_id")
	token := c.Query("token")

	userID, err := strconv.ParseInt(userIDText, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, response.UserInfo{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user_id is invalid",
			},
		})
		return
	}

	rpcResponse, err := rpc.UserInfo(
		ctx,
		&user.UserInfoRequest{
			UserId: userID,
			Token:  token,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.UserInfo{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user RPC request failed",
			},
		})
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(http.StatusOK, response.UserInfo{
			Base: response.Base{
				StatusCode: int(rpcResponse.StatusCode),
				StatusMsg:  rpcResponse.StatusMsg,
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.UserInfo{
		Base: response.Base{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		User: rpcResponse.User,
	})
}
