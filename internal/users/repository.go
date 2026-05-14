package users

import (
	"database/sql"
	"encoding/json"
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

func (r *Repository) GetUsers(cursor string, limit int, search string, role models.Role, status models.Status, sortBy string, sortOrder string, includeSuperAdmin bool) ([]models.User, int64, bool, string, error) {

	query := `SELECT id, name, email, role, status, created_at, updated_at FROM users WHERE deleted_at IS NULL AND 1=1`

	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND 1=1`

	args := []any{}
	countArgs := []any{}
	argPos := 1

	if !includeSuperAdmin {
		query += fmt.Sprintf(` AND role <> '%s'`, models.RoleSuperAdmin)
		countQuery += fmt.Sprintf(` AND role <> '%s'`, models.RoleSuperAdmin)
	}

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
			cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
			if err != nil {
				return nil, 0, false, "", err
			}
			cursorID := parts[1]
			query += fmt.Sprintf(` AND (created_at < $%d OR (created_at = $%d AND id < $%d))`, argPos, argPos, argPos+1)
			args = append(args, cursorTime, cursorID)
			argPos += 2
		}
	}

	allowedSort := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"email":      true,
		"role":       true,
		"status":     true,
	}

	if !allowedSort[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortBy, sortOrder, argPos)
	args = append(args, limit+1)

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

		nextCursor = fmt.Sprintf("%s_%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
	}

	return users, total, hasMore, nextCursor, nil
}

func (r *Repository) GetDeletedUsers(cursor string, limit int, search string, sortBy string, sortOrder string) ([]models.User, int64, bool, string, error) {
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

	if sortBy == "" {
		sortBy = "deleted_at"
	}

	if sortOrder == "" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortBy, sortOrder, argPos)
	args = append(args, limit+1)

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

		nextCursor = fmt.Sprintf("%s_%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
	}

	return users, total, hasMore, nextCursor, nil
}

func (r *Repository) UpdateUserRole(userID uuid.UUID, role models.Role, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {
	// we will save update role action to audit_logs table and update status to active with atomic transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"update user role",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. update user
	_, err = tx.Exec(`UPDATE users SET role=$1 WHERE id=$2`, string(role), userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) SuspendUser(userID uuid.UUID, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {
	// we will save suspend action to audit_logs table and update status to suspended with atomic transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"suspend",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. suspend user
	_, err = tx.Exec(`UPDATE users SET suspended_until=NOW(), status='suspended' WHERE id=$1`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) ActivateUser(userID uuid.UUID, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {
	// we will save activate action to audit_logs table and update status to active with atomic transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"activate",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. activate user
	_, err = tx.Exec(`UPDATE users SET suspended_until=NULL, banned_at=NULL, status='active' WHERE id=$1`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) BanUser(userID uuid.UUID, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {
	// we will save ban action to audit_logs table and update status to banned with atomic transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"ban",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. ban user
	_, err = tx.Exec(`UPDATE users SET banned_at=NOW(), status='banned' WHERE id=$1`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// soft delete
func (r *Repository) DeleteUser(userID uuid.UUID, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"delete",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. delete user
	_, err = tx.Exec(`UPDATE users SET deleted_at=NOW(), deleted_by=$1, status='deleted' WHERE id=$2`, actorID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) RestoreUser(userID uuid.UUID, actorID uuid.UUID, actorName string, metadata map[string]any, ipAddress *string, userAgent *string) error {

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// marshal metadata -> jsonb
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// 1. insert audit log
	_, err = tx.Exec(`
		INSERT INTO audit_logs (actor_id,actor_name,action,resource,user_id,ip_address,user_agent,metadata,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`,
		actorID,
		actorName,
		"restore",
		"user",
		userID,
		ipAddress,
		userAgent,
		metaBytes,
	)
	if err != nil {
		return err
	}

	// 2. restore user
	_, err = tx.Exec(`UPDATE users SET deleted_at = NULL, deleted_by = NULL, status = 'active' WHERE id = $1`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// audit logs
func (r *Repository) GetAuditLogs(cursor string, limit int, search, sortBy, sortOrder string) ([]models.AuditLog, int64, bool, string, error) {
	query := `SELECT id, actor_id, action, user_id, resource, metadata created_at FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	args := []any{}
	countArgs := []any{}

	argPos := 1

	if cursor != "" {
		parts := strings.Split(cursor, "_")
		if len(parts) == 2 {
			cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
			if err != nil {
				return nil, 0, false, "", err
			}
			cursorID := parts[1]

			query += fmt.Sprintf(` AND (created_at < $%d OR (created_at = $%d AND id < $%d))`, argPos, argPos, argPos+1)
			args = append(args, cursorTime, cursorTime, cursorID)
			argPos += 2
		}
	}

	if search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, argPos, argPos+1)
		countQuery += fmt.Sprintf(` AND (name ILIKE $%d OR email ILIKE $%d)`, argPos, argPos+1)
		args = append(args, ""+search+"%")
		args = append(args, ""+search+"%")
		countArgs = append(countArgs, ""+search+"%")
		argPos += 2
	}

	allowedSort := map[string]bool{
		"actor_name": true,
		"action": true,
		"resource": true,
		"created_at": true,
		"updated_at": true,
	}

	if !allowedSort[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortBy, sortOrder, argPos)
	args = append(args, limit+1)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, false, "", err
	}
	defer rows.Close()

	var logs []models.AuditLog

	for rows.Next() {
		var l models.AuditLog

		err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.ActorID,
			&l.Action,
			&l.Resource,
			&l.Metadata,
			&l.CreatedAt,
		)

		if err != nil {
			return nil, 0, false, "", err
		}

		logs = append(logs, l)
	}

	var total int64
	err = r.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, false, "", err
	}

	hasMore := len(logs) > limit

	if hasMore {
		logs = logs[:limit]
	}

	nextCursor := ""

	if len(logs) > 0 {
		last := logs[len(logs)-1]
		nextCursor = fmt.Sprintf("%s_%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
	}

	return logs, total, hasMore, nextCursor, nil
}

