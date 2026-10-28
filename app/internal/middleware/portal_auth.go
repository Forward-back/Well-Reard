package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gaze/internal/errors"
)

// Portal token 骨架(§2.13 P2): 消费者门户独立凭证体系,与内部用户 JWT 物理隔离。
//
// token 格式: portal_v1.<base64(payload)>.<base64(hmac-sha256(payload, secret))>
// payload: {"kind":"anonymous","session_id":"...","exp":unix}
// P3 扩展: kind=consumer 携带 portal 用户 ID(微信/手机号登录后签发)。
//
// 密钥来源: PORTAL_JWT_SECRET 环境变量;未配置时回退 JWT_SECRET(个人部署不强制)。

const portalTokenPrefix = "portal_v1."

type portalTokenPayload struct {
	Kind        string `json:"kind"`
	SessionID   string `json:"session_id,omitempty"`
	IdentityKey string `json:"identity_key,omitempty"` // 脱敏身份标识(P3 consumer 用)
	Exp         int64  `json:"exp"`
}

// PortalSecret returns the portal signing secret from the environment.
func PortalSecret() string {
	if s := strings.TrimSpace(os.Getenv("PORTAL_JWT_SECRET")); s != "" {
		return s
	}
	return strings.TrimSpace(os.Getenv("JWT_SECRET"))
}

// IssuePortalAnonymousToken signs a short-lived anonymous portal token.
// The token carries no identity — it only ties a device/session to its
// conversation history.
func IssuePortalAnonymousToken(secret, sessionID string, ttl time.Duration) (string, error) {
	payload, err := json.Marshal(portalTokenPayload{
		Kind:      "anonymous",
		SessionID: sessionID,
		Exp:       time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	sig := signPortalPayload(secret, payload)
	return portalTokenPrefix +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// IssuePortalConsumerToken signs a portal token for a logged-in consumer
// (§2.13 P3-2). identityKey is the de-identified consumer key (phone hash
// etc. — never the raw identifier); no session id is carried because
// consumer history aggregates by identity on the portal side. The token
// is minted by the P3-1 login flow (WeChat/phone, blocked on external
// credentials) — this function is the reserved issuance path.
func IssuePortalConsumerToken(secret, identityKey string, ttl time.Duration) (string, error) {
	if identityKey == "" {
		return "", errors.NewValidationError("identity key is required")
	}
	payload, err := json.Marshal(portalTokenPayload{
		Kind:        "consumer",
		IdentityKey: identityKey,
		Exp:         time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	sig := signPortalPayload(secret, payload)
	return portalTokenPrefix +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyPortalToken parses and verifies a portal token, returning its payload.
func VerifyPortalToken(secret, token string) (*portalTokenPayload, error) {
	if !strings.HasPrefix(token, portalTokenPrefix) {
		return nil, errors.NewValidationError("invalid portal token")
	}
	rest := strings.TrimPrefix(token, portalTokenPrefix)
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 {
		return nil, errors.NewValidationError("invalid portal token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.NewValidationError("invalid portal token")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.NewValidationError("invalid portal token")
	}
	expected := signPortalPayload(secret, payload)
	if subtle.ConstantTimeCompare(sig, expected) != 1 {
		return nil, errors.NewForbiddenError("invalid portal token signature")
	}
	var p portalTokenPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, errors.NewValidationError("invalid portal token payload")
	}
	if time.Now().Unix() > p.Exp {
		return nil, errors.NewForbiddenError("portal token expired")
	}
	return &p, nil
}

func signPortalPayload(secret string, payload []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return mac.Sum(nil)
}

// PortalAuthRequired guards portal routes with a valid portal token.
// The payload is placed into the request context for downstream handlers.
func PortalAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := PortalSecret()
		if secret == "" {
			c.Error(errors.NewInternalServerError("portal secret is not configured"))
			c.Abort()
			return
		}
		token := strings.TrimSpace(c.GetHeader("Authorization"))
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-Portal-Token"))
		}
		if token == "" {
			c.Error(errors.NewForbiddenError("portal token is required"))
			c.Abort()
			return
		}
		payload, err := VerifyPortalToken(secret, token)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		c.Set("portal_token", payload)
		c.Next()
	}
}

// PortalTokenFromContext returns the verified portal token payload.
func PortalTokenFromContext(c *gin.Context) *portalTokenPayload {
	if v, ok := c.Get("portal_token"); ok {
		if p, ok := v.(*portalTokenPayload); ok {
			return p
		}
	}
	return nil
}

// WritePortalError writes a portal-facing JSON error body in the same shape
// the internal API uses (success/message), without leaking internal details.
func WritePortalError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message, "code": status})
}

var _ = http.StatusOK
