package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type stubKBService struct {
	interfaces.KnowledgeBaseService
	get func(context.Context, string) (*types.KnowledgeBase, error)
}

func (s *stubKBService) GetKnowledgeBaseByID(ctx context.Context, id string) (*types.KnowledgeBase, error) {
	return s.get(ctx, id)
}

func newKBLookupCtx(t *testing.T, tenantID uint64, id string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set(types.TenantIDContextKey.String(), tenantID)
	return c
}

const testSkillTenantID uint64 = 1
