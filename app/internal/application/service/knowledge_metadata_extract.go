package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gaze/internal/logger"
	"gaze/internal/models/chat"
	"gaze/internal/tracing/langfuse"
	"gaze/internal/types"
	"github.com/hibiken/asynq"
)

// 企业级元数据 LLM 异步抽取（§2.14 赋值双路之二）：
// 摄取完成后 enqueue metadata:extract —— 标题 + 正文首段一次 LLM 调用出
// JSON（6 字段 + 置信度），高置信字段落库并重推 YLSW，低置信字段写入
// CustomMetadata 标黄待人工确认。

const metadataExtractPrompt = `你是文档元数据抽取助手。根据文档标题与正文首段，输出一个 JSON 对象，格式：
{"lifecycle_status":"effective|expired|draft","effective_date":"YYYY-MM-DD或空","expiry_date":"YYYY-MM-DD或空","superseded_by":"替代本文号或空","sensitivity":"internal|confidential","department":"部门名或空","confidence":{"lifecycle_status":0.0到1.0,"effective_date":0.0到1.0,"expiry_date":0.0到1.0,"superseded_by":0.0到1.0,"sensitivity":0.0到1.0,"department":0.0到1.0}}
规则：
1. 只输出 JSON，不要任何其他文字。
2. 无法确定的字段填空串，lifecycle_status 默认 effective，sensitivity 默认 internal。
3. 置信度表示该字段判断的把握，把握不足给 ≤0.6。
4. 废止线索="废止/失效/作废/替代/不再适用"等字样；替代链线索="被X号文替代"。
5. 密级线索="机密/保密/内部资料"等；部门只从正文中明确的机构/部门名判断，不确定留空。

标题：{title}

正文首段：
{head}`

// metadataExtractResult is the LLM extraction output.
type metadataExtractResult struct {
	LifecycleStatus string             `json:"lifecycle_status"`
	EffectiveDate   string             `json:"effective_date"`
	ExpiryDate      string             `json:"expiry_date"`
	SupersededBy    string             `json:"superseded_by"`
	Sensitivity     string             `json:"sensitivity"`
	Department      string             `json:"department"`
	Confidence      map[string]float64 `json:"confidence"`
}

// metadataConfidenceThreshold: below this the field is marked pending for
// manual confirmation instead of being written.
const metadataConfidenceThreshold = 0.6

// extractableMetadataFields enumerates the six extractable fields.
var extractableMetadataFields = []string{
	"lifecycle_status", "effective_date", "expiry_date",
	"superseded_by", "sensitivity", "department",
}

var extractJSONRe = regexp.MustCompile(`\{.*\}`)

