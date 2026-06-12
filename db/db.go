// Package db owns the PostgreSQL connection and the schema/cleanup helpers.
package db

import (
	"database/sql"
	"time"

	// Registers the "pgx" driver with database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect opens a connection pool to PostgreSQL and verifies it works.
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetConnMaxIdleTime(30 * time.Second)

	// Fail fast at startup if the database is unreachable, rather than on
	// the first API request.
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate creates the latency_history table if it does not already exist.
// Running it repeatedly is safe.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS latency_history (
			world       TEXT      NOT NULL,
			channel     INTEGER   NOT NULL,
			recorded_at BIGINT    NOT NULL,
			latency_ms  REAL      NOT NULL,
			PRIMARY KEY (world, channel, recorded_at)
		);
	`)
	if err != nil {
		return err
	}

	// The primary key's btree already covers every query shape (Postgres
	// scans it backwards for DESC), so the old standalone lookup index was
	// pure write amplification. Remove it from deployments that have it.
	_, err = db.Exec(`DROP INDEX IF EXISTS idx_latency_lookup;`)
	return err
}

// CleanupOldRows deletes history rows older than the retention window and
// returns how many rows were removed.
func CleanupOldRows(db *sql.DB, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention).UnixMilli()

	result, err := db.Exec(`DELETE FROM latency_history WHERE recorded_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
