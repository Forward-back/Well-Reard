package repository

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

type governanceRepository struct{ db *gorm.DB }

func NewGovernanceRepository(db *gorm.DB) interfaces.GovernanceRepository {
	return &governanceRepository{db: db}
}

func (r *governanceRepository) kbScope(tenantID uint64, departmentID string) *gorm.DB {
	query := r.db.Model(&types.KnowledgeBase{}).
		Where("tenant_id = ? AND is_temporary = ? AND deleted_at IS NULL", tenantID, false)
	if departmentID != "" {
		query = query.Where("department_id = ?", departmentID)
	}
	return query
}

func (r *governanceRepository) Stats(
	ctx context.Context, tenantID uint64, departmentID string, company bool,
) (*types.GovernanceStats, error) {
	db := r.db.WithContext(ctx)
	stats := &types.GovernanceStats{
		Scope:           "dept",
		DepartmentID:    departmentID,
		GeneratedAt:     time.Now(),
		RecentDocuments: make([]types.GovernanceRecentDocument, 0),
		Departments:     make([]types.GovernanceDepartmentStat, 0),
	}
	if company {
		stats.Scope = "company"
	}
	if err := r.kbScope(tenantID, departmentID).Count(&stats.KBTotal).Error; err != nil {
		return nil, err
	}
	docDocs := func() *gorm.DB {
		return db.Model(&types.Knowledge{}).
			Where("tenant_id = ? AND deleted_at IS NULL AND knowledge_base_id IN (?)",
				tenantID, r.kbScope(tenantID, departmentID).Select("id"))
	}
	if err := docDocs().Count(&stats.DocumentTotal).Error; err != nil {
		return nil, err
	}
	if err := docDocs().
		Where("parse_status NOT IN ?", []string{
			types.ParseStatusCompleted, types.ParseStatusFailed,
			types.ParseStatusDeleting, types.ParseStatusCancelled,
		}).Count(&stats.ProcessingTotal).Error; err != nil {
		return nil, err
	}
	if err := docDocs().
		Where("parse_status = ?", types.ParseStatusFailed).
		Count(&stats.FailedTotal).Error; err != nil {
		return nil, err
	}

	memberBase := func() *gorm.DB {
		query := db.Model(&types.TenantMember{}).
			Where("tenant_id = ? AND status = ? AND deleted_at IS NULL",
				tenantID, types.TenantMemberStatusActive)
		if departmentID != "" {
			query = query.Where("department_id = ?", departmentID)
		}
		return query
	}
	if err := memberBase().Count(&stats.MemberTotal).Error; err != nil {
		return nil, err
	}
	roleCounts := func(role types.DepartmentRole) int64 {
		var count int64
		_ = memberBase().Where("department_role = ?", role).Count(&count).Error
		return count
	}
	stats.ManagerTotal = roleCounts(types.DepartmentRoleManager)
	stats.KnowledgeAdminTotal = roleCounts(types.DepartmentRoleKnowledgeAdmin)
	stats.EditorTotal = roleCounts(types.DepartmentRoleEditor)
	stats.ViewerTotal = roleCounts(types.DepartmentRoleViewer)

	sessionQuery := db.Model(&types.Session{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	messageQuery := db.Table("messages m").
		Joins("JOIN sessions s ON s.id = m.session_id AND s.deleted_at IS NULL").
		Where("s.tenant_id = ? AND m.deleted_at IS NULL", tenantID)
	if departmentID != "" {
		memberIDs := db.Model(&types.TenantMember{}).
			Select("user_id").
			Where("tenant_id = ? AND department_id = ? AND status = ? AND deleted_at IS NULL",
				tenantID, departmentID, types.TenantMemberStatusActive)
		sessionQuery = sessionQuery.Where("user_id IN (?)", memberIDs)
		messageQuery = messageQuery.Where("s.user_id IN (?)", memberIDs)
	}
	if err := sessionQuery.Count(&stats.SessionTotal).Error; err != nil {
		return nil, err
	}
	if err := messageQuery.Count(&stats.MessageTotal).Error; err != nil {
		return nil, err
	}

	readingBase := func() *gorm.DB {
		query := db.Model(&types.ReadingRun{}).Where("tenant_id = ?", tenantID)
		if departmentID != "" {
			query = query.Where("user_id IN (?)", db.Model(&types.TenantMember{}).
				Select("user_id").
				Where("tenant_id = ? AND department_id = ? AND status = ? AND deleted_at IS NULL",
					tenantID, departmentID, types.TenantMemberStatusActive))
		}
		return query
	}
	if err := readingBase().Count(&stats.ReadingRunTotal).Error; err != nil {
		return nil, err
	}
	if err := readingBase().Where("status = ?", types.ReadingRunFinished).Count(&stats.ReadingFinishedTotal).Error; err != nil {
		return nil, err
	}
	if err := readingBase().Where("status = ?", types.ReadingRunStopped).Count(&stats.ReadingStoppedTotal).Error; err != nil {
		return nil, err
	}
	if err := readingBase().Where("status = ?", types.ReadingRunFailed).Count(&stats.ReadingFailedTotal).Error; err != nil {
		return nil, err
	}

	readingRunIDs := func() *gorm.DB { return readingBase().Select("id") }
	if err := db.Model(&types.ReadingQuestionNode{}).
		Where("run_id IN (?)", readingRunIDs()).
		Count(&stats.ReadingQuestionTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingQuestionNode{}).
		Where("run_id IN (?) AND status = ?", readingRunIDs(), types.ReadingQuestionAutoAnswered).
		Count(&stats.ReadingAutoAnsweredTotal).Error; err != nil {
		return nil, err
	}
	if stats.ReadingQuestionTotal > 0 {
		stats.ReadingAutoAnswerRate = float64(stats.ReadingAutoAnsweredTotal) /
			float64(stats.ReadingQuestionTotal) * 100
	}
	if err := db.Model(&types.ReadingConclusion{}).
		Where("run_id IN (?)", readingRunIDs()).
		Count(&stats.ReadingConclusionTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingConclusion{}).
		Where("run_id IN (?) AND status = ? AND source <> ?",
			readingRunIDs(), types.ReadingConclusionConfirmed, types.ReadingConclusionSourceLegacy).
		Count(&stats.ReadingConfirmedTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingConclusion{}).
		Where("run_id IN (?) AND source = ?", readingRunIDs(), types.ReadingConclusionSourceLegacy).
		Count(&stats.ReadingLegacyConclusionTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingTerm{}).
		Where("run_id IN (?)", readingRunIDs()).
		Count(&stats.ReadingTermTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingContradiction{}).
		Where("run_id IN (?)", readingRunIDs()).
		Count(&stats.ReadingContradictionTotal).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&types.ReadingContradiction{}).
		Where("run_id IN (?) AND status = ?", readingRunIDs(), types.ReadingContradictionOpen).
		Count(&stats.ReadingOpenContradictionTotal).Error; err != nil {
		return nil, err
	}
	var readingAverages struct {
		Steps  float64
		Tokens float64
	}
	if err := readingBase().
		Select("COALESCE(AVG(step_count), 0) AS steps, COALESCE(AVG(token_usage), 0) AS tokens").
		Scan(&readingAverages).Error; err != nil {
		return nil, err
	}
	stats.ReadingAvgSteps = readingAverages.Steps
	stats.ReadingAvgTokens = readingAverages.Tokens
	// Duration lives in the step log; PG and SQLite use different JSON access.
	durationExpr := "(s.notes_patch->>'duration_ms')::numeric"
	if db.Dialector.Name() == "sqlite" {
		durationExpr = "CAST(json_extract(s.notes_patch, '$.duration_ms') AS REAL)"
	}
	stepQuery := db.Table("reading_run_steps s").
		Joins("JOIN reading_runs r ON r.id = s.run_id").
		Where("r.tenant_id = ?", tenantID)
	if departmentID != "" {
		stepQuery = stepQuery.Where("r.user_id IN (?)", db.Model(&types.TenantMember{}).
			Select("user_id").
			Where("tenant_id = ? AND department_id = ? AND status = ? AND deleted_at IS NULL",
				tenantID, departmentID, types.TenantMemberStatusActive))
	}
	var durationAverage struct{ Value float64 }
	_ = stepQuery.
		Select("COALESCE(AVG(" + durationExpr + "), 0) AS value").
		Scan(&durationAverage).Error
	stats.ReadingAvgDurationMS = durationAverage.Value

	shareOut := db.Model(&types.KBDepartmentShare{}).
		Where("tenant_id = ? AND kb_id IN (?)", tenantID, r.kbScope(tenantID, departmentID).Select("id"))
	if err := shareOut.Distinct("kb_id").Count(&stats.SharedOutTotal).Error; err != nil {
		return nil, err
	}
	if departmentID != "" {
		shareIn := db.Table("kb_department_shares s").
			Joins("JOIN knowledge_bases kb ON kb.id = s.kb_id AND kb.deleted_at IS NULL").
			Where("s.tenant_id = ? AND s.department_id = ? AND kb.department_id <> ?",
				tenantID, departmentID, departmentID)
		if err := shareIn.Distinct("s.kb_id").Count(&stats.SharedInTotal).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Table("kb_department_shares s").
			Joins("JOIN knowledge_bases kb ON kb.id = s.kb_id AND kb.deleted_at IS NULL").
			Where("s.tenant_id = ? AND kb.department_id <> s.department_id", tenantID).
			Distinct("s.kb_id").Count(&stats.SharedInTotal).Error; err != nil {
			return nil, err
		}
	}

	if departmentID != "" {
		var dept types.Department
		if err := db.Where("tenant_id = ? AND id = ?", tenantID, departmentID).First(&dept).Error; err == nil {
			stats.DepartmentName = dept.Name
		}
	}

	// Operational insights for the department manager dashboard. The
	// retrieval audit stores query hashes, never raw questions, so these
	// aggregates intentionally avoid claim-level "knowledge gap" text.
	periodDays := 7
	since := time.Now().AddDate(0, 0, -periodDays)
	stats.PeriodDays = periodDays

	type memberDeptRow struct {
		UserID       string
		DepartmentID *string
	}
	var memberDepts []memberDeptRow
	_ = db.Model(&types.TenantMember{}).
		Select("user_id, department_id").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, types.TenantMemberStatusActive).
		Scan(&memberDepts).Error
	deptByUser := make(map[string]string, len(memberDepts))
	deptMembers := make(map[string]struct{})
	for _, row := range memberDepts {
		if row.DepartmentID != nil {
			deptByUser[row.UserID] = *row.DepartmentID
		}
		if departmentID != "" && row.DepartmentID != nil && *row.DepartmentID == departmentID {
			deptMembers[row.UserID] = struct{}{}
		}
	}

	type kbRow struct {
		ID           string
		Name         string
		DepartmentID string
		UpdatedAt    time.Time
	}
	var kbs []kbRow
	kbQuery := db.Model(&types.KnowledgeBase{}).
		Select("id, name, department_id, updated_at").
		Where("tenant_id = ? AND is_temporary = ? AND deleted_at IS NULL", tenantID, false)
	if departmentID != "" {
		kbQuery = kbQuery.Where("department_id = ?", departmentID)
	}
	_ = kbQuery.Find(&kbs).Error
	kbIDs := make([]string, 0, len(kbs))
	kbOwnerDept := make(map[string]string, len(kbs))
	for _, kb := range kbs {
		kbIDs = append(kbIDs, kb.ID)
		kbOwnerDept[kb.ID] = kb.DepartmentID
	}
	docCounts := make(map[string]int64)
	if len(kbIDs) > 0 {
		type countRow struct {
			KnowledgeBaseID string
			Total           int64
		}
		var rows []countRow
		_ = db.Table("knowledges").
			Select("knowledge_base_id, COUNT(*) AS total").
			Where("tenant_id = ? AND deleted_at IS NULL AND knowledge_base_id IN ?", tenantID, kbIDs).
			Group("knowledge_base_id").Scan(&rows).Error
		for _, row := range rows {
			docCounts[row.KnowledgeBaseID] = row.Total
		}
	}
	shareCounts := make(map[string]int64)
	if len(kbIDs) > 0 {
		type countRow struct {
			KBID  string
			Total int64
		}
		var rows []countRow
		_ = db.Table("kb_department_shares").
			Select("kb_id, COUNT(*) AS total").
			Where("tenant_id = ? AND kb_id IN ?", tenantID, kbIDs).
			Group("kb_id").Scan(&rows).Error
		for _, row := range rows {
			shareCounts[row.KBID] = row.Total
		}
	}
	if len(kbIDs) > 0 {
		_ = db.Model(&types.DocChangeRequest{}).
			Where("tenant_id = ? AND knowledge_base_id IN ? AND status = ?", tenantID, kbIDs, types.DocCRStatusPendingReview).
			Count(&stats.PendingCRTotal).Error
		var merged []struct {
			CreatedAt time.Time
			MergedAt  *time.Time
		}
		_ = db.Model(&types.DocChangeRequest{}).
			Select("created_at, merged_at").
			Where("tenant_id = ? AND knowledge_base_id IN ? AND status = ? AND merged_at IS NOT NULL AND merged_at >= ?",
				tenantID, kbIDs, types.DocCRStatusMerged, since).
			Scan(&merged).Error
		var totalDays float64
		var mergedCount int64
		for _, row := range merged {
			if row.MergedAt != nil {
				totalDays += row.MergedAt.Sub(row.CreatedAt).Hours() / 24
				mergedCount++
			}
		}
		if mergedCount > 0 {
			stats.CRAvgCycleDays = totalDays / float64(mergedCount)
		}
		_ = db.Model(&types.DocIssue{}).
			Where("tenant_id = ? AND knowledge_base_id IN ? AND status = ?", tenantID, kbIDs, types.DocIssueStatusOpen).
			Count(&stats.OpenIssueTotal).Error
	}

	type retrievalRow struct {
		UserID    string
		KBIDs     string
		HitDocIDs string
		Decision  string
	}
	var retrievalLogs []retrievalRow
	_ = db.Table("retrieval_audit_logs").
		Select("user_id, kb_ids, hit_doc_ids, decision").
		Where("tenant_id = ? AND created_at >= ?", tenantID, since).
		Scan(&retrievalLogs).Error

	kbSearch := make(map[string]int64, len(kbs))
	kbExternal := make(map[string]int64, len(kbs))
	for _, logRow := range retrievalLogs {
		inDept := departmentID == ""
		if departmentID != "" {
			_, inDept = deptMembers[logRow.UserID]
		}
		kbList := parseJSONStringList(logRow.KBIDs)
		hitList := parseJSONStringList(logRow.HitDocIDs)
		if inDept {
			stats.SearchTotal++
			if logRow.Decision == "ok" && len(hitList) == 0 {
				stats.NoHitTotal++
			}
			for _, kbID := range kbList {
				kbSearch[kbID]++
			}
		}
		if departmentID == "" {
			continue
		}
		userDept := deptByUser[logRow.UserID]
		if userDept == "" || userDept == departmentID {
			continue
		}
		for _, kbID := range kbList {
			if kbOwnerDept[kbID] == departmentID {
				kbExternal[kbID]++
				stats.SharedExternalSearches++
			}
		}
	}
	if stats.SearchTotal > 0 {
		stats.SearchHitRate = float64(stats.SearchTotal-stats.NoHitTotal) / float64(stats.SearchTotal) * 100
	}
	if departmentID != "" {
		stats.KBHealth = make([]types.GovernanceKBHealth, 0, len(kbs))
		for _, kb := range kbs {
			stats.KBHealth = append(stats.KBHealth, types.GovernanceKBHealth{
				ID: kb.ID, Name: kb.Name, DocumentTotal: docCounts[kb.ID], UpdatedAt: kb.UpdatedAt,
				SearchTotal: kbSearch[kb.ID], ExternalSearchTotal: kbExternal[kb.ID],
				SharedOutTotal: shareCounts[kb.ID],
			})
		}
	}

	recent := make([]struct {
		ID              string
		Title           string
		FileName        string
		KnowledgeBaseID string
		KnowledgeBase   string
		ParseStatus     string
		UpdatedAt       time.Time
	}, 0)
	recentQuery := db.Table("knowledges k").
		Select("k.id, k.title, k.file_name, k.knowledge_base_id, kb.name AS knowledge_base, k.parse_status, k.updated_at").
		Joins("JOIN knowledge_bases kb ON kb.id = k.knowledge_base_id").
		Where("k.tenant_id = ? AND k.deleted_at IS NULL AND kb.deleted_at IS NULL AND kb.is_temporary = ?",
			tenantID, false)
	if departmentID != "" {
		recentQuery = recentQuery.Where("kb.department_id = ?", departmentID)
	}
	if err := recentQuery.Order("k.updated_at DESC").Limit(8).Scan(&recent).Error; err != nil {
		return nil, err
	}
	for _, row := range recent {
		stats.RecentDocuments = append(stats.RecentDocuments, types.GovernanceRecentDocument{
			ID: row.ID, Title: row.Title, FileName: row.FileName,
			KnowledgeBaseID: row.KnowledgeBaseID, KnowledgeBase: row.KnowledgeBase,
			ParseStatus: row.ParseStatus, UpdatedAt: row.UpdatedAt,
		})
	}

	if company {
		var depts []*types.Department
		if err := db.Where("tenant_id = ?", tenantID).Order("is_default DESC, name ASC").
			Find(&depts).Error; err != nil {
			return nil, err
		}
		for _, dept := range depts {
			if dept == nil {
				continue
			}
			row := types.GovernanceDepartmentStat{DepartmentID: dept.ID, DepartmentName: dept.Name}
			_ = r.kbScope(tenantID, dept.ID).Count(&row.KBTotal).Error
			_ = db.Model(&types.Knowledge{}).
				Where("tenant_id = ? AND deleted_at IS NULL AND knowledge_base_id IN (?)",
					tenantID, r.kbScope(tenantID, dept.ID).Select("id")).
				Count(&row.DocumentTotal).Error
			_ = db.Model(&types.TenantMember{}).
				Where("tenant_id = ? AND status = ? AND department_id = ? AND deleted_at IS NULL",
					tenantID, types.TenantMemberStatusActive, dept.ID).
				Count(&row.MemberTotal).Error
			stats.Departments = append(stats.Departments, row)
		}
	}
	return stats, nil
}

