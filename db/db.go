package db

import (
	"database/sql"
	"fmt"

	"github.com/dzeckelev/geth-wrapper/config"

	_ "github.com/lib/pq" // Need for postgres driver.
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

// ConnectArgs returns connection string for database connection.
func ConnectArgs(cfg *config.DB) string {
	// TODO: ssl mode is disabled.
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s"+
		" port=%d sslmode=disable", cfg.Host, cfg.User, cfg.Password,
		cfg.DBName, cfg.Port)
}

// Connect connects to a database.
func Connect(connectArgs string) (*reform.DB, error) {
	conn, err := sql.Open("postgres", connectArgs)
	if err == nil {
		err = conn.Ping()
	}
	if err != nil {
		return nil, err
	}

	return reform.NewDB(conn, postgresql.Dialect, nil), nil
}

// CloseDB closes database.
func CloseDB(db *reform.DB) {
	db.DBInterface().(*sql.DB).Close()
}
