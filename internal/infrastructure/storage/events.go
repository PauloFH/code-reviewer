package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const defaultEventsPageSize = 50

func (r *SQLiteRepository) SaveEvent(e domain.ReviewEvent) error {
	if e.ReviewedAt.IsZero() {
		e.ReviewedAt = now()
	}
	var errMsg any
	if e.ErrorMessage != "" {
		errMsg = e.ErrorMessage
	}
	_, err := r.db.Exec(`
		INSERT INTO review_events
			(client_name,project_ref,mr_iid,mr_id,mr_title,mr_url,author_username,
			 prompt_tokens,completion_tokens,total_tokens,cost_usd,duration_ms,
			 status,error_message,reviewed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ClientName, e.ProjectRef, e.MRIID, e.MRID, e.MRTitle, e.MRURL, e.AuthorUsername,
		e.PromptTokens, e.CompletionTokens, e.TotalTokens, e.CostUSD, e.DurationMS,
		e.Status, errMsg, e.ReviewedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("saving review event: %w", err)
	}
	return nil
}

// PurgeEventsBefore apaga o histórico de revisões (autor, título e URL do MR) mais antigo que cutoff
// e esvazia o texto guardado das revisões antigas, cumprindo o prazo da política de privacidade.
// A claim em si fica: apagá-la permitiria revisar o mesmo MR de novo.
func (r *SQLiteRepository) PurgeEventsBefore(cutoff time.Time) (int64, error) {
	if _, err := r.db.Exec(`UPDATE reviewed_merge_requests SET review_body='' WHERE reviewed_at < ? AND review_body <> ''`, cutoff.UTC()); err != nil {
		return 0, fmt.Errorf("clearing old review bodies: %w", err)
	}
	res, err := r.db.Exec(`DELETE FROM review_events WHERE reviewed_at < ?`, cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("purging review events: %w", err)
	}
	return res.RowsAffected()
}

func (r *SQLiteRepository) ListEvents(f domain.ReviewEventFilter) ([]domain.ReviewEvent, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = defaultEventsPageSize
	}
	where, args := eventWhere(f)
	rows, err := r.db.Query(
		`SELECT id,client_name,project_ref,mr_iid,mr_id,mr_title,mr_url,author_username,
		        prompt_tokens,completion_tokens,total_tokens,cost_usd,duration_ms,
		        status,COALESCE(error_message,''),reviewed_at
		 FROM review_events `+where+` ORDER BY reviewed_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, f.Offset)...,
	)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewEvent
	for rows.Next() {
		var e domain.ReviewEvent
		if err := rows.Scan(
			&e.ID, &e.ClientName, &e.ProjectRef, &e.MRIID, &e.MRID,
			&e.MRTitle, &e.MRURL, &e.AuthorUsername,
			&e.PromptTokens, &e.CompletionTokens, &e.TotalTokens,
			&e.CostUSD, &e.DurationMS, &e.Status, &e.ErrorMessage, &e.ReviewedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CountEvents(f domain.ReviewEventFilter) (int64, error) {
	where, args := eventWhere(f)
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(1) FROM review_events `+where, args...).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) GetStats(f domain.ReviewEventFilter) (*domain.ReviewStats, error) {
	f.Status = domain.ReviewStatusSuccess
	where, args := eventWhere(f)
	stats := &domain.ReviewStats{}

	if err := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0) FROM review_events `+where, args...).
		Scan(&stats.TotalReviews, &stats.TotalTokens, &stats.TotalCostUSD); err != nil {
		return nil, fmt.Errorf("stats totals: %w", err)
	}

	if err := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(cost_usd),0) FROM review_events `+where+` AND reviewed_at >= ?`,
		append(args, now().AddDate(0, 0, -30))...).
		Scan(&stats.ReviewsLast30d, &stats.CostLast30d); err != nil {
		return nil, fmt.Errorf("stats last 30 days: %w", err)
	}

	rows, err := r.db.Query(`SELECT client_name, COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost_usd),0)
		FROM review_events `+where+` GROUP BY client_name ORDER BY client_name`, args...)
	if err != nil {
		return nil, fmt.Errorf("stats by client: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cs domain.ClientStat
		if err := rows.Scan(&cs.ClientName, &cs.ReviewCount, &cs.TotalTokens, &cs.TotalCostUSD); err != nil {
			return nil, err
		}
		stats.ByClient = append(stats.ByClient, cs)
	}
	return stats, rows.Err()
}

func (r *SQLiteRepository) GetDailyCosts(clientName string, days int) ([]domain.DailyCostPoint, error) {
	f := domain.ReviewEventFilter{ClientName: clientName, Status: domain.ReviewStatusSuccess, Since: now().AddDate(0, 0, -days), Until: now()}
	where, args := eventWhere(f)
	rows, err := r.db.Query(`SELECT date(reviewed_at) AS day, COALESCE(SUM(cost_usd),0), COUNT(*)
		FROM review_events `+where+` GROUP BY day ORDER BY day ASC`, args...)
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

// eventWhere builds the WHERE clause shared by every event query. Every value
// is bound as a parameter; times are bound as time.Time so they compare in the
// same text format the driver stores.
func eventWhere(f domain.ReviewEventFilter) (string, []any) {
	clauses := []string{"1=1"}
	var args []any
	add := func(clause string, vals ...any) {
		clauses = append(clauses, clause)
		args = append(args, vals...)
	}
	if f.ClientName != "" {
		add("client_name=?", f.ClientName)
	}
	if f.Status != "" {
		add("status=?", f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		add("(mr_title LIKE ? OR author_username LIKE ? OR project_ref LIKE ?)", like, like, like)
	}
	if !f.Since.IsZero() {
		add("reviewed_at >= ?", f.Since.UTC())
	}
	if !f.Until.IsZero() {
		add("reviewed_at <= ?", f.Until.UTC())
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
