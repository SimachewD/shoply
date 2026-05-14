package dto

import "github.com/sime/shoply/internal/models"

// admin
type ChangeRoleRequest struct {
	Role models.Role `json:"role" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type GetUsersQuery struct {
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`

	Search string `form:"search"`
	Role   string `form:"role"`
	Status string `form:"status"`
}


// user
type UpdateProfileRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

type DeleteUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type BanUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SuspendUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ActivateUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type RestoreUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}