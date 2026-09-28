package handler

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

func Health(_ context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status_code": 0,
		"status_msg":  "success",
		"service":     "douyin-api服务",
	})
}
