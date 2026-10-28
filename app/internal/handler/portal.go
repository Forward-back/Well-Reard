package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gaze/internal/application/repository"
	"gaze/internal/engine"
	apperrors "gaze/internal/errors"
	"gaze/internal/logger"
	"gaze/internal/middleware"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// PortalHandler serves the consumer portal surface (§2.13): anonymous token
// issuance, public KB listing, portal chat (SSE) and session history.
type PortalHandler struct {
	kbService  interfaces.KnowledgeBaseService
	ragEngine  engine.RAGEngine
	portalRepo *repository.PortalSessionRepository
	aclService interfaces.KBAclService
}

// NewPortalHandler creates a new portal handler.
func NewPortalHandler(
	kbService interfaces.KnowledgeBaseService,
	ragEngine engine.RAGEngine,
	portalRepo *repository.PortalSessionRepository,
	aclService interfaces.KBAclService,
) *PortalHandler {
	return &PortalHandler{
		kbService: kbService, ragEngine: ragEngine, portalRepo: portalRepo, aclService: aclService,
	}
}

// portalKBResponse is the portal-facing KB projection: only consumer-safe
// fields leave the wire (no internal IDs, model bindings or configs).
type portalKBResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// IssueAnonymousToken godoc
// @Summary      签发匿名门户令牌
// @Description  为门户匿名体验签发短期令牌(无身份,仅关联会话),P3 登录后由消费者令牌替代
// @Tags         门户
// @Produce      json
// @Success      200  {object} map[string]interface{}
// @Router       /portal/anonymous-token [post]
func (h *PortalHandler) IssueAnonymousToken(c *gin.Context) {
	ctx := c.Request.Context()
	secret := middleware.PortalSecret()
	if secret == "" {
		middleware.WritePortalError(c, http.StatusServiceUnavailable, "门户服务未配置")
		return
	}
	token, err := middleware.IssuePortalAnonymousToken(
		secret, uuid.New().String(), 24*time.Hour,
	)
	if err != nil {
		logger.Errorf(ctx, "issue portal anonymous token failed: %v", err)
		middleware.WritePortalError(c, http.StatusInternalServerError, "签发令牌失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":      token,
			"token_type": "portal_anonymous",
			"expires_in": int64((24 * time.Hour).Seconds()),
		},
	})
}

