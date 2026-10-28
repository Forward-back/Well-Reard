package types

import (
	"time"
)

// PortalSession is a consumer-portal conversation (§2.13 P3), physically
// separated from internal user sessions. token_kind records which portal
// token flow created it; identity_key holds a de-identified consumer key
// (phone hash / anonymous device id — never the raw identifier).
type PortalSession struct {
	ID            string      `json:"id" gorm:"type:varchar(36);primaryKey"`
	TokenKind     string      `json:"token_kind" gorm:"type:varchar(16);not null;default:'anonymous'"`
	IdentityKey   string      `json:"identity_key" gorm:"type:varchar(128);index"`
	KBIDs         StringArray `json:"kb_ids" gorm:"type:jsonb;not null;default:'[]'"`
	FirstQuestion string      `json:"first_question" gorm:"type:text"`
	MessageCount  int         `json:"message_count" gorm:"not null;default:0"`
	LastMessageAt *time.Time  `json:"last_message_at"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func (PortalSession) TableName() string { return "portal_sessions" }

// PortalMessage is one turn inside a portal session. Refs are sanitised
// citations (title only — no file names, doc numbers or org metadata).
type PortalMessage struct {
	ID        int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID string      `json:"session_id" gorm:"type:varchar(36);not null;index"`
	Role      string      `json:"role" gorm:"type:varchar(16);not null"`
	Content   string      `json:"content" gorm:"type:text;not null"`
	Refs      StringArray `json:"refs" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt time.Time   `json:"created_at"`
}

func (PortalMessage) TableName() string { return "portal_messages" }
