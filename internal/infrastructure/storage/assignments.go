package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// assignments implements domain.Assignments over a "<kind>_assignments" table
// whose fk column points into targets. Its SQL is built once from those fixed
// identifiers; every value is a bound parameter.
type assignments struct {
	db                                     *sql.DB
	table                                  string
	listSQL, setSQL, removeSQL, resolveSQL string
}

func newAssignments(db *sql.DB, table, fk, targets string) assignments {
	return assignments{
		db:    db,
		table: table,
		listSQL: `SELECT a.id, a.` + fk + `, t.name, a.scope, a.client_name, a.project_ref
			FROM ` + table + ` a JOIN ` + targets + ` t ON t.id = a.` + fk + `
			ORDER BY a.scope, a.client_name, a.project_ref`,
		setSQL:    `INSERT OR REPLACE INTO ` + table + `(` + fk + `, scope, client_name, project_ref) VALUES(?,?,?,?)`,
		removeSQL: `DELETE FROM ` + table + ` WHERE scope=? AND client_name=? AND project_ref=?`,
		resolveSQL: `SELECT ` + fk + ` FROM ` + table + `
			WHERE (scope='project' AND client_name=? AND project_ref=?)
			   OR (scope='client' AND client_name=? AND project_ref='')
			   OR scope='global'
			ORDER BY CASE scope WHEN 'project' THEN 0 WHEN 'client' THEN 1 ELSE 2 END
			LIMIT 1`,
	}
}

func (a assignments) ListAssignments() ([]domain.Assignment, error) {
	rows, err := a.db.Query(a.listSQL)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", a.table, err)
	}
	defer rows.Close()
	var out []domain.Assignment
	for rows.Next() {
		var x domain.Assignment
		if err := rows.Scan(&x.ID, &x.TargetID, &x.TargetName, &x.Scope, &x.ClientName, &x.ProjectRef); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (a assignments) SetAssignment(x domain.Assignment) error {
	k, err := x.ScopeKey.Normalize()
	if err != nil {
		return err
	}
	_, err = a.db.Exec(a.setSQL, x.TargetID, k.Scope, k.ClientName, k.ProjectRef)
	if isConstraint(err, sqlite3.ErrConstraintForeignKey) {
		return domain.Invalid("target %d does not exist", x.TargetID)
	}
	if err != nil {
		return fmt.Errorf("setting %s: %w", a.table, err)
	}
	return nil
}

func (a assignments) RemoveAssignment(k domain.ScopeKey) error {
	k, err := k.Normalize()
	if err != nil {
		return err
	}
	_, err = a.db.Exec(a.removeSQL, k.Scope, k.ClientName, k.ProjectRef)
	if err != nil {
		return fmt.Errorf("removing %s: %w", a.table, err)
	}
	return nil
}

// resolve returns the target assigned to the most specific matching scope, or 0.
func (a assignments) resolve(clientName, projectRef string) (int64, error) {
	var id int64
	err := a.db.QueryRow(a.resolveSQL, clientName, projectRef, clientName).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("resolving %s: %w", a.table, err)
	}
	return id, nil
}
