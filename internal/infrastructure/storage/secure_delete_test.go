package storage

import "testing"

func TestSecureDeleteIsOn(t *testing.T) {
	repo := newTestRepo(t)
	var on int
	if err := repo.db.QueryRow(`PRAGMA secure_delete`).Scan(&on); err != nil {
		t.Fatalf("PRAGMA secure_delete: %v", err)
	}
	if on != 1 {
		t.Fatalf("secure_delete = %d, want 1 (apagado precisa sumir do arquivo)", on)
	}
}
