package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID `json:"id"`
	AdminID      uuid.UUID `json:"admin_id"`
	Action       string    `json:"action"`
	TargetUserID uuid.UUID `json:"target_user_id"`
	Metadata     []byte    `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}