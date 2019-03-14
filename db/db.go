package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // Need for postgres driver.
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/dzeckelev/geth-wrapper/config"
)

// ConnectArgs returns connection string for database connection.
func ConnectArgs(cfg *config.DB) string {
	// TODO: ssl mode is disabled.
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s"+
		" port=%d sslmode=disable", cfg.Host, cfg.User, cfg.Password,
		cfg.DBName, cfg.Port)
}

func Connect(connectArgs string) (*sql.DB, error) {
	return sql.Open("postgres", connectArgs)
}

// NewDB connects to a database.
func NewDB(conn *sql.DB) (*reform.DB, error) {
	return reform.NewDB(conn, postgresql.Dialect, nil), nil
}

// CloseDB closes database.
func CloseDB(db *reform.DB) {
	_ = db.DBInterface().(*sql.DB).Close()
}
