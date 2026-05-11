package repository

import (
	"database/sql"
	"fmt"

	"github.com/volkan1985t/EmlakPro/internal/config"
	_ "github.com/lib/pq"
)

func NewDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("DB bağlantısı açılamadı: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("DB ping başarısız: %w", err)
	}

	return db, nil
}