// ListPublicKnowledgeBases godoc
// @Summary      门户公开知识库列表
// @Description  列出消费者门户可见的公开知识库(visibility=public)
// @Tags         门户
// @Produce      json
// @Success      200  {object} map[string]interface{}
// @Router       /portal/kbs [get]
func (h *PortalHandler) ListPublicKnowledgeBases(c *gin.Context) {
	ctx := c.Request.Context()
	kbs, err := h.kbService.ListPublicKnowledgeBases(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		middleware.WritePortalError(c, http.StatusInternalServerError, "获取公开知识库失败")
		return
	}
	resp := make([]portalKBResponse, 0, len(kbs))
	for _, kb := range kbs {
		if kb == nil {
			continue
		}
		resp = append(resp, portalKBResponse{
			ID:          kb.ID,
			Name:        kb.Name,
			Description: kb.Description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ── 匿名限流: 每匿名会话每日 5 次提问(§2.13) ──────────────────

const portalAnonymousDailyQuota = 5

type portalQuotaState struct {
	date  string // YYYY-MM-DD
	count int
}

var (
	portalQuotaMu sync.Mutex
	portalQuota   = map[string]*portalQuotaState{}
)

// consumeAnonymousQuota returns true when the anonymous session may ask
// another question today. State is in-memory (Lite-compatible); a restart
// resets the counters, acceptable for the P3 skeleton.
func consumeAnonymousQuota(sessionID string) bool {
	portalQuotaMu.Lock()
	defer portalQuotaMu.Unlock()
	today := time.Now().Format("2006-01-02")
	st := portalQuota[sessionID]
	if st == nil || st.date != today {
		portalQuota[sessionID] = &portalQuotaState{date: today, count: 1}
		return true
	}
	if st.count >= portalAnonymousDailyQuota {
		return false
	}
	st.count++
	return true
}

// ── 门户问答 ──────────────────────────────────────────────

// portalChatRequest is the portal chat body.
type portalChatRequest struct {
	Question string `json:"question"`
	KBID     string `json:"kb_id"` // 可选: 限定单个公开 KB;空=全部公开 KB
}

// sanitizeRefs projects engine references down to titles only (§2.13: no
// file names, doc numbers or org metadata leave the portal).
// sanitizeRefs builds the consumer-facing reference strings（§2.14 portal 透出）:
// 标题 + 文号 + 发文机构 + 日期 + 「已废止」标签。密级/部门/自定义字段一律
// 不透出（confidential 已被外发闸拦截，不会出现在 refs 里）。
func sanitizeRefs(refs []*types.SearchResult) []string {
	out := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, r := range refs {
		if r == nil || r.KnowledgeTitle == "" {
			continue
		}
		if _, ok := seen[r.KnowledgeTitle]; ok {
			continue
		}
		seen[r.KnowledgeTitle] = struct{}{}
		parts := []string{r.KnowledgeTitle}
		meta := r.Metadata
		if n := meta["doc_number"]; n != "" {
			parts = append(parts, "文号:"+n)
		}
		if o := meta["org_name"]; o != "" {
			parts = append(parts, o)
		}
		if d := meta["doc_date"]; d != "" {
			parts = append(parts, d)
		}
		if meta["lifecycle_status"] == "expired" {
			parts = append(parts, "已废止")
		}
		out = append(out, strings.Join(parts, " · "))
	}
	return out
}

// writeSSE writes one SSE data frame to the portal client.
func writeSSE(c *gin.Context, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// Chat godoc
// @Summary      门户问答(SSE 流式)
// @Description  消费者门户问答: 检索范围=public KB 白名单,引用脱敏仅标题
// @Tags         门户
// @Produce      text/event-stream
// @Param        request body portalChatRequest true "{\"question\": \"...\"}"
// @Router       /portal/chat [post]
func (h *PortalHandler) Chat(c *gin.Context) {
	ctx := c.Request.Context()
	token := middleware.PortalTokenFromContext(c)
	if token == nil {
		middleware.WritePortalError(c, http.StatusUnauthorized, "门户令牌无效")
		return
	}
	var req portalChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Question == "" {
		middleware.WritePortalError(c, http.StatusBadRequest, "question 必填")
		return
	}
	if h.ragEngine == nil {
		middleware.WritePortalError(c, http.StatusServiceUnavailable, "问答引擎未配置")
		return
	}

	// 匿名限次: 超过每日配额引导登录(P3 接入登录后此处放行 consumer)
	if token.Kind == "anonymous" && !consumeAnonymousQuota(token.SessionID) {
		middleware.WritePortalError(c, http.StatusTooManyRequests, "今日匿名提问次数已用完,请登录后继续")
		return
	}

	// 检索范围: 全部 public KB(或请求指定的单个公开 KB)
	kbs, err := h.kbService.ListPublicKnowledgeBases(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		middleware.WritePortalError(c, http.StatusInternalServerError, "获取公开知识库失败")
		return
	}
	kbIDs := make([]string, 0, len(kbs))
	for _, kb := range kbs {
		if kb == nil {
			continue
		}
		if req.KBID != "" && kb.ID != req.KBID {
			continue
		}
		kbIDs = append(kbIDs, kb.ID)
	}
	if len(kbIDs) == 0 {
		middleware.WritePortalError(c, http.StatusNotFound, "暂无公开知识库")
		return
	}

	// 会话落库(匿名 token 的 session_id 即会话 ID)
	sessionID := token.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	if _, err := h.portalRepo.GetSessionByID(ctx, sessionID); err != nil {
		session := &types.PortalSession{
			ID:            sessionID,
			TokenKind:     token.Kind,
			IdentityKey:   token.IdentityKey,
			KBIDs:         kbIDs,
			FirstQuestion: req.Question,
		}
		if err := h.portalRepo.CreateSession(ctx, session); err != nil {
			logger.Warnf(ctx, "create portal session failed: %v", err)
		}
	}
	_ = h.portalRepo.AppendMessage(ctx, &types.PortalMessage{
		SessionID: sessionID, Role: "user", Content: req.Question,
	})

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	if h.aclService == nil {
		middleware.WritePortalError(c, http.StatusInternalServerError, "问答服务暂不可用")
		return
	}
	// Anonymous portal principal: only subject=all allow rules match, deny all
	// hides, and department/user rules do not apply.
	aclScope, scopeErr := h.aclService.RetrievalScope(ctx, kbIDs)
	if scopeErr != nil {
		middleware.WritePortalError(c, http.StatusInternalServerError, "问答服务暂不可用")
		return
	}

	answer, refs, err := h.ragEngine.StreamChat(
		ctx, req.Question, kbIDs, aclScope, nil,
		// 消费者门户密级闸(§2.14 B):只召回 internal,confidential 一律不出
		[]string{"internal"},
		func(delta string) {
			if werr := writeSSE(c, gin.H{"type": "delta", "content": delta}); werr != nil {
				logger.Warnf(ctx, "portal SSE write failed: %v", werr)
			}
		},
		func(r []*types.SearchResult) {
			if werr := writeSSE(c, gin.H{"type": "references", "references": sanitizeRefs(r)}); werr != nil {
				logger.Warnf(ctx, "portal SSE write failed: %v", werr)
			}
		},
		nil,
	)
	if err != nil {
		logger.Warnf(ctx, "portal chat engine error: %v", err)
		_ = writeSSE(c, gin.H{"type": "error", "message": "问答服务暂时不可用,请稍后重试"})
		return
	}
	_ = h.portalRepo.AppendMessage(ctx, &types.PortalMessage{
		SessionID: sessionID, Role: "assistant", Content: answer,
		Refs: sanitizeRefs(refs),
	})
	_ = writeSSE(c, gin.H{"type": "done", "session_id": sessionID})
}

// Sessions godoc
// @Summary      门户会话历史
// @Description  anonymous=单会话消息;consumer=按身份聚合的会话列表
// @Tags         门户
// @Produce      json
// @Success      200  {object} map[string]interface{}
// @Router       /portal/sessions [get]
func (h *PortalHandler) Sessions(c *gin.Context) {
	ctx := c.Request.Context()
	token := middleware.PortalTokenFromContext(c)
	if token == nil {
		middleware.WritePortalError(c, http.StatusUnauthorized, "门户令牌无效")
		return
	}

	// consumer: 历史按身份聚合(§2.13 P3-2),返回会话列表,消息按需单查
	if token.Kind == "consumer" && token.IdentityKey != "" {
		sessions, err := h.portalRepo.ListSessionsByIdentity(ctx, token.IdentityKey, 50)
		if err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			middleware.WritePortalError(c, http.StatusInternalServerError, "获取会话失败")
			return
		}
		if sessions == nil {
			sessions = []*types.PortalSession{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"sessions": sessions, "messages": []interface{}{},
		}})
		return
	}

	if token.SessionID == "" {
		middleware.WritePortalError(c, http.StatusUnauthorized, "门户令牌无效")
		return
	}
	session, err := h.portalRepo.GetSessionByID(ctx, token.SessionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"session": nil, "messages": []interface{}{}}})
		return
	}
	messages, err := h.portalRepo.ListMessages(ctx, token.SessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		middleware.WritePortalError(c, http.StatusInternalServerError, "获取会话失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"session":  session,
		"messages": messages,
	}})
}

