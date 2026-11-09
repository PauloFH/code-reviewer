package domain

import (
	"errors"
	"time"
)

// ErrLastSuper refuses a change that would leave no super user to administer the panel.
var ErrLastSuper = errors.New("cannot remove the last super user")

type Role string

const (
	RoleSuper  Role = "super"
	RoleGestor Role = "gestor"
	RolePadrao Role = "padrao"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks a user as submitted; Password is the plaintext and may be
// empty only when updating (keeps the stored one).
func (u User) Validate(creating bool) error {
	switch {
	case u.Username == "":
		return Invalid("username is required")
	case creating && u.Password == "":
		return Invalid("password is required")
	case u.Role != RoleSuper && u.Role != RoleGestor && u.Role != RolePadrao:
		return Invalid("role must be super, gestor, or padrao")
	}
	return nil
}
