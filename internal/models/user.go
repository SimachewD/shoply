// internal/models/user.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
	RoleAdmin  Role = "admin"
	RoleSuperAdmin Role = "super_admin"
	RoleModerator  Role = "moderator"
	RoleSupport  Role = "support"
	RoleEditor Role = "editor"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusPendingVerification  Status = "pending_verification"
	StatusBanned   Status = "banned"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)


type User struct {
    ID           uuid.UUID `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"`
    Role         Role      `json:"role" db:"role"`
    Status       Status    `json:"status" db:"status"`
	SuspendedUntil  *time.Time `json:"suspended_until"`
	BannedAt        *time.Time `json:"banned_at"`

    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`

	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
	DeletedBy *uuid.UUID `json:"deleted_by" db:"deleted_by"`
}

type SellerRequest struct {
	ID        string    `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}