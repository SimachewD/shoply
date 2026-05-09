package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
	"github.com/sime/shoply/internal/response"
	"github.com/sime/shoply/internal/users/dto"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	user, err := h.service.Register(req)
	mappedUser := dto.UserResponse{
		ID:       user.ID.String(),
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "User created successfully", mappedUser, nil)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	access, refresh, err := h.service.Login(req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	// 🍪 set refresh token cookie
	c.SetCookie(
		"refresh_token",
		refresh,
		int(auth.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true, // Secure
		true, // HttpOnly
	)

	response.Success(c, http.StatusOK, "User logged in successfully", gin.H{
		"access_token": access,
	}, nil)
}

func (h *Handler) Refresh(c *gin.Context) {
	token, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing refresh token")
		return
	}

	access, newRefresh, err := h.service.Refresh(token)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	// rotate cookie
	c.SetCookie(
		"refresh_token",
		newRefresh,
		int(auth.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true,
		true,
	)

	response.Success(c, http.StatusOK, "User refreshed successfully", gin.H{
		"access_token": access,
	}, nil)
}

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie("refresh_token")

	err := h.service.Logout(token)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
        return
    }

	// clear cookie
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	response.Success(c, http.StatusOK, "User logged out successfully", nil, nil)
}

// admin routes
func (h *Handler) GetSellerRequests(c *gin.Context) {
	reqs, err := h.service.GetSellerRequests()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pending requests retrieved successfully", reqs, nil)
}

func (h *Handler) GetUsers(c *gin.Context) {

	cursor := c.Query("cursor")
	limitStr := c.Query("limit")

	search := c.Query("search")
	role := models.Role(c.Query("role"))
	status := models.Status(c.Query("status"))
	sortedBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")

	users, total, hasMore, nextCursor, err := h.service.GetUsers(cursor, limitStr, search, role, status, sortedBy, sortOrder)

	mappedUsers := make([]dto.UserResponse, len(users))
	meta := gin.H{
		"total": total,
		"has_more": hasMore,
		"next_cursor": nextCursor,
	}

	for i, u := range users {
		mappedUsers[i] = dto.UserResponse{
			ID:       u.ID.String(),
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			Status: u.Status,
		}
	}

	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "users fetched successfully", mappedUsers, meta)
}

func (h *Handler) GetDeletedUsers(c *gin.Context) {

	cursor := c.Query("cursor")
	limitStr := c.Query("limit")

	search := c.Query("search")
	sortedBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")

	users, total, hasMore, nextCursor, err := h.service.GetDeletedUsers(cursor, limitStr, search, sortedBy, sortOrder)

	mappedUsers := make([]dto.UserResponse, len(users))
	meta := gin.H{
		"total": total,
		"has_more": hasMore,
		"next_cursor": nextCursor,
	}

	for i, u := range users {
		mappedUsers[i] = dto.UserResponse{
			ID:       u.ID.String(),
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			Status: u.Status,
		}
	}
	
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Deleted users retrieved successfully", mappedUsers, meta)
}

func (h *Handler) ChangeRole(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))

	var req dto.ChangeRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.ChangeRole(adminID, id, req.Role, req.Reason)

	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User role updated successfully", gin.H{
		"id":       id.String(),
		"role":     req.Role,
		"admin_id": adminID.String(),
	}, nil)
}

func (h *Handler) SuspendUser(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))

	var req dto.SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.SuspendUser(adminID, id, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User suspended successfully", nil, nil)
}

func (h *Handler) BanUser(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))

	var req dto.BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.BanUser(adminID, id, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User banned successfully", nil, nil)
}

func (h *Handler) ActivateUser(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))

	var req dto.ActivateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.ActivateUser(adminID, id, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User reactivated successfully", nil, nil)
}

func (h *Handler) DeleteUser(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))
	
	var req dto.DeleteUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.DeleteUser(adminID, id, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User deleted successfully", nil, nil)
}

func (h *Handler) RestoreUser(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	adminID, _ := uuid.Parse(c.GetString("userID"))

	var req dto.RestoreUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.service.RestoreUser(adminID, id, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User restored successfully", nil, nil)
}

func (h *Handler) GetUserAuditLogs(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))
	logs, err := h.service.GetUserAuditLogs(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Audit logs retrieved successfully", logs, nil)
}

// user routes
func (h *Handler) GetProfile(c *gin.Context) {
    id, _ := uuid.Parse(c.GetString("userID"))
    
	user, err := h.service.GetUserByID(id)
	mappedUser := dto.UserResponse{
		ID:       user.ID.String(),
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		Status: user.Status,
	}
	if err != nil {
        response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
    
	response.Success(c, http.StatusOK, "User profile retrieved successfully", mappedUser, nil)
}


func (h *Handler) GetUserByEmail(c *gin.Context) {
    email := c.Param("email")

	user, err := h.service.GetUserByEmail(email)
	mappedUser := dto.UserResponse{
		ID:       user.ID.String(),
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		Status: user.Status,
	}
	if err != nil {
        response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
    
	response.Success(c, http.StatusOK, "User retrieved successfully", mappedUser, nil)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
    id, _ := uuid.Parse(c.GetString("userID"))
    var req dto.UpdateProfileRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
        return
    }

    user, err := h.service.GetUserByID(id)
    mappedUser := dto.UserResponse{
        ID:       user.ID.String(),
        Name:     user.Name,
        Email:    user.Email,
        Role:     user.Role,
        Status: user.Status,
    }
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
        return
    }

    user.Name = req.Name
    user.Email = req.Email

    user, err = h.service.UpdateProfile(user)
    mappedUser = dto.UserResponse{
        ID:       user.ID.String(),
        Name:     user.Name,
        Email:    user.Email,
        Role:     user.Role,
        Status: user.Status,
    }
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
        return
    }

    response.Success(c, http.StatusOK, "Profile updated successfully", mappedUser, nil)
}

func (h *Handler) RequestSeller(c *gin.Context) {
	id, _ := uuid.Parse(c.GetString("userID"))

	err := h.service.CreateSellerRequest(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Seller request submitted successfully", nil, nil)
}