// SessionMessages godoc
// @Summary      单会话消息
// @Description  返回指定门户会话的消息;consumer 只能查自己身份下的会话
// @Tags         门户
// @Produce      json
// @Param        id path string true "会话 ID"
// @Success      200  {object} map[string]interface{}
// @Router       /portal/sessions/{id}/messages [get]
func (h *PortalHandler) SessionMessages(c *gin.Context) {
	ctx := c.Request.Context()
	token := middleware.PortalTokenFromContext(c)
	if token == nil {
		middleware.WritePortalError(c, http.StatusUnauthorized, "门户令牌无效")
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		middleware.WritePortalError(c, http.StatusBadRequest, "会话 ID 必填")
		return
	}
	session, err := h.portalRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		middleware.WritePortalError(c, http.StatusNotFound, "会话不存在")
		return
	}
	// 访问控制: consumer 只能访问自己身份下的会话;anonymous 只能访问令牌绑定的会话
	if token.Kind == "consumer" {
		if session.IdentityKey == "" || session.IdentityKey != token.IdentityKey {
			middleware.WritePortalError(c, http.StatusForbidden, "无权访问该会话")
			return
		}
	} else if session.ID != token.SessionID {
		middleware.WritePortalError(c, http.StatusForbidden, "无权访问该会话")
		return
	}
	messages, err := h.portalRepo.ListMessages(ctx, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		middleware.WritePortalError(c, http.StatusInternalServerError, "获取会话消息失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": messages})
}

