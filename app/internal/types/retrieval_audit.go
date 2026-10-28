package types

import "time"

// RetrievalAuditLog is one best-effort retrieval ACL record. It is written
// asynchronously and never blocks the retrieval path.
type RetrievalAuditLog struct {
	ID            uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	TenantID      uint64    `json:"tenant_id" gorm:"not null;index"`
	UserID        string    `json:"user_id" gorm:"type:varchar(36);not null;default:''"`
	PrincipalKind string    `json:"principal_kind" gorm:"type:varchar(32);not null;default:'member'"`
	SessionID     string    `json:"session_id" gorm:"type:varchar(36);not null;default:''"`
	QueryDigest   string    `json:"query_digest" gorm:"type:varchar(128);not null;default:''"`
	KBIDs         string    `json:"kb_ids" gorm:"type:text;not null;default:''"`
	ScopeMode     string    `json:"scope_mode" gorm:"type:varchar(8);not null;default:''"`
	ScopeVersion  int64     `json:"scope_version" gorm:"not null;default:0"`
	ScopeDocCount int       `json:"scope_doc_count" gorm:"not null;default:0"`
	HitDocIDs     string    `json:"hit_doc_ids" gorm:"type:text;not null;default:''"`
	Decision      string    `json:"decision" gorm:"type:varchar(16);not null;default:''"`
	Error         string    `json:"error" gorm:"type:text;not null;default:''"`
	CreatedAt     time.Time `json:"created_at"`
}

func (RetrievalAuditLog) TableName() string { return "retrieval_audit_logs" }
