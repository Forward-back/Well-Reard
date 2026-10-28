package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// WebSearchRuntimeHandler exposes the read-only provider list used by the
// chat input to decide whether web search is available. Provider management
// stays disabled.
type WebSearchRuntimeHandler struct {
	repo interfaces.WebSearchProviderRepository
}

func NewWebSearchRuntimeHandler(repo interfaces.WebSearchProviderRepository) *WebSearchRuntimeHandler {
	return &WebSearchRuntimeHandler{repo: repo}
}

type webSearchProviderView struct {
	ID          string                          `json:"id"`
	Name        string                          `json:"name"`
	Provider    types.WebSearchProviderType     `json:"provider"`
	Description string                          `json:"description,omitempty"`
	Parameters  webSearchProviderParametersView `json:"parameters"`
	IsDefault   bool                            `json:"is_default"`
	Credentials map[string]credentialView       `json:"credentials"`
	CreatedAt   any                             `json:"created_at,omitempty"`
	UpdatedAt   any                             `json:"updated_at,omitempty"`
}

type webSearchProviderParametersView struct {
	EngineID    string            `json:"engine_id,omitempty"`
	BaseURL     string            `json:"base_url,omitempty"`
	ProxyURL    string            `json:"proxy_url,omitempty"`
	ExtraConfig map[string]string `json:"extra_config,omitempty"`
}

type credentialView struct {
	Configured bool `json:"configured"`
}

func (h *WebSearchRuntimeHandler) ListProviders(c *gin.Context) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	providers, err := h.repo.List(c.Request.Context(), tenantID)
	if err != nil {
		_ = c.Error(apperrors.NewInternalServerError(err.Error()))
		return
	}
	out := make([]webSearchProviderView, 0, len(providers))
	for _, p := range providers {
		out = append(out, webSearchProviderView{
			ID:          p.ID,
			Name:        p.Name,
			Provider:    p.Provider,
			Description: p.Description,
			Parameters: webSearchProviderParametersView{
				EngineID:    p.Parameters.EngineID,
				BaseURL:     p.Parameters.BaseURL,
				ProxyURL:    p.Parameters.ProxyURL,
				ExtraConfig: p.Parameters.ExtraConfig,
			},
			IsDefault:   p.IsDefault,
			Credentials: map[string]credentialView{"api_key": {Configured: p.Parameters.APIKey != ""}},
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}