func (r *Repository) GetUserAuditLogs(userID uuid.UUID, cursor string, limit int, search, sortBy, sortOrder string) ([]models.AuditLog, int64, bool, string, error) {

	query := `SELECT id, user_id, actor_id, actor_name, action, resource, metadata, ip_address, user_agent, created_at FROM audit_logs WHERE user_id = $1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE user_id = $1`

	args := []any{userID}
	countArgs := []any{userID}

	argPos := 2

	if search != "" {
		query += fmt.Sprintf(` AND (actor_name ILIKE $%d OR action ILIKE $%d)`, argPos, argPos+1)
		countQuery += fmt.Sprintf(` AND (actor_name ILIKE $%d OR action ILIKE $%d)`, argPos, argPos+1)
		args = append(args, ""+search+"%")
		args = append(args, ""+search+"%")
		countArgs = append(countArgs, ""+search+"%")
		argPos += 2
	}

	if cursor != "" {
		parts := strings.Split(cursor, "_")
		
		if len(parts) == 2{
			cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
			if err != nil {
				return nil, 0, false, "", err
			}
			cursorID := parts[1]
			query += fmt.Sprintf(` AND (created_at < $%d OR (created_at = $%d AND id < $%d))`, argPos, argPos, argPos+1)
			args = append(args, cursorTime, cursorTime, cursorID)
			argPos += 2
		}
	}

	allowedSort := map[string]bool{
		"actor_name": true,
		"action": true,
		"resource": true,
		"created_at": true,
		"updated_at": true,
	}

	if !allowedSort[sortBy] {
		sortBy = "created_at"
	}

	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d", sortBy, sortOrder, argPos)
	args = append(args, limit+1)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, false, "", err
	}
	defer rows.Close()

	var logs []models.AuditLog

	for rows.Next() {
		var l models.AuditLog

		err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.ActorID,
			&l.ActorName,
			&l.Action,
			&l.Resource,
			&l.Metadata,
			&l.IPAddress,
			&l.UserAgent,
			&l.CreatedAt,
		)

		if err != nil {
			return nil, 0, false, "", err
		}

		logs = append(logs, l)
	}

	var total int64
	err = r.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, false, "", err
	}

	hasMore := len(logs) > limit

	if hasMore {
		logs = logs[:limit]
	}

	nextCursor := ""

	if len(logs) > 0 {
		last := logs[len(logs)-1]
		nextCursor = fmt.Sprintf("%s_%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
	}

	return logs, total, hasMore, nextCursor, nil
}

// user routes
func (r *Repository) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.DB.QueryRow("SELECT id, name, email, role, status, created_at, updated_at FROM users WHERE id=$1", userID).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (r *Repository) GetMyProfile(userID uuid.UUID) (*models.User, error) {
	var user models.User

	err := r.DB.QueryRow(`
		SELECT id, name, email, role, status, created_at, updated_at 
		FROM users WHERE id=$1 AND deleted_at IS NULL`,
		userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
		return &user, err
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
