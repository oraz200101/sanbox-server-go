package database

import (
	"database/sql"
	"fmt"

	"github.com/oraz200101/sandbox-server/main/internal/config"
)

type PostgresClient struct {
	db *sql.DB
}

func NewPostgresClient(cfg *config.Config) (*PostgresClient, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		cfg.DBPostgres.User,
		cfg.DBPostgres.Password,
		cfg.DBPostgres.Name,
		cfg.DBPostgres.Host,
		cfg.DBPostgres.Port,
	)

	db, err := sql.Open("postgres", dsn)

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetConnMaxIdleTime(5)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresClient{db: db}, nil
}

func (p *PostgresClient) Close() error {
	return p.db.Close()
}
