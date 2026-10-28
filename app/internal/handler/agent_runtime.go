package handler

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gaze/internal/application/service"
	apperrors "gaze/internal/errors"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// AgentRuntimeHandler exposes the read-only agent surface used by external
// integrations (MCP clients, customer-service gateways). Agent authoring stays
// disabled in this deployment.
type AgentRuntimeHandler struct {
	service interfaces.CustomAgentService
}

func NewAgentRuntimeHandler(service interfaces.CustomAgentService) *AgentRuntimeHandler {
	return &AgentRuntimeHandler{service: service}
}

// ListAgents godoc
// @Summary      列出可用智能体（外部接入只读）
// @Description  供外部客服、网页问答和 MCP 客户端解析智能体；不包含作者面写操作。
// @Tags         Agents
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "智能体列表"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents [get]
func (h *AgentRuntimeHandler) ListAgents(c *gin.Context) {
	agents, err := h.service.ListAgents(c.Request.Context())
	if err != nil {
		_ = c.Error(apperrors.NewInternalServerError(err.Error()))
		return
	}
	out := make([]agentRuntimeView, 0, len(agents))
	for _, agent := range agents {
		out = append(out, newAgentRuntimeView(agent))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// GetAgent godoc
// @Summary      获取智能体详情（外部接入只读）
// @Tags         Agents
// @Produce      json
// @Param        id   path      string  true  "智能体 ID"
// @Success      200  {object}  map[string]interface{}  "智能体详情"
// @Failure      404  {object}  map[string]interface{}    "智能体不存在"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [get]
// SuggestedQuestions godoc
// @Summary      获取智能体推荐问题
// @Tags         Agents
// @Produce      json
// @Param        id   path      string  true  "智能体 ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/suggested-questions [get]
func (h *AgentRuntimeHandler) SuggestedQuestions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	var tagScopes []types.TagScope
	if raw := strings.TrimSpace(c.Query("tag_scopes")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &tagScopes); err != nil {
			_ = c.Error(apperrors.NewBadRequestError("invalid tag_scopes"))
			return
		}
	}
	questions, err := h.service.GetSuggestedQuestions(
		c.Request.Context(),
		c.Param("id"),
		splitComma(c.Query("knowledge_base_ids")),
		splitComma(c.Query("knowledge_ids")),
		tagScopes,
		limit,
	)
	if err != nil {
		if stderrors.Is(err, service.ErrAgentNotFound) {
			_ = c.Error(apperrors.NewNotFoundError("agent not found"))
			return
		}
		_ = c.Error(apperrors.NewInternalServerError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"questions": questions}})
}

func splitComma(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (h *AgentRuntimeHandler) GetAgent(c *gin.Context) {
	agent, err := h.service.GetAgentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if stderrors.Is(err, service.ErrAgentNotFound) {
			_ = c.Error(apperrors.NewNotFoundError("agent not found"))
			return
		}
		_ = c.Error(apperrors.NewInternalServerError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": newAgentRuntimeView(agent)})
}

type agentRuntimeView struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"`
	IsBuiltin   bool                   `json:"is_builtin"`
	Config      agentRuntimeConfigView `json:"config"`
}

type agentRuntimeConfigView struct {
	AgentMode           string   `json:"agent_mode"`
	AgentType           string   `json:"agent_type,omitempty"`
	KBSelectionMode     string   `json:"kb_selection_mode"`
	KnowledgeBases      []string `json:"knowledge_bases"`
	WebSearchEnabled    bool     `json:"web_search_enabled"`
	WebSearchProviderID string   `json:"web_search_provider_id,omitempty"`
}

func newAgentRuntimeView(agent *types.CustomAgent) agentRuntimeView {
	if agent == nil {
		return agentRuntimeView{}
	}
	return agentRuntimeView{
		ID:          agent.ID,
		Name:        agent.Name,
		Description: agent.Description,
		Avatar:      agent.Avatar,
		IsBuiltin:   agent.IsBuiltin,
		Config: agentRuntimeConfigView{
			AgentMode:           agent.Config.AgentMode,
			AgentType:           agent.Config.AgentType,
			KBSelectionMode:     agent.Config.KBSelectionMode,
			KnowledgeBases:      agent.Config.KnowledgeBases,
			WebSearchEnabled:    agent.Config.WebSearchEnabled,
			WebSearchProviderID: agent.Config.WebSearchProviderID,
		},
	}
}