// ProcessMetadataExtract handles Asynq metadata:extract tasks.
func (s *knowledgeService) ProcessMetadataExtract(ctx context.Context, t *asynq.Task) error {
	var payload types.MetadataExtractPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal metadata extract payload: %v", err)
		return nil
	}
	if payload.TenantID == 0 || payload.KnowledgeID == "" || payload.KnowledgeBaseID == "" {
		return nil
	}
	ctx = logger.WithRequestID(ctx, payload.RequestId)
	ctx = logger.WithField(ctx, "metadata_extract", payload.KnowledgeID)
	ctx = types.WithExecutionTenant(ctx, payload.TenantID)

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil || tenantInfo == nil {
		return err
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil || knowledge == nil {
		return nil // deleted meanwhile
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, payload.KnowledgeBaseID)
	if err != nil || kb == nil || kb.ID != payload.KnowledgeBaseID || kb.TenantID != payload.TenantID {
		return fmt.Errorf("metadata extract: KB owner changed: %w", asynq.SkipRetry)
	}

	modelID := strings.TrimSpace(kb.SummaryModelID)
	if modelID == "" {
		logger.Infof(ctx, "metadata extract skipped: kb %s has no summary model", kb.ID)
		return nil
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil {
		return err
	}

	title := payload.Title
	if title == "" {
		title = knowledge.Title
	}
	prompt := strings.ReplaceAll(metadataExtractPrompt, "{title}", title)
	prompt = strings.ReplaceAll(prompt, "{head}", payload.ContentHead)

	resp, err := chatModel.Chat(ctx, []chat.Message{{Role: "user", Content: prompt}},
		&chat.ChatOptions{Temperature: 0, MaxTokens: 512})
	if err != nil {
		return err
	}
	result, ok := parseMetadataExtract(resp.Content)
	if !ok {
		logger.Warnf(ctx, "metadata extract: LLM 输出非 JSON: %s", previewText(resp.Content, 200))
		return nil
	}

	// 高置信字段落库;低置信字段进 CustomMetadata 标黄待人工确认
	before := *knowledge
	after := *knowledge
	pending := map[string]string{}
	changed := false
	apply := func(field string, value string) {
		conf, has := result.Confidence[field]
		if !has || conf >= metadataConfidenceThreshold {
			setMetadataField(&after, field, value)
			if getMetadataField(&after, field) != getMetadataField(&before, field) {
				changed = true
			}
			return
		}
		pending[field] = value
	}
	apply("lifecycle_status", strings.TrimSpace(result.LifecycleStatus))
	apply("effective_date", strings.TrimSpace(result.EffectiveDate))
	apply("expiry_date", strings.TrimSpace(result.ExpiryDate))
	apply("superseded_by", strings.TrimSpace(result.SupersededBy))
	apply("sensitivity", strings.TrimSpace(result.Sensitivity))
	apply("department", strings.TrimSpace(result.Department))

	// 提取快照 + 待确认列表写入 CustomMetadata（标黄数据源）
	custom := map[string]any{}
	if len(after.CustomMetadata) > 0 {
		_ = json.Unmarshal(after.CustomMetadata, &custom)
	}
	if len(pending) > 0 {
		custom["metadata_extract_pending"] = pending
	} else {
		delete(custom, "metadata_extract_pending")
	}
	custom["metadata_extract"] = result
	rawCustom, err := json.Marshal(custom)
	if err == nil {
		after.CustomMetadata = rawCustom
	}

	if changed || len(pending) > 0 {
		// 列级更新,不能用整行 UpdateKnowledge:extract 与 summary 并发,
		// 整行写会把 summary 刚落库的 summary_status/description 用陈旧值覆盖。
		updates := map[string]interface{}{
			"lifecycle_status": after.LifecycleStatus,
			"effective_date":   after.EffectiveDate,
			"expiry_date":      after.ExpiryDate,
			"superseded_by":    after.SupersededBy,
			"sensitivity":      after.Sensitivity,
			"department":       after.Department,
			"custom_metadata":  after.CustomMetadata,
			"updated_at":       time.Now(),
		}
		if err := s.repo.UpdateKnowledgeColumns(ctx, knowledge.ID, updates); err != nil {
			logger.Errorf(ctx, "metadata extract: update knowledge failed: %v", err)
			return err
		}
		logger.Infof(ctx, "metadata extract: knowledge %s 更新 %d 高置信字段, %d 待确认",
			knowledge.ID, countChangedMetadata(&before, &after), len(pending))
	}

	// 抽取完成后同步 YLSW。生命周期/密级/部门是纯元数据，走原地更新避免重嵌；
	// 引擎不支持（文档未入库等）时回退整篇重推。
	if changed && s.ragEngine != nil {
		if err := s.ragEngine.UpdateDocumentMetadata(
			ctx, knowledge.KnowledgeBaseID, knowledge.ID, buildEngineDocumentMetadata(&after),
		); err != nil {
			logger.Warnf(ctx, "metadata extract: fast metadata update failed for %s, falling back: %v", knowledge.ID, err)
			if pushErr := s.pushUpdatedToEngine(ctx, payload.TenantID, payload.KnowledgeID, payload.KnowledgeBaseID); pushErr != nil {
				logger.Warnf(ctx, "metadata extract: re-push engine failed for %s: %v", knowledge.ID, pushErr)
			}
		}
	}
	return nil
}

// parseMetadataExtract extracts the JSON object from the LLM response.
func parseMetadataExtract(content string) (metadataExtractResult, bool) {
	content = strings.TrimSpace(content)
	m := extractJSONRe.FindString(content)
	if m == "" {
		return metadataExtractResult{}, false
	}
	var out metadataExtractResult
	if err := json.Unmarshal([]byte(m), &out); err != nil {
		return metadataExtractResult{}, false
	}
	return out, true
}

func getMetadataField(k *types.Knowledge, field string) string {
	switch field {
	case "lifecycle_status":
		return k.LifecycleStatus
	case "effective_date":
		return k.EffectiveDate
	case "expiry_date":
		return k.ExpiryDate
	case "superseded_by":
		return k.SupersededBy
	case "sensitivity":
		return k.Sensitivity
	case "department":
		return k.Department
	}
	return ""
}

func setMetadataField(k *types.Knowledge, field, value string) {
	switch field {
	case "lifecycle_status":
		k.LifecycleStatus = value
	case "effective_date":
		k.EffectiveDate = value
	case "expiry_date":
		k.ExpiryDate = value
	case "superseded_by":
		k.SupersededBy = value
	case "sensitivity":
		k.Sensitivity = value
	case "department":
		k.Department = value
	}
}

func countChangedMetadata(before, after *types.Knowledge) int {
	n := 0
	for _, f := range extractableMetadataFields {
		if getMetadataField(before, f) != getMetadataField(after, f) {
			n++
		}
	}
	return n
}

// applyEnterpriseMetadataFromMap reads §2.14 form fields from the upload
// metadata map (人工表单路) into the knowledge row. Unknown keys ignored;
// custom_metadata must be a JSON object string of the controlled KV.
func applyEnterpriseMetadataFromMap(k *types.Knowledge, metadata map[string]string) {
	if k == nil || metadata == nil {
		return
	}
	if v, ok := metadata["lifecycle_status"]; ok && v != "" {
		k.LifecycleStatus = v
	}
	if v, ok := metadata["effective_date"]; ok {
		k.EffectiveDate = v
	}
	if v, ok := metadata["expiry_date"]; ok {
		k.ExpiryDate = v
	}
	if v, ok := metadata["superseded_by"]; ok {
		k.SupersededBy = v
	}
	if v, ok := metadata["sensitivity"]; ok && v != "" {
		k.Sensitivity = v
	}
	if v, ok := metadata["department"]; ok {
		k.Department = v
	}
	if v, ok := metadata["custom_metadata"]; ok && v != "" {
		var cm map[string]any
		if json.Unmarshal([]byte(v), &cm) == nil {
			k.CustomMetadata = types.JSON(v)
		}
	}
}

// enqueueMetadataExtract schedules the LLM extraction after a successful
// parse+push. Best-effort: enqueue failures are logged only.
func (s *knowledgeService) enqueueMetadataExtract(
	ctx context.Context, knowledge *types.Knowledge, title, contentHead string,
) {
	if s.task == nil || knowledge == nil || knowledge.TenantID == 0 ||
		knowledge.ID == "" || knowledge.KnowledgeBaseID == "" {
		return
	}
	head := []rune(contentHead)
	if len(head) > 2000 {
		head = head[:2000]
	}
	payload := types.MetadataExtractPayload{
		TenantID:        knowledge.TenantID,
		KnowledgeID:     knowledge.ID,
		KnowledgeBaseID: knowledge.KnowledgeBaseID,
		Title:           title,
		ContentHead:     string(head),
	}
	langfuse.InjectTracing(ctx, &payload)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Warnf(ctx, "Failed to marshal metadata extract payload for %s: %v", knowledge.ID, err)
		return
	}
	task := asynq.NewTask(types.TypeMetadataExtract, payloadBytes,
		asynq.Queue(types.QueueSummary), asynq.MaxRetry(2), asynq.Timeout(2*time.Minute))
	if info, err := s.task.Enqueue(task); err != nil {
		logger.Warnf(ctx, "Failed to enqueue metadata extract for %s: %v", knowledge.ID, err)
	} else {
		logger.Infof(ctx, "Enqueued metadata extract: id=%s queue=%s knowledge_id=%s",
			info.ID, info.Queue, knowledge.ID)
	}
}
