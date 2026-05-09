package users

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sime/shoply/internal/models"
	"github.com/sime/shoply/internal/utils"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) Register(user *models.User) (*models.User, error) {
	err := r.DB.QueryRow(`
		INSERT INTO users (name, email, password_hash, role) 
		VALUES ($1,$2,$3,$4) RETURNING id, name, email, role, created_at, updated_at`,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	return user, err
}

func (r *Repository) StoreRefreshToken(userID uuid.UUID, jti string, expiresAt time.Time) error {
	hashed := utils.HashToken(jti)

	_, err := r.DB.Exec(`INSERT INTO refresh_tokens (user_id, jti, expires_at) VALUES ($1, $2, $3)`, userID, hashed, expiresAt)

	return err
}

func (r *Repository) GetRefreshToken(jti string) (bool, error) {
	hashed := utils.HashToken(jti)

	var exists int
	err := r.DB.QueryRow(`SELECT 1 FROM refresh_tokens WHERE jti = $1 AND expires_at > NOW()`, hashed).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}

func (r *Repository) DeleteRefreshToken(jti string) error {
	hashed := utils.HashToken(jti)

	_, err := r.DB.Exec(`DELETE FROM refresh_tokens WHERE jti = $1`, hashed)

	return err
}

// admin routes
func (r *Repository) GetSellerRequests() ([]models.SellerRequest, error) {
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

func (r *Repository) GetUsers(cursor string, limit int, search string, role models.Role, status models.Status, sortedBy string, sortOrder string) ([]models.User, int64, bool, string, error) {

	query := `SELECT id, name, email, role, status, created_at, updated_at FROM users WHERE deleted_at IS NULL AND 1=1`

	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND 1=1`

	args := []any{}
	countArgs := []any{}

	argPos := 1

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argPos, argPos)

		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argPos, argPos)

		searchValue := "%" + search + "%"

		args = append(args, searchValue)
		countArgs = append(countArgs, searchValue)

		argPos++
	}

	if role != "" {
		query += fmt.Sprintf(" AND role=$%d", argPos)
		countQuery += fmt.Sprintf(" AND role=$%d", argPos)

		args = append(args, role)
		countArgs = append(countArgs, role)

		argPos++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", argPos)
		countQuery += fmt.Sprintf(" AND status=$%d", argPos)

		args = append(args, status)
		countArgs = append(countArgs, status)

		argPos++
	}

	if cursor != "" {

		parts := strings.Split(cursor, "_")

		if len(parts) == 2 {

			cursorTime := parts[0]
			cursorID := parts[1]

			query += fmt.Sprintf(` AND (created_at < $%d OR (created_at = $%d AND id < $%d))`, argPos, argPos, argPos+1)

			args = append(args, cursorTime, cursorID)

			argPos += 2
		}
	}

	if sortedBy == "" {
		sortedBy = "created_at"
	}

	if sortOrder == "" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortedBy, sortOrder, argPos)
	args = append(args, limit + 1)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, false, "", err
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var u models.User

		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.Status,
			&u.CreatedAt,
			&u.UpdatedAt,
		)

		if err != nil {
			return nil, 0, false, "", err
		}

		users = append(users, u)
	}

	var total int64

	err = r.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, false, "", err
	}

	hasMore := len(users) > limit

	if hasMore {
		users = users[:limit]
	}

	nextCursor := ""

	if len(users) > 0 {
		last := users[len(users)-1]

		nextCursor = fmt.Sprintf("%s_%s",last.CreatedAt.Format(time.RFC3339Nano),last.ID.String())
	}

	return users, total, hasMore, nextCursor, nil
}

func (r *Repository) GetDeletedUsers(cursor string, limit int, search string, sortedBy string, sortOrder string) ([]models.User, int64, bool, string, error) {
	query := `SELECT id, name, email, role, status, created_at, updated_at, deleted_at, deleted_by FROM users WHERE deleted_at IS NOT NULL AND 1=1`

	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NOT NULL AND 1=1`

	args := []any{}
	countArgs := []any{}

	argPos := 1

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argPos, argPos)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argPos, argPos)

		searchValue := "%" + search + "%"

		args = append(args, searchValue)
		countArgs = append(countArgs, searchValue)

		argPos++
	}

	if cursor != "" {

		parts := strings.Split(cursor, "_")

		if len(parts) == 2 {

			cursorTime := parts[0]
			cursorID := parts[1]

			query += fmt.Sprintf(` AND (created_at < $%d OR (created_at = $%d AND id < $%d))`, argPos, argPos, argPos+1)

			args = append(args, cursorTime, cursorID)

			argPos += 2
		}
	}

	if sortedBy == "" {
		sortedBy = "deleted_at"
	}

	if sortOrder == "" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortedBy, sortOrder, argPos)
	args = append(args, limit + 1)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, false, "", err
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var u models.User

		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.Status,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.DeletedAt,
			&u.DeletedBy,
		)

		if err != nil {
			return nil, 0, false, "", err
		}

		users = append(users, u)
	}

	var total int64

	err = r.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, false, "", err
	}

	hasMore := len(users) > limit

	if hasMore {
		users = users[:limit]
	}

	nextCursor := ""

	if len(users) > 0 {
		last := users[len(users)-1]

		nextCursor = fmt.Sprintf("%s_%s",last.CreatedAt.Format(time.RFC3339Nano),last.ID.String())
	}

	return users, total, hasMore, nextCursor, nil
}

