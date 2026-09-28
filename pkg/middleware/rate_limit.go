package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"douyin/internal/limiter"
	"douyin/internal/response"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// UserRateLimit 创建一个按登录用户限流的 Hertz 中间件。
//
// name 用于区分不同业务，例如 favorite_action、comment_action。
// ratePerSecond 表示每秒补充多少个令牌。
// capacity 表示桶中最多保存多少个令牌。
func UserRateLimit(
	bucket *limiter.TokenBucket,
	name string,
	ratePerSecond float64,
	capacity int64,
) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Header("X-RateLimit-Debug", "entered")
		userID, ok := getAuthenticatedUserID(c)
		if !ok || userID <= 0 {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				response.Base{
					StatusCode: -1,
					StatusMsg:  "authentication information is missing",
				},
			)
			return
		}

		// 不同用户、不同业务使用不同的令牌桶。
		key := fmt.Sprintf("%s:user:%d", name, userID)

		result, err := bucket.Allow(
			ctx,
			key,
			ratePerSecond,
			capacity,
			1,
		)
		if err != nil {
			// 当前采用 fail-open 策略：
			// Redis发生故障时记录日志，但不让整个接口不可用。
			hlog.CtxErrorf(
				ctx,
				"rate limiter failed: key=%s error=%v",
				key,
				err,
			)

			c.Next(ctx)
			return
		}

		// 返回当前限流信息，方便调试和观察。
		c.Header(
			"X-RateLimit-Limit",
			strconv.FormatInt(capacity, 10),
		)
		c.Header(
			"X-RateLimit-Remaining",
			strconv.FormatInt(result.Remaining, 10),
		)

		if !result.Allowed {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				response.Base{
					StatusCode: -1,
					StatusMsg:  "too many requests, please try again later",
				},
			)
			return
		}

		c.Next(ctx)
	}
}

// getAuthenticatedUserID 读取 JWT 中间件保存的用户ID。
func getAuthenticatedUserID(c *app.RequestContext) (int64, bool) {
	value, exists := c.Get("authenticated_user_id")
	if !exists {
		return 0, false
	}

	switch userID := value.(type) {
	case int64:
		return userID, true

	case int:
		return int64(userID), true

	case int32:
		return int64(userID), true

	case uint:
		if uint64(userID) > uint64(1<<63-1) {
			return 0, false
		}
		return int64(userID), true

	case uint64:
		if userID > uint64(1<<63-1) {
			return 0, false
		}
		return int64(userID), true

	case uint32:
		return int64(userID), true

	default:
		return 0, false
	}
}