// Logout godoc
// @Summary      消费者注销
// @Description  consumer 注销后会话脱钩匿名化(§2.13 P3-7):保留历史但清空身份关联
// @Tags         门户
// @Produce      json
// @Success      200  {object} map[string]interface{}
// @Router       /portal/logout [post]
func (h *PortalHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()
	token := middleware.PortalTokenFromContext(c)
	if token == nil {
		middleware.WritePortalError(c, http.StatusUnauthorized, "门户令牌无效")
		return
	}
	if token.Kind == "consumer" && token.IdentityKey != "" {
		n, err := h.portalRepo.AnonymizeSessionsByIdentity(ctx, token.IdentityKey)
		if err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			middleware.WritePortalError(c, http.StatusInternalServerError, "注销失败")
			return
		}
		logger.Infof(ctx, "portal logout: anonymized %d sessions for identity %s", n, token.IdentityKey)
	}
	// anonymous 无身份,无历史脱钩需求
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"logged_out": true}})
}

// ── 运营面(Admin+,§2.13) ──────────────────────────────────

// portalOpsSession is the operations projection: identity stays de-identified.
type portalOpsSession struct {
	ID            string     `json:"id"`
	TokenKind     string     `json:"token_kind"`
	IdentityKey   string     `json:"identity_key"`
	FirstQuestion string     `json:"first_question"`
	MessageCount  int        `json:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ListPortalSessionsForOps godoc
// @Summary      门户会话列表(运营)
// @Description  Admin+ 查看消费者门户会话(身份脱敏),用于内容优化与问题追踪
// @Tags         门户
// @Produce      json
// @Success      200  {object} map[string]interface{}
// @Router       /portal-analytics/sessions [get]
func (h *PortalHandler) ListPortalSessionsForOps(c *gin.Context) {
	ctx := c.Request.Context()
	offset, limit := 0, 50
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v >= 0 {
		offset = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && v > 0 {
		limit = v
	}
	sessions, err := h.portalRepo.ListSessionsForOps(ctx, offset, limit)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(apperrors.NewInternalServerError("获取门户会话失败"))
		return
	}
	resp := make([]portalOpsSession, 0, len(sessions))
	for _, s := range sessions {
		if s == nil {
			continue
		}
		resp = append(resp, portalOpsSession{
			ID:            s.ID,
			TokenKind:     s.TokenKind,
			IdentityKey:   s.IdentityKey,
			FirstQuestion: s.FirstQuestion,
			MessageCount:  s.MessageCount,
			LastMessageAt: s.LastMessageAt,
			CreatedAt:     s.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ListPortalSessionMessages godoc
// @Summary      门户会话消息(运营)
// @Description  Admin+ 查看单个门户会话的消息全文(身份脱敏)
// @Tags         门户
// @Produce      json
// @Param        id path string true "会话 ID"
// @Success      200  {object} map[string]interface{}
// @Router       /portal-analytics/sessions/{id}/messages [get]
func (h *PortalHandler) ListPortalSessionMessages(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")
	if sessionID == "" {
		c.Error(apperrors.NewBadRequestError("session id is required"))
		return
	}
	if _, err := h.portalRepo.GetSessionByID(ctx, sessionID); err != nil {
		c.Error(apperrors.NewNotFoundError("portal session not found"))
		return
	}
	messages, err := h.portalRepo.ListMessages(ctx, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(apperrors.NewInternalServerError("获取门户会话消息失败"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": messages})
}
