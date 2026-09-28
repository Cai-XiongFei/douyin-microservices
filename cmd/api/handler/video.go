package handler

import (
	"context"
	"douyin/cmd/api/rpc"
	"douyin/internal/response"
	"douyin/kitex/kitex_gen/video"
	videopb "douyin/kitex/kitex_gen/video"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func PublishList(ctx context.Context, c *app.RequestContext) {
	userIDText := c.Query("user_id")
	token := c.Query("token")

	userID, err := strconv.ParseInt(userIDText, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, response.PublishList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "user_id is invalid",
			},
			VideoList: make([]*videopb.Video, 0),
		})
		return
	}

	rpcResponse, err := rpc.PublishList(
		ctx,
		&videopb.PublishListRequest{
			UserId: userID,
			Token:  token,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.PublishList{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video RPC request failed",
			},
			VideoList: make([]*videopb.Video, 0),
		})
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(http.StatusOK, response.PublishList{
			Base: response.Base{
				StatusCode: int(rpcResponse.StatusCode),
				StatusMsg:  rpcResponse.StatusMsg,
			},
			VideoList: make([]*videopb.Video, 0),
		})
		return
	}

	c.JSON(http.StatusOK, response.PublishList{
		Base: response.Base{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		VideoList: rpcResponse.VideoList,
	})
}

const maxUploadVideoSize int64 = 50 * 1024 * 1024

func PublishAction(ctx context.Context, c *app.RequestContext) {
	token := c.GetString("authenticated_token")
	title := strings.TrimSpace(c.PostForm("title"))

	if title == "" {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video title cannot be empty",
			},
		})
		return
	}

	fileHeader, err := c.FormFile("data")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video file is required",
			},
		})
		return
	}

	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video file cannot be empty",
			},
		})
		return
	}

	if fileHeader.Size > maxUploadVideoSize {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video size cannot exceed 50 MB",
			},
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "open video file failed",
			},
		})
		return
	}
	defer file.Close()

	// 最多读取 50 MB + 1 字节，防止客户端伪造文件大小。
	videoData, err := io.ReadAll(
		io.LimitReader(
			file,
			maxUploadVideoSize+1,
		),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "read video file failed",
			},
		})
		return
	}

	if int64(len(videoData)) > maxUploadVideoSize {
		c.JSON(http.StatusBadRequest, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video size cannot exceed 50 MB",
			},
		})
		return
	}

	rpcResponse, err := rpc.PublishAction(
		ctx,
		&videopb.PublishActionRequest{
			Token: token,
			Data:  videoData,
			Title: title,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.PublishAction{
			Base: response.Base{
				StatusCode: -1,
				StatusMsg:  "video RPC request failed",
			},
		})
		return
	}

	if rpcResponse.StatusCode != 0 {
		c.JSON(http.StatusOK, response.PublishAction{
			Base: response.Base{
				StatusCode: int(rpcResponse.StatusCode),
				StatusMsg:  rpcResponse.StatusMsg,
			},
		})
		return
	}

	c.JSON(http.StatusOK, response.PublishAction{
		Base: response.Base{
			StatusCode: 0,
			StatusMsg:  rpcResponse.StatusMsg,
		},
	})
}

func Feed(ctx context.Context, c *app.RequestContext) {
	var latestTime int64

	latestTimeText := c.Query("latest_time")
	if latestTimeText != "" {
		value, err := strconv.ParseInt(latestTimeText, 10, 64)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				response.Feed{
					Base: response.Base{
						StatusCode: -1,
						StatusMsg:  "latest_time must be an integer",
					},
					VideoList: nil,
					NextTime:  0,
				},
			)
			return
		}

		latestTime = value
	}

	token := c.Query("token")

	result, err := rpc.Feed(ctx, &video.FeedRequest{
		LatestTime: latestTime,
		Token:      token,
	})
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			response.Feed{
				Base: response.Base{
					StatusCode: -1,
					StatusMsg:  "video RPC request failed",
				},
				VideoList: nil,
				NextTime:  0,
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		response.Feed{
			Base: response.Base{
				StatusCode: int(result.StatusCode),
				StatusMsg:  result.StatusMsg,
			},
			VideoList: result.VideoList,
			NextTime:  result.NextTime,
		},
	)
}
