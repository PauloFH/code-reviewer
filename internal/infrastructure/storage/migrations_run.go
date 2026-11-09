package storage

import (
	"database/sql"
	"fmt"
)

// migrations[i] is schema version i+1; append only.
var migrations = []string{
	migration001, migration002, migration003, migration004, migration005,
	migration006, migration007, migration008, migration009, migration010,
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return err
	}
	for i, m := range migrations {
		if err := applyMigration(db, i+1, m); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	return nil
}

// applyMigration runs one migration and records it atomically, so a crash in
// between cannot leave an ALTER TABLE applied but unrecorded.
func applyMigration(db *sql.DB, version int, stmt string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit
	var applied int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version=?`, version).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}
	if _, err := tx.Exec(stmt); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, version, now()); err != nil {
		return err
	}
	return tx.Commit()
}
