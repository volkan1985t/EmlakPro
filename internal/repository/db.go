package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/config"
	_ "github.com/lib/pq"
)

func NewDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("DB bağlantısı açılamadı: %w", err)
	}

	// 50 kullanıcı için uygun pool ayarları
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 { maxOpen = 25 }
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 { maxIdle = 10 }

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(30 * time.Minute)  // 30dk'dan eski bağlantıları yenile
	db.SetConnMaxIdleTime(10 * time.Minute)  // 10dk idle kalan bağlantıyı kapat

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("DB ping başarısız: %w", err)
	}

	log.Printf("[DB] pool: maxOpen=%d maxIdle=%d lifetime=30m idleTime=10m", maxOpen, maxIdle)
	return db, nil
}
