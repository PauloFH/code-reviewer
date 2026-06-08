package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// SQLiteRepository implements domain.ReviewRepository, domain.ClientRepository,
// domain.ReviewEventRepository, domain.PromptRepository and
// domain.AIConfigRepository using a single SQLite database.
type SQLiteRepository struct {
	db  *sql.DB
	enc *encryptor
}

// UseEncryptionKey configures AES-256-GCM encryption for AI credentials using a
// base64-encoded 32-byte key. Must be called before any AIConfig operation that
// touches secrets.
func (r *SQLiteRepository) UseEncryptionKey(base64Key string) error {
	enc, err := newEncryptor(base64Key)
	if err != nil {
		return err
	}
	r.enc = enc
	return nil
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
		{3, migration003},
		{4, migration004},
		{5, migration005},
		{6, migration006},
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

func (r *SQLiteRepository) GetStats(clientName, sinceStr, untilStr string) (*domain.ReviewStats, error) {
	stats := &domain.ReviewStats{}

	sinceClause := ""
	untilClause := ""
	if sinceStr != "" {
		sinceClause = " AND reviewed_at >= '" + sinceStr + "'"
	}
	if untilStr != "" {
		untilClause = " AND reviewed_at <= '" + untilStr + "'"
	}

	globalQ := fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0) FROM review_events WHERE status='success'%s%s`,
		sinceClause, untilClause)
	r.db.QueryRow(globalQ).Scan(&stats.TotalReviews, &stats.TotalTokens, &stats.TotalCostUSD)

	last30Q := fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(cost_usd),0) FROM review_events WHERE status='success' AND reviewed_at >= datetime('now','-30 days')%s%s`,
		sinceClause, untilClause)
	r.db.QueryRow(last30Q).Scan(&stats.ReviewsLast30d, &stats.CostLast30d)

	byClientQ := fmt.Sprintf(`SELECT client_name, COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0)
		FROM review_events WHERE status='success'%s%s`, sinceClause, untilClause)
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
	if f.Search != "" {
		clauses = append(clauses, "(mr_title LIKE ? OR author_username LIKE ? OR project_ref LIKE ?)")
		like := "%" + f.Search + "%"
		*args = append(*args, like, like, like)
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

// ──────────────────────────────────────────────────────────────────────────────
// domain.ReviewRepository — ClearReview
// ──────────────────────────────────────────────────────────────────────────────

func (r *SQLiteRepository) ClearReview(clientName, projectRef string, mrIID int64) error {
	_, err := r.db.Exec(
		`DELETE FROM reviewed_merge_requests WHERE client_name=? AND project_ref=? AND mr_iid=?`,
		clientName, projectRef, mrIID,
	)
	if err != nil {
		return fmt.Errorf("clearing review: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// domain.UserRepository
// ──────────────────────────────────────────────────────────────────────────────

func (r *SQLiteRepository) GetByUsername(username string) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password, role, created_at, updated_at FROM users WHERE username=?`,
		username,
	)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteRepository) GetByID(id int64) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password, role, created_at, updated_at FROM users WHERE id=?`,
		id,
	)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteRepository) ListUsers() ([]domain.User, error) {
	rows, err := r.db.Query(`SELECT id, username, password, role, created_at, updated_at FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateUser(u domain.User) (*domain.User, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`INSERT INTO users(username, password, role, created_at, updated_at) VALUES(?,?,?,?,?)`,
		u.Username, u.Password, string(u.Role), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("user %q already exists", u.Username)
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}
	id, _ := res.LastInsertId()
	u.ID = id
	u.CreatedAt = now
	u.UpdatedAt = now
	return &u, nil
}

func (r *SQLiteRepository) UpdateUser(id int64, u domain.User) (*domain.User, error) {
	now := time.Now().UTC()
	if u.Password != "" {
		_, err := r.db.Exec(
			`UPDATE users SET username=?, password=?, role=?, updated_at=? WHERE id=?`,
			u.Username, u.Password, string(u.Role), now, id,
		)
		if err != nil {
			return nil, fmt.Errorf("updating user: %w", err)
		}
	} else {
		_, err := r.db.Exec(
			`UPDATE users SET username=?, role=?, updated_at=? WHERE id=?`,
			u.Username, string(u.Role), now, id,
		)
		if err != nil {
			return nil, fmt.Errorf("updating user: %w", err)
		}
	}
	u.ID = id
	u.UpdatedAt = now
	return &u, nil
}

func (r *SQLiteRepository) DeleteUser(id int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) CountUsers() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) UserExists(username string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM users WHERE username=?`, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// domain.PromptRepository
// ──────────────────────────────────────────────────────────────────────────────

func (r *SQLiteRepository) ListPrompts() ([]domain.Prompt, error) {
	rows, err := r.db.Query(`SELECT id, name, system_prompt, output_template, created_at, updated_at FROM prompts ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing prompts: %w", err)
	}
	defer rows.Close()

	var out []domain.Prompt
	for rows.Next() {
		p, err := scanPrompt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetPrompt(id int64) (*domain.Prompt, error) {
	row := r.db.QueryRow(`SELECT id, name, system_prompt, output_template, created_at, updated_at FROM prompts WHERE id=?`, id)
	p, err := scanPrompt(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLiteRepository) CreatePrompt(p domain.Prompt) (*domain.Prompt, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO prompts(name, system_prompt, output_template, created_at, updated_at) VALUES(?,?,?,?,?)`,
		p.Name, p.SystemPrompt, p.OutputTemplate, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("prompt %q already exists", p.Name)
		}
		return nil, fmt.Errorf("creating prompt: %w", err)
	}
	id, _ := res.LastInsertId()
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) UpdatePrompt(id int64, p domain.Prompt) (*domain.Prompt, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(`UPDATE prompts SET name=?, system_prompt=?, output_template=?, updated_at=? WHERE id=?`,
		p.Name, p.SystemPrompt, p.OutputTemplate, now, id)
	if err != nil {
		return nil, fmt.Errorf("updating prompt: %w", err)
	}
	p.ID = id
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) DeletePrompt(id int64) error {
	_, err := r.db.Exec(`DELETE FROM prompts WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting prompt: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SetAssignment(a domain.PromptAssignment) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO prompt_assignments(prompt_id, scope, client_name, project_ref) VALUES(?,?,?,?)`,
		a.PromptID, a.Scope, a.ClientName, a.ProjectRef)
	if err != nil {
		return fmt.Errorf("setting prompt assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RemoveAssignment(scope, clientName, projectRef string) error {
	_, err := r.db.Exec(`DELETE FROM prompt_assignments WHERE scope=? AND client_name=? AND project_ref=?`,
		scope, clientName, projectRef)
	if err != nil {
		return fmt.Errorf("removing prompt assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListAssignments() ([]domain.PromptAssignment, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.prompt_id, p.name, a.scope, a.client_name, a.project_ref
		FROM prompt_assignments a
		JOIN prompts p ON p.id = a.prompt_id
		ORDER BY a.scope, a.client_name, a.project_ref`)
	if err != nil {
		return nil, fmt.Errorf("listing assignments: %w", err)
	}
	defer rows.Close()

	var out []domain.PromptAssignment
	for rows.Next() {
		var a domain.PromptAssignment
		if err := rows.Scan(&a.ID, &a.PromptID, &a.PromptName, &a.Scope, &a.ClientName, &a.ProjectRef); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) ResolvePrompt(clientName, projectRef string) (*domain.Prompt, error) {
	// project-level
	row := r.db.QueryRow(`
		SELECT p.id, p.name, p.system_prompt, p.output_template, p.created_at, p.updated_at
		FROM prompts p JOIN prompt_assignments a ON a.prompt_id = p.id
		WHERE a.scope='project' AND a.client_name=? AND a.project_ref=?`,
		clientName, projectRef)
	p, err := scanPrompt(row)
	if err == nil {
		return &p, nil
	}

	// client-level
	row = r.db.QueryRow(`
		SELECT p.id, p.name, p.system_prompt, p.output_template, p.created_at, p.updated_at
		FROM prompts p JOIN prompt_assignments a ON a.prompt_id = p.id
		WHERE a.scope='client' AND a.client_name=? AND a.project_ref=''`,
		clientName)
	p, err = scanPrompt(row)
	if err == nil {
		return &p, nil
	}

	// global
	row = r.db.QueryRow(`
		SELECT p.id, p.name, p.system_prompt, p.output_template, p.created_at, p.updated_at
		FROM prompts p JOIN prompt_assignments a ON a.prompt_id = p.id
		WHERE a.scope='global'`)
	p, err = scanPrompt(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPrompt(s scanner) (domain.Prompt, error) {
	var p domain.Prompt
	var createdAt, updatedAt string
	err := s.Scan(&p.ID, &p.Name, &p.SystemPrompt, &p.OutputTemplate, &createdAt, &updatedAt)
	if err != nil {
		return p, err
	}
	p.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
	if p.CreatedAt.IsZero() {
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	p.UpdatedAt, _ = time.Parse("2006-01-02T15:04:05Z", updatedAt)
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}
	return p, nil
}

func scanUser(s scanner) (domain.User, error) {
	var u domain.User
	var role, createdAt, updatedAt string
	err := s.Scan(&u.ID, &u.Username, &u.Password, &role, &createdAt, &updatedAt)
	if err != nil {
		return u, err
	}
	u.Role = domain.Role(role)
	u.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
	if u.CreatedAt.IsZero() {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	u.UpdatedAt, _ = time.Parse("2006-01-02T15:04:05Z", updatedAt)
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	}
	return u, nil
}
