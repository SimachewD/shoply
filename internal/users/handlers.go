package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
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
	sortBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")

	requestorRole := c.GetString("role")

	users, total, hasMore, nextCursor, err := h.service.GetUsers(cursor, limitStr, search, role, status, sortBy, sortOrder, requestorRole)

	mappedUsers := make([]dto.UserResponse, len(users))
	meta := gin.H{
		"total":       total,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	}

	for i, u := range users {
		mappedUsers[i] = dto.UserResponse{
			ID:        u.ID.String(),
			Name:      u.Name,
			Email:     u.Email,
			Role:      u.Role,
			Status:    u.Status,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
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
	sortBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")

	users, total, hasMore, nextCursor, err := h.service.GetDeletedUsers(cursor, limitStr, search, sortBy, sortOrder)

	mappedUsers := make([]dto.UserResponse, len(users))
	meta := gin.H{
		"total":       total,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	}

	for i, u := range users {
		mappedUsers[i] = dto.UserResponse{
			ID:     u.ID.String(),
			Name:   u.Name,
			Email:  u.Email,
			Role:   u.Role,
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
	requestorRole := c.GetString("role")

	var req dto.ChangeRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	err := h.service.ChangeRole(adminID, id, req.Role, req.Reason, &ip, &ua, requestorRole)

	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrPromoteToAdmin),
				errors.Is(err, ErrPromoteToSuperAdmin),
				errors.Is(err, ErrChangeSuperAdminRole):

				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	requestorRole := c.GetString("role")

	err := h.service.SuspendUser(adminID, id, req.Reason, &ip, &ua, requestorRole)
	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrSuspendYourself):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			case errors.Is(err, ErrSuspendAdmins):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	requestorRole := c.GetString("role")

	err := h.service.BanUser(adminID, id, req.Reason, &ip, &ua, requestorRole)
	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrBanYourself):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			case errors.Is(err, ErrBanAdmins):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	requestorRole := c.GetString("role")

	err := h.service.ActivateUser(adminID, id, req.Reason, &ip, &ua, requestorRole)
	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrActivateAdmins):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	requestorRole := c.GetString("role")

	err := h.service.DeleteUser(adminID, id, req.Reason, &ip, &ua, requestorRole)
	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrDeleteYourself):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			case errors.Is(err, ErrDeleteAdmins):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
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

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	requestorRole := c.GetString("role")

	err := h.service.RestoreUser(adminID, id, req.Reason, &ip, &ua, requestorRole)
	if err != nil {
		switch {
			case errors.Is(err, ErrUserNotFound):
				response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")

			case errors.Is(err, ErrRestoreAdmins):
				response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())

			default:
				response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			}
			return
		}

	response.Success(c, http.StatusOK, "User restored successfully", nil, nil)
}

func (h *Handler) GetUserAuditLogs(c *gin.Context) {

	id, _ := uuid.Parse(c.Param("id"))

	cursor := c.Query("cursor")
	limitStr := c.Query("limit")
	search := c.Query("search")
	sortBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")

	logs, total, hasMore, nextCursor, err := h.service.GetUserAuditLogs(id, cursor, limitStr, search, sortBy, sortOrder)

	meta := gin.H{
		"total":       total,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	mappedLogs := make([]dto.AuditLogResponse, len(logs))
	for i, l := range logs {
		var metadata map[string]any

		if len(l.Metadata) > 0 {
			err := json.Unmarshal(l.Metadata, &metadata)
			if err != nil {
				metadata = nil
			}
		}
		mappedLogs[i] = dto.AuditLogResponse{
			ID:        l.ID.String(),
			UserID:    l.UserID.String(),
			ActorID:   l.ActorID.String(),
			ActorName: l.ActorName,
			Action:    l.Action,
			Resource:  l.Resource,
			Metadata:  metadata,
			IPAddress: l.IPAddress,
			UserAgent: l.UserAgent,
			CreatedAt: l.CreatedAt,
		}
	}

	response.Success(c, http.StatusOK, "Audit logs retrieved successfully", mappedLogs, meta)
}

// user routes
func (h *Handler) GetUserProfile(c *gin.Context) {
	requestorRole := c.GetString("role")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	user, err := h.service.GetUserProfile(id, requestorRole)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
	mappedUser := dto.UserResponse{
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", mappedUser, nil)
}

func (h *Handler) GetMyProfile(c *gin.Context) {
	id, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	user, err := h.service.GetMyProfile(id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Your profile not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
	mappedUser := dto.UserResponse{
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", mappedUser, nil)
}

func (h *Handler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	user, err := h.service.GetUserByEmail(email)
	mappedUser := dto.UserResponse{
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", mappedUser, nil)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	id, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
	var req dto.UpdateProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	user, err := h.service.GetUserByID(id)
	mappedUser := dto.UserResponse{
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
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
		ID:     user.ID.String(),
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
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
