package users

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/sime/shoply/internal/models"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) Register(user *models.User) (*models.User, error) {
	err := r.DB.QueryRow(`
		INSERT INTO users (
			name, email, password_hash, role
		) VALUES ($1,$2,$3,$4) RETURNING id, name, email, role, created_at, updated_at`,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	return user, err
}

// admin routes
func (r *Repository) GetAllUsers() ([]models.User, error) {
	rows, err := r.DB.Query("SELECT id, name, email, role, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *Repository) UpdateUserRole(userID string, role models.Role) error {
	_, err := r.DB.Exec(
		"UPDATE users SET role=$1 WHERE id=$2",
		role,
		userID,
	)
	return err
}

func (r *Repository) GetPendingRequests() ([]models.SellerRequest, error) {
	rows, err := r.DB.Query(`
		SELECT id, user_id, status, created_at
		FROM seller_requests
		WHERE status = 'pending'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.SellerRequest

	for rows.Next() {
		var r models.SellerRequest
		rows.Scan(&r.ID, &r.UserID, &r.Status, &r.CreatedAt)
		list = append(list, r)
	}

	return list, nil
}

func (r *Repository) DeleteUser(userID string) (*models.User, error) {
	var user models.User
	err := r.DB.QueryRow(
		"DELETE FROM users WHERE id=$1 RETURNING id, name, email, role, created_at, updated_at",
		userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	return &user, err
}

// user routes
func (r *Repository) GetUserByID(userID string) (*models.User, error) {
	var u models.User
	err := r.DB.QueryRow("SELECT id, name, email, role FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
	return &u, err
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User

	err := r.DB.QueryRow(`
		SELECT id, name, email, password_hash, role
		FROM users
		WHERE email=$1
	`, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) UpdateUser(user *models.User) (*models.User, error) {
	err := r.DB.QueryRow(
		"UPDATE users SET name=$1, email=$2, role=$3, updated_at=NOW() WHERE id=$4 RETURNING id, name, email, role, created_at, updated_at",
		user.Name,
		user.Email,
		user.Role,
		user.ID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) CreateSellerRequest(userID string) error {
	_, err := r.DB.Exec(`
		INSERT INTO seller_requests (id, user_id, status, created_at)
		VALUES ($1, $2, 'pending', NOW())
	`, uuid.New(), userID)

	return err
}
