package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// reviewProfileColumns is the canonical column list for review_profiles selects.
const reviewProfileColumns = `id, name, include_related, agentic, execution,
	max_iterations, created_at, updated_at`

func (r *SQLiteRepository) ListProfiles() ([]domain.ReviewProfile, error) {
	rows, err := r.db.Query(`SELECT ` + reviewProfileColumns + ` FROM review_profiles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing review profiles: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewProfile
	for rows.Next() {
		p, err := scanReviewProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetProfile(id int64) (*domain.ReviewProfile, error) {
	row := r.db.QueryRow(`SELECT `+reviewProfileColumns+` FROM review_profiles WHERE id=?`, id)
	p, err := scanReviewProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLiteRepository) CreateProfile(p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO review_profiles(
		name, include_related, agentic, execution, max_iterations, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?)`,
		p.Name, p.IncludeRelated, p.Agentic, p.Execution, p.MaxIterations, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("review profile %q already exists", p.Name)
		}
		return nil, fmt.Errorf("creating review profile: %w", err)
	}
	id, _ := res.LastInsertId()
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) UpdateProfile(id int64, p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(`UPDATE review_profiles SET
		name=?, include_related=?, agentic=?, execution=?, max_iterations=?, updated_at=?
		WHERE id=?`,
		p.Name, p.IncludeRelated, p.Agentic, p.Execution, p.MaxIterations, now, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("review profile %q already exists", p.Name)
		}
		return nil, fmt.Errorf("updating review profile: %w", err)
	}
	p.ID = id
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) DeleteProfile(id int64) error {
	if _, err := r.db.Exec(`DELETE FROM review_profiles WHERE id=?`, id); err != nil {
		return fmt.Errorf("deleting review profile: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SetProfileAssignment(a domain.ReviewProfileAssignment) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO review_profile_assignments(profile_id, scope, client_name, project_ref) VALUES(?,?,?,?)`,
		a.ProfileID, a.Scope, a.ClientName, a.ProjectRef)
	if err != nil {
		return fmt.Errorf("setting review profile assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RemoveProfileAssignment(scope, clientName, projectRef string) error {
	_, err := r.db.Exec(`DELETE FROM review_profile_assignments WHERE scope=? AND client_name=? AND project_ref=?`,
		scope, clientName, projectRef)
	if err != nil {
		return fmt.Errorf("removing review profile assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListProfileAssignments() ([]domain.ReviewProfileAssignment, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.profile_id, p.name, a.scope, a.client_name, a.project_ref
		FROM review_profile_assignments a
		JOIN review_profiles p ON p.id = a.profile_id
		ORDER BY a.scope, a.client_name, a.project_ref`)
	if err != nil {
		return nil, fmt.Errorf("listing review profile assignments: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewProfileAssignment
	for rows.Next() {
		var a domain.ReviewProfileAssignment
		if err := rows.Scan(&a.ID, &a.ProfileID, &a.ProfileName, &a.Scope, &a.ClientName, &a.ProjectRef); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// reviewProfileColumnsAliased is the column list with table alias for JOIN queries.
const reviewProfileColumnsAliased = `p.id, p.name, p.include_related, p.agentic, p.execution,
	p.max_iterations, p.created_at, p.updated_at`

// ResolveProfile returns the review profile assigned to the most specific scope
// that matches: project → client → global. Returns nil when nothing is assigned.
func (r *SQLiteRepository) ResolveProfile(clientName, projectRef string) (*domain.ReviewProfile, error) {
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT ` + reviewProfileColumnsAliased + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='project' AND a.client_name=? AND a.project_ref=?`, []any{clientName, projectRef}},
		{`SELECT ` + reviewProfileColumnsAliased + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='client' AND a.client_name=? AND a.project_ref=''`, []any{clientName}},
		{`SELECT ` + reviewProfileColumnsAliased + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='global'`, nil},
	}

	for _, q := range queries {
		row := r.db.QueryRow(q.sql, q.args...)
		p, err := scanReviewProfile(row)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, nil
}

func scanReviewProfile(s scanner) (domain.ReviewProfile, error) {
	var p domain.ReviewProfile
	var includeRelated, agentic, execution int
	var createdAt, updatedAt string

	err := s.Scan(&p.ID, &p.Name, &includeRelated, &agentic, &execution,
		&p.MaxIterations, &createdAt, &updatedAt)
	if err != nil {
		return p, err
	}

	p.IncludeRelated = includeRelated != 0
	p.Agentic = agentic != 0
	p.Execution = execution != 0
	p.CreatedAt = parseDBTime(createdAt)
	p.UpdatedAt = parseDBTime(updatedAt)
	return p, nil
}
