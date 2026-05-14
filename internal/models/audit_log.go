package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	ActorID      uuid.UUID       `json:"actor_id" db:"actor_id"`
	ActorName    string          `json:"actor_name" db:"actor_name"`
	Action       string          `json:"action" db:"action"`
	Resource     string          `json:"resource" db:"resource"`
	Metadata     json.RawMessage `json:"metadata" db:"metadata"`

	IPAddress    string          `json:"ip_address" db:"ip_address"`
	UserAgent    string         `json:"user_agent" db:"user_agent"`

	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

type AuditLogMetadata struct {
	// ─── General audit fields ──────────────────────────────────────
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`

	// Optional: which resource was affected?
	ResourceID string `json:"resource_id,omitempty"`
	ResourceName string `json:"resource_name,omitempty"` // useful for users, products, etc.

	// ─── User-specific audit fields ─────────────────────────────────
	TargetUserID   string           `json:"target_user_id,omitempty"`
	TargetUserName string           `json:"target_user_name,omitempty"`
	TargetUserRole string           `json:"target_user_role,omitempty"`
	TargetUserEmail string          `json:"target_user_email,omitempty"`

	// Changes in fields (before vs after)
	FieldsChanged map[string]FieldChange `json:"fields_changed,omitempty"`

	// For “before and after” field comparison
	OldValues map[string]any `json:"old_values,omitempty"`
	NewValues map[string]any `json:"new_values,omitempty"`

	// ─── Reasons and optional messages ──────────────────────────────
	Reason string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`

	// IP and User-Agent are stored in the audit_logs table as separate columns
	// But you can keep them here if you want easy access in metadata
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`

	// Optional – admin who performed the action (besides actor)
	PerformedBy string `json:"performed_by,omitempty"`

	// Optional – extra structured data
	Extra map[string]any `json:"extra,omitempty"`
}

type FieldChange struct {
	Old string `json:"old,omitempty"`
	New string `json:"new,omitempty"`
}