func (r *governanceRepository) Audit(
	ctx context.Context, tenantID uint64, departmentID string, afterID uint64, limit int,
) ([]*types.GovernanceAuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	db := r.db.WithContext(ctx).Table("audit_logs a").
		Select("a.id, a.actor_user_id, a.action, a.target_type, a.target_id, a.outcome, a.scope_type, a.scope_id, a.created_at").
		Where("a.tenant_id = ?", tenantID)
	if afterID > 0 {
		db = db.Where("a.id < ?", afterID)
	}
	if departmentID != "" {
		db = db.Joins(
			"LEFT JOIN tenant_members tm ON tm.user_id = a.actor_user_id AND tm.tenant_id = a.tenant_id",
		).Where(
			"(tm.department_id = ? AND tm.deleted_at IS NULL) OR "+
				"(a.scope_type = 'knowledge_base' AND a.scope_id IN "+
				"(SELECT id FROM knowledge_bases WHERE tenant_id = ? AND department_id = ? AND deleted_at IS NULL))",
			departmentID, tenantID, departmentID,
		)
	}
	var rows []*types.GovernanceAuditEntry
	if err := db.Order("a.id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = make([]*types.GovernanceAuditEntry, 0)
	}
	return rows, nil
}

func parseJSONStringList(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}
