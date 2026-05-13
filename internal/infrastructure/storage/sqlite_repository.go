package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// SQLiteRepository implements domain.ReviewRepository, domain.ClientRepository
// and domain.ReviewEventRepository using a single SQLite database.
type SQLiteRepository struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at path and runs all pending migrations.
func New(path string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite allows only one writer at a time

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Versioned migrations
// ──────────────────────────────────────────────────────────────────────────────

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return err
	}

	type migration struct {
		version int
		sql     string
	}

	pending := []migration{
		{1, migration001},
		{2, migration002},
	}

	for _, m := range pending {
		var count int
		db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version=?`, m.version).Scan(&count)
		if count > 0 {
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration %d: %w", m.version, err)
		}
		if _, err := db.Exec(
			`INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`,
			m.version, time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// domain.ReviewRepository
// ──────────────────────────────────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────────────────────────────────
// domain.ClientRepository
// ──────────────────────────────────────────────────────────────────────────────

func (r *SQLiteRepository) CountClients() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM clients`).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) ListClients() ([]domain.Client, error) {
	rows, err := r.db.Query(`
		SELECT id,name,platform,base_url,token,webhook_secret,username,created_at,updated_at
		FROM clients ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Client
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetClient(name string) (*domain.Client, error) {
	row := r.db.QueryRow(`
		SELECT id,name,platform,base_url,token,webhook_secret,username,created_at,updated_at
		FROM clients WHERE name=?`, name)
	c, err := scanClient(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) CreateClient(c domain.Client) (*domain.Client, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`
		INSERT INTO clients(name,platform,base_url,token,webhook_secret,username,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		c.Name, string(c.Platform), c.BaseURL, c.Token, c.WebhookSecret, c.Username, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("client %q already exists", c.Name)
		}
		return nil, fmt.Errorf("creating client: %w", err)
	}
	id, _ := res.LastInsertId()
	c.ID = id
	c.CreatedAt = now
	c.UpdatedAt = now
	return &c, nil
}

func (r *SQLiteRepository) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(`
		UPDATE clients SET name=?,platform=?,base_url=?,token=?,webhook_secret=?,username=?,updated_at=?
		WHERE name=?`,
		c.Name, string(c.Platform), c.BaseURL, c.Token, c.WebhookSecret, c.Username, now, oldName,
	)
	if err != nil {
		return nil, fmt.Errorf("updating client: %w", err)
	}
	c.UpdatedAt = now
	return &c, nil
}

