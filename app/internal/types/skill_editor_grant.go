package types

import "time"

// SkillGrantSubjectType identifies the principal a skill editor grant applies
// to. Grants are resource-level and never alter global roles.
type SkillGrantSubjectType string

const (
	SkillGrantSubjectUser       SkillGrantSubjectType = "user"
	SkillGrantSubjectDepartment SkillGrantSubjectType = "department"
)

// SkillEditorGrant grants one user or department permission to edit one
// tenant-level skill definition.
type SkillEditorGrant struct {
	ID          string                `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64                `json:"tenant_id" gorm:"not null;index"`
	SkillID     string                `json:"skill_id" gorm:"column:skill_id;type:varchar(36);not null;index"`
	SubjectType SkillGrantSubjectType `json:"subject_type" gorm:"type:varchar(16);not null"`
	SubjectID   string                `json:"subject_id" gorm:"type:varchar(64);not null"`
	CreatedBy   string                `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt   time.Time             `json:"created_at"`
}

func (SkillEditorGrant) TableName() string { return "skill_editor_grants" }

// SkillEditorGrantRequest is the create body for a grant.
type SkillEditorGrantRequest struct {
	SubjectType SkillGrantSubjectType `json:"subject_type" binding:"required"`
	SubjectID   string                `json:"subject_id" binding:"required"`
}