func (r *Repository) UpdateUserRole(userID uuid.UUID, role models.Role, adminID uuid.UUID, reason string) error {
	// we will save update role action to moderation_actions table and update status to active with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "update_role", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET role=$1 WHERE id=$2`, string(role), userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

func (r *Repository) SuspendUser(userID uuid.UUID, adminID uuid.UUID, reason string) error {
	// we will save suspend action to moderation_actions table and update status to suspended with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "suspend", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET suspended_until=NOW(), status='suspended' WHERE id=$1`, userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

func (r *Repository) ActivateUser(userID uuid.UUID, adminID uuid.UUID, reason string) error {
	// we will save restore action to moderation_actions table and update status to active with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "reactivate", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET suspended_until=NULL, banned_at=NULL, status='active' WHERE id=$1`, userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

func (r *Repository) BanUser(userID uuid.UUID, adminID uuid.UUID, reason string) error {
	// we will save ban action to moderation_actions table and update status to banned with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "ban", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET banned_at=NOW(), status='banned' WHERE id=$1`, userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

// soft delete
func (r *Repository) DeleteUser(userID uuid.UUID, adminID uuid.UUID, reason string) error {
	// we will save delete action to moderation_actions table and update status to deleted with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "delete", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET deleted_at=NOW(), deleted_by=$1, status='deleted' WHERE id=$2`, adminID, userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

func (r *Repository) RestoreUser(userID uuid.UUID, adminID uuid.UUID, reason string) error {
	// we will save restore action to moderation_actions table and update status to active with atomic transaction
	transaction,err := r.DB.Begin() 
	if err != nil {
		return err
	}

	_, err = transaction.Exec(`INSERT INTO audit_logs (admin_id, target_user_id, action, reason, created_at) VALUES ($1, $2, $3, $4, NOW())`, adminID, userID, "restore", reason)
	if err != nil {
		transaction.Rollback()
		return err
	}
	
	_, err = transaction.Exec(`UPDATE users SET deleted_at=NULL, deleted_by=NULL, status='active' WHERE id=$1`, userID)
	if err != nil {
		transaction.Rollback()
		return err
	}

	return transaction.Commit()
}

// audit logs
func (r *Repository) CreateAuditLog(adminID uuid.UUID, action string, targetUserID uuid.UUID) error {

	_, err := r.DB.Exec(`
		INSERT INTO audit_logs (
			id,
			admin_id,
			action,
			target_user_id,
			created_at
		)
		VALUES ($1,$2,$3,$4,NOW())
	`,
		uuid.New(),
		adminID,
		action,
		targetUserID,
	)

	return err
}

func (r *Repository) GetAuditLogs() ([]models.AuditLog, error) {
	query := `SELECT id, admin_id, action, target_user_id, created_at FROM audit_logs ORDER BY created_at DESC`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog

	for rows.Next() {
		var l models.AuditLog

		err := rows.Scan(
			&l.ID,
			&l.AdminID,
			&l.Action,
			&l.TargetUserID,
			&l.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
}

func (r *Repository) GetUserAuditLogs(userID uuid.UUID) ([]models.AuditLog, error) {

	query := `
		SELECT id, admin_id, action, target_user_id, created_at
		FROM audit_logs
		WHERE target_user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog = make([]models.AuditLog, 0)

	for rows.Next() {
		var l models.AuditLog

		err := rows.Scan(
			&l.ID,
			&l.AdminID,
			&l.Action,
			&l.TargetUserID,
			&l.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
}

// user routes
func (r *Repository) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.DB.QueryRow("SELECT id, name, email, role, status, created_at, updated_at FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User

	err := r.DB.QueryRow(`
		SELECT id, name, email, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE email=$1
		AND deleted_at IS NULL
	`, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		fmt.Println("Error getting user by email:", err)
		return nil, err
	}

	return &user, nil
}

func (r *Repository) UpdateProfile(user *models.User) (*models.User, error) {
	err := r.DB.QueryRow(
		"UPDATE users SET name=$1, email=$2, updated_at=NOW() WHERE id=$3 RETURNING id, name, email, role, created_at, updated_at",
		user.Name,
		user.Email,
		user.ID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) CreateSellerRequest(userID uuid.UUID) error {
	_, err := r.DB.Exec(`
		INSERT INTO seller_requests (id, user_id, status, created_at)
		VALUES ($1, $2, 'pending', NOW())
	`, uuid.New(), userID)

	return err
}
