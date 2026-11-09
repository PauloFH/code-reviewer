package storage

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

// testEncKey is base64 of "0123456789abcdef0123456789abcdef" (32 bytes).
const testEncKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func openAt(t *testing.T, path, key string) (*SQLiteRepository, error) {
	t.Helper()
	repo, err := Open(path, key, discard)
	if err == nil {
		t.Cleanup(func() { _ = repo.Close() })
	}
	return repo, err
}

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	repo, err := openAt(t, filepath.Join(t.TempDir(), "test.db"), testEncKey)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return repo
}
