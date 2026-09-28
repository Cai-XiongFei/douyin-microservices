package httpapi

import (
	"douyin/internal/httpapi/handler"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func RegisterRouters(h *server.Hertz) {
	h.GET("/healthz", handler.Health)
}
