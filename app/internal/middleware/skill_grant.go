package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gaze/internal/logger"
	"gaze/internal/types/interfaces"
)

// RequireSkillEdit allows callers who may edit the skill in the :id path
// parameter: default skill managers, system admins, or users holding a
// per-skill user/department grant. It is a resource-level guard layered under
// the coarse capability checks used for catalog-wide endpoints.
func RequireSkillEdit(service interfaces.SkillEditorGrantService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}
		ok, err := service.CanEdit(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		if ok {
			c.Next()
			return
		}
		logger.Warnf(c.Request.Context(), "[rbac] skill edit denied: skill=%s path=%s",
			c.Param("id"), c.Request.URL.Path)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Forbidden: skill editor grant required",
		})
		c.Abort()
	}
}
