package middleware

import (
	"context"
	"net/http"

	"douyin/internal/response"
	appjwt "douyin/pkg/jwt"

	"github.com/cloudwego/hertz/pkg/app"
)

func TokenAuth(jwtManager *appjwt.JWT) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		token := c.Query("token")

		if token == "" {
			token = c.PostForm("token")
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				response.Base{
					StatusCode: -1,
					StatusMsg:  "token is invalid or expired",
				},
			)
			return
		}

		// 把解析出来的登录用户 ID 传给后面的 Handler。
		c.Set("authenticated_user_id", claims.Id)
		c.Set("authenticated_token", token)
		c.Next(ctx)
	}
}
