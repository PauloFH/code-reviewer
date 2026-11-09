package storage

import (
	"errors"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestUsersAlwaysKeepASuper(t *testing.T) {
	repo := newTestRepo(t)
	mk := func(name string, role domain.Role) *domain.User {
		u, err := repo.CreateUser(domain.User{Username: name, Password: "h", Role: role})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	root, staff := mk("root", domain.RoleSuper), mk("staff", domain.RolePadrao)

	if _, err := repo.UpdateUser(root.ID, domain.User{Username: "root", Role: domain.RoleGestor}); !errors.Is(err, domain.ErrLastSuper) {
		t.Errorf("demoting the only super: %v, want ErrLastSuper", err)
	}
	if err := repo.DeleteUser(root.ID); !errors.Is(err, domain.ErrLastSuper) {
		t.Errorf("deleting the only super: %v, want ErrLastSuper", err)
	}
	if u, err := repo.UpdateUser(root.ID, domain.User{Username: "root2", Role: domain.RoleSuper}); err != nil || u.Username != "root2" || u.Password != "h" {
		t.Errorf("renaming the only super (password kept): %+v %v", u, err)
	}
	if err := repo.DeleteUser(staff.ID); err != nil {
		t.Errorf("deleting a non-super: %v", err)
	}
	if err := repo.DeleteUser(999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting a missing user: %v, want ErrNotFound", err)
	}

	other := mk("other", domain.RoleSuper)
	if _, err := repo.UpdateUser(root.ID, domain.User{Username: "root2", Role: domain.RolePadrao}); err != nil {
		t.Errorf("demoting one of two supers: %v", err)
	}
	if err := repo.DeleteUser(other.ID); !errors.Is(err, domain.ErrLastSuper) {
		t.Errorf("deleting the remaining super: %v, want ErrLastSuper", err)
	}
}
