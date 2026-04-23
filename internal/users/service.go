package users

import (
	"errors"

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

func (s *Service) Login(req LoginRequest) (string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)

	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := auth.GenerateJWT(
		user.ID.String(),
		string(user.Role),
		s.jwtSecret,
	)

	if err != nil {
		return "", nil, err
	}

	return token, user, nil
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