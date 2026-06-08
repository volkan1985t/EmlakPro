package repository

import (
	"database/sql"
)

type DeviceTokenRepository struct {
	db *sql.DB
}

func NewDeviceTokenRepository(db *sql.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

// ResolveUserID — aktif bir cihaz token'ını sahibinin user_id'sine çevirir.
// Geçersiz/pasif/bulunamayan token'da (0, false) döner.
// last_used_at damgasını best-effort günceller (hata yutulur).
func (r *DeviceTokenRepository) ResolveUserID(token string) (int64, bool) {
	if token == "" {
		return 0, false
	}
	var userID int64
	err := r.db.QueryRow(
		`SELECT user_id FROM device_tokens WHERE token=$1 AND is_active=TRUE`, token,
	).Scan(&userID)
	if err != nil {
		return 0, false
	}
	_, _ = r.db.Exec(`UPDATE device_tokens SET last_used_at=NOW() WHERE token=$1`, token)
	return userID, true
}
