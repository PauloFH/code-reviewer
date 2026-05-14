package domain

import "time"

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
