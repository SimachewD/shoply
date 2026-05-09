package dto

import "github.com/sime/shoply/internal/models"

type UserResponse struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Email    string `json:"email" db:"email"`
	Role     models.Role `json:"role" db:"role"`
	Status   models.Status `json:"status" db:"status"`
}