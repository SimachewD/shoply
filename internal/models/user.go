// internal/models/user.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
    RoleAdmin  Role = "admin"
    RoleSeller Role = "seller"
    RoleBuyer  Role = "buyer"
)

type User struct {
    ID           uuid.UUID `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"`
    Role         Role      `json:"role" db:"role"`

    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}