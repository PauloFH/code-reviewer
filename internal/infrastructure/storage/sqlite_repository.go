package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository implements domain.ReviewRepository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at path and runs migrations.
func New(path string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite allows only one writer at a time

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reviewed_merge_requests (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			client_name TEXT    NOT NULL,
			project_ref TEXT    NOT NULL,
			mr_iid      INTEGER NOT NULL,
			mr_id       INTEGER,
			reviewed_user TEXT  NOT NULL,
			reviewed_at DATETIME NOT NULL,
			UNIQUE(client_name, project_ref, mr_iid, reviewed_user)
		)
	`)
	return err
}

func (r *SQLiteRepository) AlreadyReviewed(clientName, projectRef string, mrIID int64, username string) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(1) FROM reviewed_merge_requests
		WHERE client_name = ? AND project_ref = ? AND mr_iid = ? AND reviewed_user = ?`,
		clientName, projectRef, mrIID, username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking reviewed status: %w", err)
	}
	return count > 0, nil
}

func (r *SQLiteRepository) MarkAsReviewed(clientName, projectRef string, mrIID, mrID int64, username string) error {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO reviewed_merge_requests
			(client_name, project_ref, mr_iid, mr_id, reviewed_user, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		clientName, projectRef, mrIID, mrID, username, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("marking as reviewed: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
