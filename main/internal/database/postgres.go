package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/oraz200101/sandbox-server/main/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type PostgresClient struct {
	Db *sql.DB
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

func (p *PostgresClient) Migrate() error {
	src, err := iofs.New(migrationsFS, "migrations")

	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}

	driver, err := postgres.WithInstance(p.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func (p *PostgresClient) Close() error {
	return p.db.Close()
}
