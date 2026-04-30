package users

import (
	"errors"
	"time"

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

func (s *Service) Login(req LoginRequest) (string, string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return "", "", nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return "", "", nil, errors.New("invalid credentials")
	}

	// access token
	accessToken, err := auth.GenerateAccessToken(
		user.ID.String(),
		string(user.Role),
		s.jwtSecret,
	)
	if err != nil {
		return "", "", nil, err
	}

	// refresh token
	refreshToken, jti, err := auth.GenerateRefreshToken(
		user.ID.String(),
		s.jwtSecret,
	)
	if err != nil {
		return "", "", nil, err
	}

	// store refresh token
	err = s.repo.StoreRefreshToken(
		user.ID.String(),
		jti,
		time.Now().Add(auth.RefreshTokenTTL),
	)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

//
// 🔁 REFRESH
//

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
	_ = s.repo.DeleteRefreshToken(claims.JTI)

	// generate new access
	accessToken, err := auth.GenerateAccessToken(
		claims.UserID,
		"", // role optional here (or fetch if needed)
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
	_ = s.repo.StoreRefreshToken(
		claims.UserID,
		newJTI,
		time.Now().Add(auth.RefreshTokenTTL),
	)

	return accessToken, newRefresh, nil
}

//
// 🚪 LOGOUT
//

func (s *Service) Logout(refreshToken string) error {
	claims, err := auth.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil // don't expose info
	}

	return s.repo.DeleteRefreshToken(claims.JTI)
}

// admin routes
func (s *Service) GetAllUsers() ([]models.User, error) {
	return s.repo.GetAllUsers()
}

func (s *Service) GetPendingRequests() ([]models.SellerRequest, error) {
	return s.repo.GetPendingRequests()
}

func (s *Service) DeleteUser(userID string) (*models.User, error) {
	return s.repo.DeleteUser(userID)
}

func (s *Service) ChangeUserRole(userID string, role models.Role) error {
	return s.repo.UpdateUserRole(userID, role)
}

// user routes
func (s *Service) GetUserByID(userID string) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *Service) GetUserByEmail(email string) (*models.User, error) {
	return s.repo.GetUserByEmail(email)
}

func (s *Service) UpdateUser(user *models.User) (*models.User, error) {
	return s.repo.UpdateUser(user)
}

func (s *Service) CreateSellerRequest(userID string) error {
	return s.repo.CreateSellerRequest(userID)
}