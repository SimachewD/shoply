package users

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrPromoteToAdmin         = errors.New("you don't have permission to promote users to admin")
	ErrPromoteToSuperAdmin    = errors.New("you don't have permission to promote users to super admin")
	ErrChangeSuperAdminRole   = errors.New("you can't change super admin's role")
	ErrSuspendYourself        = errors.New("you can't suspend yourself")
	ErrBanYourself            = errors.New("you can't ban yourself")
	ErrSuspendAdmins          = errors.New("you don't have permission to suspend admins")
	ErrBanAdmins              = errors.New("you don't have permission to ban admins")
	ErrDeleteYourself         = errors.New("you can't delete yourself")
	ErrDeleteAdmins           = errors.New("you don't have permission to delete admins")
	ErrActivateAdmins         = errors.New("you don't have permission to activate admins")
	ErrRestoreAdmins          = errors.New("you don't have permission to restore admins")
	
	ErrUnauthorized           = errors.New("unauthorized")
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Service) Register(req RegisterRequest) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         models.RoleBuyer,
	}

	return s.repo.Register(user)
}

func (s *Service) Login(req LoginRequest) (string, string, error) {
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}

	// access token
	accessToken, err := auth.GenerateAccessToken(
		user.ID,
		string(user.Role),
		s.jwtSecret,
	)
	if err != nil {
		return "", "", err
	}

	// refresh token
	refreshToken, jti, err := auth.GenerateRefreshToken(
		user.ID,
		s.jwtSecret,
	)
	if err != nil {
		return "", "", err
	}

	// store refresh token
	err = s.repo.StoreRefreshToken(
		user.ID,
		jti,
		time.Now().Add(auth.RefreshTokenTTL),
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) Refresh(oldToken string) (string, string, error) {
	claims, err := auth.ValidateRefreshToken(oldToken, s.jwtSecret)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	// check if exists
	exists, err := s.repo.GetRefreshToken(claims.JTI)
	if err != nil || !exists {
		return "", "", errors.New("refresh token not found")
	}

	// delete old (rotation)
	err = s.repo.DeleteRefreshToken(claims.JTI)
	if err != nil {
		return "", "", err
	}

	user, err := s.repo.GetUserByID(claims.UserID)
	if err != nil {
		return "", "", err
	}

	// generate new access
	accessToken, err := auth.GenerateAccessToken(
		claims.UserID,
		string(user.Role),
		s.jwtSecret,
	)
	if err != nil {
		return "", "", err
	}

	// generate new refresh
	newRefresh, newJTI, err := auth.GenerateRefreshToken(
		claims.UserID,
		s.jwtSecret,
	)
	if err != nil {
		return "", "", err
	}

	// store new refresh
	err = s.repo.StoreRefreshToken(
		claims.UserID,
		newJTI,
		time.Now().Add(auth.RefreshTokenTTL),
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefresh, nil
}

func (s *Service) Logout(refreshToken string) error {
	claims, err := auth.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil // don't expose info
	}

	return s.repo.DeleteRefreshToken(claims.JTI)
}

// admin routes
func (s *Service) GetSellerRequests() ([]models.SellerRequest, error) {
	return s.repo.GetSellerRequests()
}

func (s *Service) GetUsers(cursor, limitStr, search string, role models.Role, status models.Status, sortBy string, sortOrder string, requestorRole string) ([]models.User, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	includeSuperAdmin := requestorRole == string(models.RoleSuperAdmin)

	users, total, hasMore, nextCursor, err := s.repo.GetUsers(cursor, limit, search, role, status, sortBy, sortOrder, includeSuperAdmin)
	if err != nil {
		return nil, 0, false, "", err
	}
	return users, total, hasMore, nextCursor, nil
}

func (s *Service) GetDeletedUsers(cursor, limitStr, search string, sortBy string, sortOrder string) ([]models.User, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	users, total, hasMore, nextCursor, err := s.repo.GetDeletedUsers(cursor, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, 0, false, "", err
	}
	return users, total, hasMore, nextCursor, nil
}

func (s *Service) ChangeRole(actorID uuid.UUID, userID uuid.UUID, role models.Role, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	user, err := s.repo.GetUserByID(userID)
	actor, err := s.repo.GetUserByID(actorID)

	if user.Role == role {
		return errors.New("user already has " + string(user.Role) + " role")
	}

	if err != nil {
		if strings.Contains(err.Error(), "no rows"){
			return ErrUserNotFound
		}
		return err
	}

	if requestorRole != string(models.RoleSuperAdmin) && role == models.RoleAdmin {
		return ErrPromoteToAdmin
	}
	if role == models.RoleSuperAdmin {
		return ErrPromoteToSuperAdmin
	}
	if user.Role == models.RoleSuperAdmin {
		return ErrChangeSuperAdminRole
	}

	metadata := map[string]any{
		"oldRole": user.Role,
		"newRole": role,
		"reason":  reason,
	}
	return s.repo.UpdateUserRole(userID, role, actorID, actor.Name, metadata, ipAddress, userAgent)
}

