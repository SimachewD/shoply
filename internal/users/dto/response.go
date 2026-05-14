package dto

import (
	"time"

	"github.com/sime/shoply/internal/models"
)

type UserResponse struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Email    string `json:"email" db:"email"`
	Role     models.Role `json:"role" db:"role"`
	Status   models.Status `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type AuditLogResponse struct {
	ID             string               `json:"id"`
	UserID         string               `json:"user_id"`
	Action         string               `json:"action"`
	ActorID        string               `json:"actor_id,omitempty"`
	ActorName      string               `json:"actor_name,omitempty"`

	Resource       string               `json:"resource"`
	Metadata       map[string]any     `json:"metadata"`
	IPAddress      string               `json:"ip_address,omitempty"`
	UserAgent      string               `json:"user_agent,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
}
