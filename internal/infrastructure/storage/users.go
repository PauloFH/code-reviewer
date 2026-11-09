package storage

import (
	"errors"
	"fmt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const userSelect = `SELECT id, username, password, role, created_at, updated_at FROM users`

func scanUser(s scanner) (*domain.User, error) {
	var u domain.User
	if err := s.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

func (r *SQLiteRepository) GetByUsername(username string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(userSelect+` WHERE username=?`, username))
}

func (r *SQLiteRepository) GetByID(id int64) (*domain.User, error) {
	return scanUser(r.db.QueryRow(userSelect+` WHERE id=?`, id))
}

func (r *SQLiteRepository) ListUsers() ([]domain.User, error) {
	rows, err := r.db.Query(userSelect + ` ORDER BY username`)
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
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateUser(u domain.User) (*domain.User, error) {
	t := now()
	res, err := r.db.Exec(`INSERT INTO users(username, password, role, created_at, updated_at) VALUES(?,?,?,?,?)`,
		u.Username, u.Password, u.Role, t, t)
	if isUnique(err) {
		return nil, fmt.Errorf("user %q: %w", u.Username, domain.ErrConflict)
	}
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

// keepsASuper is true for the row unless it is the only super user; with
// one statement per write (and a single connection) the check cannot race.
const keepsASuper = `(role <> 'super' OR (SELECT COUNT(*) FROM users WHERE role = 'super') > 1)`

func (r *SQLiteRepository) UpdateUser(id int64, u domain.User) (*domain.User, error) {
	err := r.userWrite(id, affectedOne(r.db.Exec(
		`UPDATE users SET username=?, password=COALESCE(NULLIF(?, ''), password), role=?, updated_at=?
		 WHERE id=? AND (? = 'super' OR `+keepsASuper+`)`,
		u.Username, u.Password, u.Role, now(), id, u.Role)))
	if isUnique(err) {
		return nil, fmt.Errorf("user %q: %w", u.Username, domain.ErrConflict)
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *SQLiteRepository) DeleteUser(id int64) error {
	return r.userWrite(id, affectedOne(r.db.Exec(`DELETE FROM users WHERE id=? AND `+keepsASuper, id)))
}

// userWrite tells a missing user from one the last-super rule protected.
func (r *SQLiteRepository) userWrite(id int64, err error) error {
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if _, getErr := r.GetByID(id); getErr != nil {
		return getErr
	}
	return domain.ErrLastSuper
}

func (r *SQLiteRepository) CountUsers() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}
