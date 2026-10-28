package types

import "time"

// GovernanceDepartmentStat is one department row in the governance dashboard.
type GovernanceDepartmentStat struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	KBTotal        int64  `json:"kb_total"`
	DocumentTotal  int64  `json:"document_total"`
	MemberTotal    int64  `json:"member_total"`
}

// GovernanceRecentDocument is metadata-only;正文仍走 KB/ACL 守卫.
type GovernanceRecentDocument struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	FileName        string    `json:"file_name"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	KnowledgeBase   string    `json:"knowledge_base"`
	ParseStatus     string    `json:"parse_status"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GovernanceStats is the read-only aggregate returned to a governance page.
type GovernanceStats struct {
	Scope                         string                     `json:"scope"`
	DepartmentID                  string                     `json:"department_id,omitempty"`
	DepartmentName                string                     `json:"department_name,omitempty"`
	KBTotal                       int64                      `json:"kb_total"`
	DocumentTotal                 int64                      `json:"document_total"`
	ProcessingTotal               int64                      `json:"processing_total"`
	FailedTotal                   int64                      `json:"failed_total"`
	MemberTotal                   int64                      `json:"member_total"`
	SessionTotal                  int64                      `json:"session_total"`
	MessageTotal                  int64                      `json:"message_total"`
	ManagerTotal                  int64                      `json:"manager_total"`
	KnowledgeAdminTotal           int64                      `json:"knowledge_admin_total"`
	EditorTotal                   int64                      `json:"editor_total"`
	ViewerTotal                   int64                      `json:"viewer_total"`
	SharedInTotal                 int64                      `json:"shared_in_total"`
	SharedOutTotal                int64                      `json:"shared_out_total"`
	Departments                   []GovernanceDepartmentStat `json:"departments,omitempty"`
	RecentDocuments               []GovernanceRecentDocument `json:"recent_documents"`
	GeneratedAt                   time.Time                  `json:"generated_at"`
	PeriodDays                    int                        `json:"period_days"`
	SearchTotal                   int64                      `json:"search_total"`
	NoHitTotal                    int64                      `json:"no_hit_total"`
	SearchHitRate                 float64                    `json:"search_hit_rate"`
	PendingCRTotal                int64                      `json:"pending_cr_total"`
	OpenIssueTotal                int64                      `json:"open_issue_total"`
	CRAvgCycleDays                float64                    `json:"cr_avg_cycle_days"`
	ReadingRunTotal               int64                      `json:"reading_run_total"`
	ReadingFinishedTotal          int64                      `json:"reading_finished_total"`
	ReadingStoppedTotal           int64                      `json:"reading_stopped_total"`
	ReadingFailedTotal            int64                      `json:"reading_failed_total"`
	ReadingQuestionTotal          int64                      `json:"reading_question_total"`
	ReadingAutoAnsweredTotal      int64                      `json:"reading_auto_answered_total"`
	ReadingAutoAnswerRate         float64                    `json:"reading_auto_answer_rate"`
	ReadingConclusionTotal        int64                      `json:"reading_conclusion_total"`
	ReadingConfirmedTotal         int64                      `json:"reading_confirmed_total"`
	ReadingLegacyConclusionTotal  int64                      `json:"reading_legacy_conclusion_total"`
	ReadingTermTotal              int64                      `json:"reading_term_total"`
	ReadingContradictionTotal     int64                      `json:"reading_contradiction_total"`
	ReadingOpenContradictionTotal int64                      `json:"reading_open_contradiction_total"`
	ReadingAvgSteps               float64                    `json:"reading_avg_steps"`
	ReadingAvgTokens              float64                    `json:"reading_avg_tokens"`
	ReadingAvgDurationMS          float64                    `json:"reading_avg_duration_ms"`
	SharedExternalSearches        int64                      `json:"shared_external_searches"`
	KBHealth                      []GovernanceKBHealth       `json:"kb_health,omitempty"`
}

// GovernanceAuditEntry is the UI projection of one audit row.
type GovernanceAuditEntry struct {
	ID          uint64    `json:"id"`
	ActorUserID string    `json:"actor_user_id"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Outcome     string    `json:"outcome"`
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// GovernanceKBHealth is one department KB row. Search counts cover the
// reporting window; external searches count retrievals by users outside the
// KB's owning department.
type GovernanceKBHealth struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	DocumentTotal       int64     `json:"document_total"`
	UpdatedAt           time.Time `json:"updated_at"`
	SearchTotal         int64     `json:"search_total"`
	ExternalSearchTotal int64     `json:"external_search_total"`
	SharedOutTotal      int64     `json:"shared_out_total"`
}
