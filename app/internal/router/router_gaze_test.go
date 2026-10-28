package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apprepo "gaze/internal/application/repository"
	"gaze/internal/config"
	"gaze/internal/handler"
	"gaze/internal/middleware"
	"gaze/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubGazeKBLookup struct {
	kbs map[string]*types.KnowledgeBase
}

func (s *stubGazeKBLookup) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	if kb, ok := s.kbs[id]; ok {
		return kb, nil
	}
	return nil, apprepo.ErrKnowledgeBaseNotFound
}

func newGazeRouteTestEngine(t *testing.T, callerTenantID uint64, kbLookup *stubGazeKBLookup) *gin.Engine {
	return newKBRouteTestEngine(t, callerTenantID, kbLookup, nil, func(r *gin.RouterGroup, guards *rbacGuards) {
		RegisterGazePageRoutes(r, &handler.GazePageHandler{}, guards)
	})
}

func newInitializationRouteTestEngine(t *testing.T, callerTenantID uint64, kbLookup *stubGazeKBLookup) *gin.Engine {
	return newKBRouteTestEngine(t, callerTenantID, kbLookup, nil, func(r *gin.RouterGroup, guards *rbacGuards) {
		RegisterInitializationRoutes(r, &handler.InitializationHandler{}, guards)
	})
}

func newKBRouteTestEngine(
	t *testing.T,
	callerTenantID uint64,
	kbLookup *stubGazeKBLookup,
	apiKeyScope *types.TenantAPIKeyScope,
	register func(r *gin.RouterGroup, guards *rbacGuards),
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	enabled := true
	cfg := &config.Config{
		Tenant: &config.TenantConfig{EnableRBAC: &enabled},
	}
	guards := &rbacGuards{
		cfg:       cfg,
		kbService: kbLookup,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, callerTenantID)
		role := types.TenantRoleViewer
		if apiKeyScope != nil {
			ctx = types.WithTenantAPIKeyScope(ctx, *apiKeyScope)
			if apiKeyScope.FullAccess {
				role = types.TenantRoleOwner
			}
		}
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenantID)
		c.Next()
	})

	register(r.Group("/api/v1"), guards)
	return r
}

func tenantKBLookupFixture() *stubGazeKBLookup {
	return &stubGazeKBLookup{
		kbs: map[string]*types.KnowledgeBase{
			"kb-allowed": {ID: "kb-allowed", TenantID: 1, Type: types.KnowledgeBaseTypeGaze},
			"kb-other":   {ID: "kb-other", TenantID: 1, Type: types.KnowledgeBaseTypeGaze},
		},
	}
}

func TestInitializationConfigRouteDenyCrossTenantKB(t *testing.T) {
	kbLookup := &stubGazeKBLookup{
		kbs: map[string]*types.KnowledgeBase{
			"kb-victim": {ID: "kb-victim", TenantID: 999},
		},
	}
	engine := newInitializationRouteTestEngine(t, 1, kbLookup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/initialization/config/kb-victim", nil)
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

func TestGazeReadRoutesDenyCrossTenantKB(t *testing.T) {
	kbLookup := &stubGazeKBLookup{
		kbs: map[string]*types.KnowledgeBase{
			"kb-victim": {
				ID:       "kb-victim",
				TenantID: 999,
				Type:     types.KnowledgeBaseTypeGaze,
			},
		},
	}
	engine := newGazeRouteTestEngine(t, 1, kbLookup)

	paths := []string{
		"/api/v1/knowledgebase/kb-victim/gaze/pages",
		"/api/v1/knowledgebase/kb-victim/gaze/pages/secret-page",
		"/api/v1/knowledgebase/kb-victim/gaze/folders",
		"/api/v1/knowledgebase/kb-victim/gaze/index",
		"/api/v1/knowledgebase/kb-victim/gaze/graph",
		"/api/v1/knowledgebase/kb-victim/gaze/stats",
		"/api/v1/knowledgebase/kb-victim/gaze/search?q=test",
		"/api/v1/knowledgebase/kb-victim/gaze/lint",
		"/api/v1/knowledgebase/kb-victim/gaze/issues",
		"/api/v1/knowledgebase/kb-victim/gaze/revisions/secret-page",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			engine.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		})
	}
}

func TestGazeOperationLogRouteIsRemoved(t *testing.T) {
	engine := newGazeRouteTestEngine(t, 1, tenantKBLookupFixture())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledgebase/kb-allowed/gaze/log", nil)
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

func TestInitializationWriteRoutesDenyOutOfScopeAPIKeyKB(t *testing.T) {
	kbLookup := tenantKBLookupFixture()
	scope := &types.TenantAPIKeyScope{
		KnowledgeBaseIDs: types.StringArray{"kb-allowed"},
		Capabilities:     types.StringArray{string(types.APIKeyCapabilityManageKnowledgeBases)},
	}
	engine := newKBRouteTestEngine(t, 1, kbLookup, scope, func(r *gin.RouterGroup, guards *rbacGuards) {
		RegisterInitializationRoutes(r, &handler.InitializationHandler{}, guards)
	})

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPut, "/api/v1/initialization/config/kb-other", `{}`},
		{http.MethodPost, "/api/v1/initialization/initialize/kb-other", `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		})
	}
}

func TestGazeWriteRoutesDenyOutOfScopeAPIKeyKB(t *testing.T) {
	kbLookup := tenantKBLookupFixture()
	scope := &types.TenantAPIKeyScope{
		KnowledgeBaseIDs: types.StringArray{"kb-allowed"},
		Capabilities:     types.StringArray{string(types.APIKeyCapabilityIngest)},
	}
	engine := newKBRouteTestEngine(t, 1, kbLookup, scope, func(r *gin.RouterGroup, guards *rbacGuards) {
		RegisterGazePageRoutes(r, &handler.GazePageHandler{}, guards)
	})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/knowledgebase/kb-other/gaze/pages"},
		{http.MethodPut, "/api/v1/knowledgebase/kb-other/gaze/pages/some-page"},
		{http.MethodDelete, "/api/v1/knowledgebase/kb-other/gaze/pages/some-page"},
		{http.MethodPost, "/api/v1/knowledgebase/kb-other/gaze/folders"},
		{http.MethodPost, "/api/v1/knowledgebase/kb-other/gaze/rebuild-links"},
		{http.MethodPost, "/api/v1/knowledgebase/kb-other/gaze/auto-fix"},
		{http.MethodPut, "/api/v1/knowledgebase/kb-other/gaze/issues/1/status"},
		{http.MethodPost, "/api/v1/knowledgebase/kb-other/gaze/revert"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(rec, req)
			require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		})
	}
}
