package repository

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *model.User) error {
	return r.db.QueryRow(`
		INSERT INTO users (username, email, password_hash, full_name, role, is_active, telegram_chat_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		u.Username, u.Email, u.PasswordHash, u.FullName, u.Role, u.IsActive, nullStr(u.TelegramChatID),
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// CreateRaw — ensureAdmin için, ON CONFLICT ile idempotent
func (r *UserRepository) CreateRaw(username, email, passwordHash, fullName, role string, isActive bool) error {
	_, err := r.db.Exec(`
		INSERT INTO users (username, email, password_hash, full_name, role, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (username) DO NOTHING`,
		username, email, passwordHash, fullName, role, isActive,
	)
	return err
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	u := &model.User{}
	var chatID sql.NullString
	err := r.db.QueryRow(`
		SELECT id,username,email,password_hash,full_name,role,is_active,telegram_chat_id,COALESCE(notify_telegram,true),created_at,updated_at
		FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.FullName, &u.Role, &u.IsActive, &chatID, &u.NotifyTelegram, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.TelegramChatID = chatID.String
	return u, err
}

func (r *UserRepository) GetByID(id int64) (*model.User, error) {
	u := &model.User{}
	var chatID sql.NullString
	err := r.db.QueryRow(`
		SELECT id,username,email,password_hash,full_name,role,is_active,telegram_chat_id,COALESCE(notify_telegram,true),created_at,updated_at
		FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.FullName, &u.Role, &u.IsActive, &chatID, &u.NotifyTelegram, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.TelegramChatID = chatID.String
	return u, err
}

func (r *UserRepository) List() ([]model.User, error) {
	rows, err := r.db.Query(`
		SELECT id,username,email,full_name,role,is_active,COALESCE(telegram_chat_id,''),COALESCE(notify_telegram,true),created_at,updated_at
		FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FullName,
			&u.Role, &u.IsActive, &u.TelegramChatID, &u.NotifyTelegram, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) SetActive(id int64, active bool) error {
	_, err := r.db.Exec(
		`UPDATE users SET is_active=$1, updated_at=NOW() WHERE id=$2 AND role!='admin'`,
		active, id)
	return err
}

func (r *UserRepository) Delete(id int64) error {
	res, err := r.db.Exec(`DELETE FROM users WHERE id=$1 AND role!='admin'`, id)
	if err != nil {
		return fmt.Errorf("kullanıcı silinemedi: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("kullanıcı bulunamadı veya admin silinemez")
	}
	return nil
}

func (r *UserRepository) SetTelegramChatID(id int64, chatID string) error {
	_, err := r.db.Exec(`UPDATE users SET telegram_chat_id=$1, updated_at=NOW() WHERE id=$2`, nullStr(chatID), id)
	return err
}

func (r *UserRepository) ListWithChatIDs() ([]model.User, error) {
	rows, err := r.db.Query(`
		SELECT id, full_name, COALESCE(telegram_chat_id,''), COALESCE(notify_telegram,true)
		FROM users WHERE is_active=true AND telegram_chat_id IS NOT NULL AND telegram_chat_id != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		rows.Scan(&u.ID, &u.FullName, &u.TelegramChatID, &u.NotifyTelegram)
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) GetByTelegramChatID(chatID int64) (*model.User, error) {
	u := &model.User{}
	var chatIDStr sql.NullString
	err := r.db.QueryRow(`
		SELECT id,username,email,password_hash,full_name,role,is_active,telegram_chat_id,COALESCE(notify_telegram,true),created_at,updated_at
		FROM users WHERE telegram_chat_id=$1 AND is_active=true`,
		strconv.FormatInt(chatID, 10),
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.FullName, &u.Role, &u.IsActive, &chatIDStr, &u.NotifyTelegram, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.TelegramChatID = chatIDStr.String
	return u, err
}

func nullStr(s string) interface{} {
	if s == "" { return nil }
	return s
}

func (r *UserRepository) AdminExists() (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin'`).Scan(&count)
	return count > 0, err
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE username=$1`, username).Scan(&count)
	return count > 0, err
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email=$1`, email).Scan(&count)
	return count > 0, err
}

// ── Refresh Token ─────────────────────────────────────────────────────────────

func (r *UserRepository) SaveRefreshToken(userID int64, token string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1,$2,$3)`,
		userID, token, expiresAt,
	)
	return err
}

func (r *UserRepository) GetRefreshToken(token string) (*model.RefreshToken, error) {
	rt := &model.RefreshToken{}
	err := r.db.QueryRow(`
		SELECT id,user_id,token,expires_at,created_at
		FROM refresh_tokens WHERE token=$1 AND expires_at > NOW()`, token,
	).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rt, err
}

func (r *UserRepository) DeleteRefreshToken(token string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token=$1`, token)
	return err
}

func (r *UserRepository) DeleteUserRefreshTokens(userID int64) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id=$1`, userID)
	return err
}