func (s *Service) SuspendUser(actorID uuid.UUID, userID uuid.UUID, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	if actorID == userID {
		return ErrSuspendYourself
	}
	
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows"){
			return ErrUserNotFound
		}
		return err
	}
	if requestorRole != string(models.RoleSuperAdmin) && (user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdmin) {
		return ErrSuspendAdmins
	}
	
	metadata := map[string]any{
		"oldStatus": user.Status,
		"newStatus": "suspended",
		"reason":    reason,
	}
	return s.repo.SuspendUser(userID, actorID, user.Name, metadata, ipAddress, userAgent)
}

func (s *Service) BanUser(actorID uuid.UUID, userID uuid.UUID, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	if actorID == userID {
		return ErrBanYourself
	}

	user, err := s.repo.GetUserByID(actorID)
	if err != nil {
		return err
	}
	if requestorRole != string(models.RoleSuperAdmin) && (user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdmin) {
		return ErrBanAdmins
	}
	metadata := map[string]any{
		"oldStatus": user.Status,
		"newStatus": "banned",
		"reason":    reason,
	}
	return s.repo.BanUser(userID, actorID, user.Name, metadata, ipAddress, userAgent)
}

func (s *Service) ActivateUser(actorID uuid.UUID, userID uuid.UUID, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	user, err := s.repo.GetUserByID(actorID)
	if err != nil {
		return err
	}
	if requestorRole != string(models.RoleSuperAdmin) && (user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdmin) {
		return ErrActivateAdmins
	}
	
	metadata := map[string]any{
		"oldStatus": user.Status,
		"newStatus": "active",
		"reason":    reason,
	}
	return s.repo.ActivateUser(userID, actorID, user.Name, metadata, ipAddress, userAgent)
}

func (s *Service) DeleteUser(actorID uuid.UUID, userID uuid.UUID, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	if actorID == userID {
		return ErrDeleteYourself
	}

	user, err := s.repo.GetUserByID(actorID)
	if err != nil {
		return err
	}
	if requestorRole != string(models.RoleSuperAdmin) && (user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdmin) {
		return ErrDeleteAdmins
	}

	metadata := map[string]any{
		"oldStatus": user.Status,
		"newStatus": "deleted",
		"reason":    reason,
	}
	return s.repo.DeleteUser(userID, actorID, user.Name, metadata, ipAddress, userAgent)
}

func (s *Service) RestoreUser(actorID uuid.UUID, userID uuid.UUID, reason string, ipAddress *string, userAgent *string, requestorRole string) error {
	user, err := s.repo.GetUserByID(actorID)
	if err != nil {
		return err
	}
	if requestorRole != string(models.RoleSuperAdmin) && (user.Role == models.RoleSuperAdmin || user.Role == models.RoleAdmin) {
		return ErrRestoreAdmins
	}

	metadata := map[string]any{
		"oldStatus": user.Status,
		"newStatus": "active",
		"reason":    reason,
	}

	return s.repo.RestoreUser(
		userID,
		actorID,
		user.Name,
		metadata,
		ipAddress,
		userAgent,
	)
}

func (s *Service) GetAuditLogs(cursor, limitStr, search string, sortBy string, sortOrder string) ([]models.AuditLog, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	logs, total, hasMore, nextCursor, err := s.repo.GetAuditLogs(cursor, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, 0, false, "", err
	}
	return logs, total, hasMore, nextCursor, nil
}

func (s *Service) GetUserAuditLogs(userID uuid.UUID, cursor, limitStr, search string, sortBy string, sortOrder string) ([]models.AuditLog, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	logs, total, hasMore, nextCursor, err := s.repo.GetUserAuditLogs(userID, cursor, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, 0, false, "", err
	}
	return logs, total, hasMore, nextCursor, nil
}

// user routes
func (s *Service) GetUserByID(userID uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *Service) GetUserProfile(userID uuid.UUID, requestorRole string) (*models.User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if requestorRole == string(models.RoleSuperAdmin) {
		return user, nil
	}
	if requestorRole == string(models.RoleAdmin) && user.Role != models.RoleSuperAdmin {
		return user, nil
	}
	if requestorRole == string(models.RoleModerator) && (
		user.Role == models.RoleModerator || 
		user.Role == models.RoleEditor || 
		user.Role == models.RoleSupport || 
		user.Role == models.RoleSeller || 
		user.Role == models.RoleBuyer) {
		return user, nil
	}
	if requestorRole == string(models.RoleSupport) && (
		user.Role == models.RoleSupport || 
		user.Role == models.RoleEditor || 
		user.Role == models.RoleModerator || 
		user.Role == models.RoleSeller || 
		user.Role == models.RoleBuyer) {
		return user, nil
	}
	if requestorRole == string(models.RoleSeller) && (
		user.Role == models.RoleSeller || 
		user.Role == models.RoleBuyer) {
		return user, nil
	}
	if requestorRole == string(models.RoleBuyer) && user.Role == models.RoleBuyer {
		return user, nil
	}
	return nil, ErrUnauthorized
}

func (s *Service) GetMyProfile(userID uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *Service) GetUserByEmail(email string) (*models.User, error) {
	return s.repo.GetUserByEmail(email)
}

func (s *Service) UpdateProfile(user *models.User) (*models.User, error) {
	return s.repo.UpdateProfile(user)
}

func (s *Service) CreateSellerRequest(userID uuid.UUID) error {
	return s.repo.CreateSellerRequest(userID)
}
