package users

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
	"golang.org/x/crypto/bcrypt"
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

func (s *Service) GetUsers(cursor, limitStr, search string, role models.Role, status models.Status, sortedBy string, sortOrder string) ([]models.User, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	users, total, hasMore, nextCursor, err := s.repo.GetUsers(cursor, limit, search, role, status, sortedBy, sortOrder)
	if err != nil {
		return nil, 0, false, "", err
	}
	return users, total, hasMore, nextCursor, nil
}

func (s *Service) GetDeletedUsers(cursor, limitStr, search string, sortedBy string, sortOrder string) ([]models.User, int64, bool, string, error) {
	limit := 20

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	users, total, hasMore, nextCursor, err := s.repo.GetDeletedUsers(cursor, limit, search, sortedBy, sortOrder)
	if err != nil {
		return nil, 0, false, "", err
	}
	return users, total, hasMore, nextCursor, nil
}

func (s *Service) ChangeRole(adminID uuid.UUID, userID uuid.UUID, role models.Role, reason string) error {
	return s.repo.UpdateUserRole(userID, role, adminID, reason)
}

func (s *Service) SuspendUser(adminID uuid.UUID, userID uuid.UUID, reason string) error {
	return s.repo.SuspendUser(userID, adminID, reason)
}

func (s *Service) BanUser(adminID uuid.UUID, userID uuid.UUID, reason string) error {
	return s.repo.BanUser(userID, adminID, reason)
}

func (s *Service) ActivateUser(adminID uuid.UUID, userID uuid.UUID, reason string) error {
	return s.repo.ActivateUser(userID, adminID, reason)
}

func (s *Service) DeleteUser(adminID uuid.UUID, userID uuid.UUID, reason string) error {
	return s.repo.DeleteUser(userID, adminID, reason)
}

func (s *Service) RestoreUser(adminID uuid.UUID, userID uuid.UUID, reason string) error {
	return s.repo.RestoreUser(userID, adminID, reason)
}

func (s *Service) GetAuditLogs() ([]models.AuditLog, error) {
	return s.repo.GetAuditLogs()
}

func (s *Service) GetUserAuditLogs(userID uuid.UUID) ([]models.AuditLog, error) {
	return s.repo.GetUserAuditLogs(userID)
}

// user routes
func (s *Service) GetUserByID(userID uuid.UUID) (*models.User, error) {
	fmt.Println("service get user id: ", userID)
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