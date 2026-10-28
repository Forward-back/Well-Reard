package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaze/internal/capabilities"
	"gaze/internal/logger"
	"gaze/internal/types"
)

// RequireCapability gates a route on one capability key derived from the
// caller's company and department roles. This is a coarse page/API-surface
// guard for new management surfaces; resource-level guards (department shares,
// KB grants, ACL) remain the security boundary.
//
// API-key principals are handled by APIKeyGate and short-circuit here.
// System administrators bypass the membership capability projection.
func RequireCapability(key string) gin.HandlerFunc {
	return RequireAnyCapability(key)
}

// RequireTenantKVCapability maps /tenants/kv/:key writes onto the settings
// capability that owns that partition. Unknown keys pass through so the handler
// can return its normal 400.
func RequireTenantKVCapability() gin.HandlerFunc {
	keyCaps := map[string]string{
		"web-search-config":     capabilities.SettingsIntegrationsManage,
		"parser-engine-config":  capabilities.SettingsInfraManage,
		"storage-engine-config": capabilities.SettingsInfraManage,
		"chat-history-config":   capabilities.SettingsWorkspaceManage,
		"retrieval-config":      capabilities.SettingsWorkspaceManage,
		"memory-config":         capabilities.SettingsWorkspaceManage,
		"engine-sync-config":    capabilities.SettingsWorkspaceManage,
		"reading-config":        capabilities.SettingsModelsManage,
	}
	return func(c *gin.Context) {
		key, ok := keyCaps[c.Param("key")]
		if !ok {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		if _, apiKey := types.TenantAPIKeyScopeFromContext(ctx); apiKey {
			c.Next()
			return
		}
		if types.IsSystemAdminFromContext(ctx) {
			c.Next()
			return
		}
		caller := types.CallerFromContext(ctx).Normalize()
		caps := capabilities.MustDefault().CapsFor(string(caller.Role), string(caller.DepartmentRole), false)
		if capabilities.Has(caps, key) {
			c.Next()
			return
		}
		logger.Warnf(ctx, "[rbac] capability %q required for tenant kv %q: user=%s",
			key, c.Param("key"), caller.UserID)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Forbidden: missing capability",
			"capability": key,
		})
		c.Abort()
	}
}

// RequireAnyCapability allows a request when at least one listed capability
// is granted. Service-layer scope checks still narrow the data range.
func RequireAnyCapability(keys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
			c.Next()
			return
		}
		if types.IsSystemAdminFromContext(ctx) {
			c.Next()
			return
		}
		caller := types.CallerFromContext(ctx).Normalize()
		caps := capabilities.MustDefault().CapsFor(string(caller.Role), string(caller.DepartmentRole), false)
		for _, key := range keys {
			if capabilities.Has(caps, key) {
				c.Next()
				return
			}
		}
		logger.Warnf(ctx, "[rbac] one of capabilities %v required: user=%s tenant=%d path=%s",
			keys, caller.UserID, caller.TenantID, c.Request.URL.Path)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Forbidden: missing capability",
			"capability": keys,
		})
		c.Abort()
	}
}
