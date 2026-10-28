package types

import "time"

type KBAclResourceType string

const (
	KBAclResourceFolder KBAclResourceType = "folder"
	KBAclResourceFile   KBAclResourceType = "file"
)

type KBAclEffect string

const (
	KBAclEffectAllow KBAclEffect = "allow"
	KBAclEffectDeny  KBAclEffect = "deny"
)

type KBAclSubjectType string

const (
	KBAclSubjectUser       KBAclSubjectType = "user"
	KBAclSubjectDepartment KBAclSubjectType = "department"
	KBAclSubjectAll        KBAclSubjectType = "all"
)

// KBAclRule is one in-knowledge-base ACL entry. File rules are deny-only;
// folder rules may be allow or deny.
type KBAclRule struct {
	ID              string            `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64            `json:"tenant_id" gorm:"not null;index"`
	KnowledgeBaseID string            `json:"knowledge_base_id" gorm:"column:kb_id;type:varchar(36);not null;index"`
	ResourceType    KBAclResourceType `json:"resource_type" gorm:"type:varchar(16);not null"`
	ResourceID      string            `json:"resource_id" gorm:"type:varchar(64);not null"`
	Effect          KBAclEffect       `json:"effect" gorm:"type:varchar(8);not null"`
	SubjectType     KBAclSubjectType  `json:"subject_type" gorm:"type:varchar(16);not null"`
	SubjectID       string            `json:"subject_id" gorm:"type:varchar(64);not null;default:''"`
	CreatedBy       string            `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt       time.Time         `json:"created_at"`
}

func (KBAclRule) TableName() string { return "kb_acl_rules" }

func (e KBAclEffect) Valid() bool {
	return e == KBAclEffectAllow || e == KBAclEffectDeny
}

func (s KBAclSubjectType) Valid() bool {
	return s == KBAclSubjectUser || s == KBAclSubjectDepartment || s == KBAclSubjectAll
}

type TenantACLEpoch struct {
	TenantID  uint64    `json:"tenant_id" gorm:"primaryKey"`
	Epoch     int64     `json:"epoch" gorm:"not null;default:0"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TenantACLEpoch) TableName() string { return "tenant_acl_epochs" }

// KBAclRuleRequest is the create/update body for an ACL rule.
type KBAclRuleRequest struct {
	ResourceType KBAclResourceType `json:"resource_type" binding:"required"`
	ResourceID   string            `json:"resource_id" binding:"required"`
	Effect       KBAclEffect       `json:"effect" binding:"required"`
	SubjectType  KBAclSubjectType  `json:"subject_type" binding:"required"`
	SubjectID    string            `json:"subject_id"`
}

// KBAclScope is the effective read scope for one caller/KB. Active=false means
// no ACL rules apply and the caller keeps the normal department/share scope.
type KBAclScope struct {
	Active               bool             `json:"active"`
	AllowedKnowledgeIDs  []string         `json:"allowed_knowledge_ids,omitempty"`
	AllowedFolderIDs     []string         `json:"allowed_folder_ids,omitempty"`
	RootFolderID         string           `json:"root_folder_id,omitempty"`
	FolderDocumentCounts map[string]int64 `json:"folder_document_counts,omitempty"`
}

// ACLScopeMode controls how the engine applies a retrieval scope.
type ACLScopeMode string

const (
	ACLScopeAll   ACLScopeMode = "all"
	ACLScopeAllow ACLScopeMode = "allow"
	ACLScopeDeny  ACLScopeMode = "deny"
)

// ACLScope is the engine-facing retrieval scope. The app computes it from
// department shares + in-KB ACL; the engine only mechanically applies it.
type ACLScope struct {
	Mode    ACLScopeMode `json:"mode"`
	DocIDs  []string     `json:"document_ids,omitempty"`
	Version int64        `json:"scope_version"`
}

// Valid reports whether the scope is safe to apply. Unknown modes and allow/
// deny scopes without document IDs are invalid.
func (s *ACLScope) Valid() bool {
	if s == nil {
		return false
	}
	switch s.Mode {
	case ACLScopeAll:
		return true
	case ACLScopeAllow, ACLScopeDeny:
		return len(s.DocIDs) > 0 || s.Mode == ACLScopeAllow
	default:
		return false
	}
}