func (r *SQLiteRepository) DeleteClient(name string) error {
	_, err := r.db.Exec(`DELETE FROM clients WHERE name=?`, name)
	if err != nil {
		return fmt.Errorf("deleting client: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SeedFromConfigs(clients []domain.Client) error {
	now := time.Now().UTC()
	for _, c := range clients {
		_, err := r.db.Exec(`
			INSERT OR IGNORE INTO clients
				(name,platform,base_url,token,webhook_secret,username,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?)`,
			c.Name, string(c.Platform), c.BaseURL, c.Token, c.WebhookSecret, c.Username, now, now,
		)
		if err != nil {
			return fmt.Errorf("seeding client %q: %w", c.Name, err)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// domain.ReviewEventRepository
// ──────────────────────────────────────────────────────────────────────────────

func (r *SQLiteRepository) SaveEvent(e domain.ReviewEvent) error {
	if e.ReviewedAt.IsZero() {
		e.ReviewedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(`
		INSERT INTO review_events
			(client_name,project_ref,mr_iid,mr_id,mr_title,mr_url,author_username,
			 prompt_tokens,completion_tokens,total_tokens,cost_usd,duration_ms,
			 status,error_message,reviewed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ClientName, e.ProjectRef, e.MRIID, e.MRID, e.MRTitle, e.MRURL, e.AuthorUsername,
		e.PromptTokens, e.CompletionTokens, e.TotalTokens, e.CostUSD, e.DurationMS,
		string(e.Status), nullString(e.ErrorMessage), e.ReviewedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("saving review event: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListEvents(f domain.ReviewEventFilter) ([]domain.ReviewEvent, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	args := []any{}
	where := buildEventWhere(f, &args)

	rows, err := r.db.Query(
		`SELECT id,client_name,project_ref,mr_iid,mr_id,mr_title,mr_url,author_username,
		        prompt_tokens,completion_tokens,total_tokens,cost_usd,duration_ms,
		        status,COALESCE(error_message,''),reviewed_at
		 FROM review_events `+where+
			` ORDER BY reviewed_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, f.Offset)...,
	)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewEvent
	for rows.Next() {
		var e domain.ReviewEvent
		var reviewedAt string
		if err := rows.Scan(
			&e.ID, &e.ClientName, &e.ProjectRef, &e.MRIID, &e.MRID,
			&e.MRTitle, &e.MRURL, &e.AuthorUsername,
			&e.PromptTokens, &e.CompletionTokens, &e.TotalTokens,
			&e.CostUSD, &e.DurationMS, &e.Status, &e.ErrorMessage, &reviewedAt,
		); err != nil {
			return nil, err
		}
		e.ReviewedAt, _ = time.Parse("2006-01-02T15:04:05Z", reviewedAt)
		if e.ReviewedAt.IsZero() {
			e.ReviewedAt, _ = time.Parse("2006-01-02 15:04:05", reviewedAt)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CountEvents(f domain.ReviewEventFilter) (int64, error) {
	args := []any{}
	where := buildEventWhere(f, &args)
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(1) FROM review_events `+where, args...).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) GetStats(clientName string) (*domain.ReviewStats, error) {
	stats := &domain.ReviewStats{}

	globalQ := `SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0) FROM review_events WHERE status='success'`
	r.db.QueryRow(globalQ).Scan(&stats.TotalReviews, &stats.TotalTokens, &stats.TotalCostUSD)

	last30Q := `SELECT COUNT(*), COALESCE(SUM(cost_usd),0) FROM review_events WHERE status='success' AND reviewed_at >= datetime('now','-30 days')`
	r.db.QueryRow(last30Q).Scan(&stats.ReviewsLast30d, &stats.CostLast30d)

	byClientQ := `
		SELECT client_name, COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0)
		FROM review_events WHERE status='success'`
	byClientArgs := []any{}
	if clientName != "" {
		byClientQ += ` AND client_name = ?`
		byClientArgs = append(byClientArgs, clientName)
	}
	byClientQ += ` GROUP BY client_name ORDER BY client_name`

	rows, err := r.db.Query(byClientQ, byClientArgs...)
	if err != nil {
		return nil, fmt.Errorf("stats by client: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cs domain.ClientStat
		rows.Scan(&cs.ClientName, &cs.ReviewCount, &cs.TotalTokens, &cs.TotalCostUSD)
		stats.ByClient = append(stats.ByClient, cs)
	}
	return stats, rows.Err()
}

func (r *SQLiteRepository) GetDailyCosts(clientName string, days int) ([]domain.DailyCostPoint, error) {
	if days <= 0 {
		days = 30
	}
	q := `
		SELECT date(reviewed_at) as day,
		       COALESCE(SUM(cost_usd),0),
		       COUNT(*)
		FROM review_events
		WHERE status='success'
		  AND reviewed_at >= datetime('now', ? )
		  AND reviewed_at <= datetime('now')`
	args := []any{fmt.Sprintf("-%d days", days)}
	if clientName != "" {
		q += ` AND client_name = ?`
		args = append(args, clientName)
	}
	q += ` GROUP BY day ORDER BY day ASC`

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("daily costs: %w", err)
	}
	defer rows.Close()

	var out []domain.DailyCostPoint
	for rows.Next() {
		var p domain.DailyCostPoint
		if err := rows.Scan(&p.Date, &p.CostUSD, &p.Reviews); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ──────────────────────────────────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanClient(s scanner) (domain.Client, error) {
	var c domain.Client
	var platform, createdAt, updatedAt string
	err := s.Scan(&c.ID, &c.Name, &platform, &c.BaseURL, &c.Token, &c.WebhookSecret, &c.Username, &createdAt, &updatedAt)
	if err != nil {
		return c, err
	}
	c.Platform = domain.PlatformType(platform)
	c.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
	if c.CreatedAt.IsZero() {
		c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	c.UpdatedAt, _ = time.Parse("2006-01-02T15:04:05Z", updatedAt)
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}
	return c, nil
}

func buildEventWhere(f domain.ReviewEventFilter, args *[]any) string {
	clauses := []string{}
	if f.ClientName != "" {
		clauses = append(clauses, "client_name=?")
		*args = append(*args, f.ClientName)
	}
	if !f.Since.IsZero() {
		clauses = append(clauses, "reviewed_at >= ?")
		*args = append(*args, f.Since.UTC().Format(time.RFC3339))
	}
	if !f.Until.IsZero() {
		clauses = append(clauses, "reviewed_at <= ?")
		*args = append(*args, f.Until.UTC().Format(time.RFC3339))
	}
	if len(clauses) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
