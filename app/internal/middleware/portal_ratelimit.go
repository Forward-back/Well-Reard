package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "gaze/internal/errors"
)

// portal_ratelimit.go — per-IP sliding-window limiter for the consumer
// portal surface (§2.13 P3-3). The /portal group sits outside the
// internal Auth chain and /portal/anonymous-token is unauthenticated,
// so IP limiting is the only backstop against token spam and chat
// flooding. In-memory only (Lite-compatible); the anonymous daily quota
// in the chat handler already bounds per-session volume.
//
// Budget is configurable via PORTAL_RATE_LIMIT_PER_MINUTE (default 60).
// 60/min covers a real user browsing the portal + a chat stream, while
// clamping a single IP well below LLM-cost-abuse territory. Invalid or
// non-positive values fall back to the default.

const portalRateLimitDefaultPerMinute = 60

// portalRateLimitWindow is the rolling window per IP: one minute.
const portalRateLimitWindow = time.Minute

var portalIPLimiter = newIPRateLimiter(
	portalRateLimitWindow, portalRateLimitPerMinute())

func portalRateLimitPerMinute() int {
	raw := strings.TrimSpace(os.Getenv("PORTAL_RATE_LIMIT_PER_MINUTE"))
	if raw == "" {
		return portalRateLimitDefaultPerMinute
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return portalRateLimitDefaultPerMinute
	}
	return n
}

// PortalIPRateLimit returns a Gin middleware that rate-limits all portal
// routes by client IP. 429 flows through the project AppError type so it
// is rendered by the same error middleware as the rest of the API.
func PortalIPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !portalIPLimiter.allow(c.ClientIP()) {
			c.Error(&apperrors.AppError{
				Code:     apperrors.ErrTooManyRequests,
				Message:  "请求过于频繁,请稍后再试",
				HTTPCode: http.StatusTooManyRequests,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
