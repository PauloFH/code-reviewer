package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// record implements domain.Library for a table with id, a unique name, the
// listed columns and created_at/updated_at.
type record[T any] struct {
	db      *sql.DB
	table   string
	columns []string // writable columns, first is "name", in values() order
	// values returns the writable column values of v.
	values func(v T) ([]any, error)
	// scan reads "id, columns..., created_at, updated_at".
	scan func(s scanner) (T, error)
}

func (r record[T]) selectSQL() string {
	return `SELECT id, ` + strings.Join(r.columns, ", ") + `, created_at, updated_at FROM ` + r.table
}

func (r record[T]) List() ([]T, error) {
	rows, err := r.db.Query(r.selectSQL() + ` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", r.table, err)
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r record[T]) Get(id int64) (*T, error) {
	v, err := r.scan(r.db.QueryRow(r.selectSQL()+` WHERE id=?`, id))
	if err != nil {
		return nil, notFound(err)
	}
	return &v, nil
}

func (r record[T]) Create(v T) (*T, error) {
	vals, err := r.values(v)
	if err != nil {
		return nil, err
	}
	t := now()
	res, err := r.db.Exec(
		`INSERT INTO `+r.table+`(`+strings.Join(r.columns, ", ")+`, created_at, updated_at) VALUES(`+
			strings.Repeat("?, ", len(r.columns))+`?, ?)`,
		append(vals, t, t)...)
	if err != nil {
		return nil, r.writeErr(vals, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.Get(id)
}

func (r record[T]) Update(id int64, v T) (*T, error) {
	vals, err := r.values(v)
	if err != nil {
		return nil, err
	}
	err = affectedOne(r.db.Exec(
		`UPDATE `+r.table+` SET `+strings.Join(r.columns, "=?, ")+`=?, updated_at=? WHERE id=?`,
		append(vals, now(), id)...))
	if err != nil {
		return nil, r.writeErr(vals, err)
	}
	return r.Get(id)
}

func (r record[T]) Delete(id int64) error {
	return affectedOne(r.db.Exec(`DELETE FROM `+r.table+` WHERE id=?`, id))
}

func (r record[T]) writeErr(vals []any, err error) error {
	if isUnique(err) {
		return fmt.Errorf("%v: %w", vals[0], domain.ErrConflict)
	}
	return err
}

// scopedLibrary adds scope assignments and resolution to a record.
type scopedLibrary[T any] struct {
	record[T]
	assignments
}

func (s scopedLibrary[T]) Resolve(clientName, projectRef string) (*T, error) {
	id, err := s.resolve(clientName, projectRef)
	if err != nil || id == 0 {
		return nil, err
	}
	return s.Get(id)
}
