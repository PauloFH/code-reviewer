package gitfetch

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// openTree builds dir/repo with a few files and a secret beside it in dir.
func openTree(t *testing.T) *localWorkspace {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "repo")
	mustWrite(t, filepath.Join(root, "main.go"), "package main\nfunc Run() {}\n")
	mustWrite(t, filepath.Join(root, "internal", "svc.go"), "package internal\n// TODO: fix\nfunc Do() {}\n")
	mustWrite(t, filepath.Join(root, ".git", "config"), "[remote]\nurl = https://oauth2:TOKEN@host/x.git\n")
	mustWrite(t, filepath.Join(dir, "secret.txt"), "TODO steal me\n")
	ws, err := openLocalWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ws.close() })
	return ws
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceReadsListsAndGreps(t *testing.T) {
	ws := openTree(t)
	if b, err := ws.ReadFile("main.go"); err != nil || !strings.Contains(string(b), "func Run()") {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
	entries, err := ws.ListDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"internal", "main.go"}) {
		t.Fatalf("ListDir = %v, want internal and main.go (no .git)", names)
	}
	hits, err := ws.Grep("TODO", "")
	if err != nil || len(hits) != 1 || hits[0].Path != "internal/svc.go" || hits[0].Line != 2 {
		t.Fatalf("Grep = %+v, %v; want only internal/svc.go:2", hits, err)
	}
	hits, _ = ws.Grep("func", "*.go")
	if len(hits) != 1 || hits[0].Path != "main.go" {
		t.Fatalf("glob *.go matched %+v, want only main.go", hits)
	}
}

// The tree comes from the MR author, so nothing they commit may reach files
// outside it or the clone's own credentials.
func TestWorkspaceStaysInsideTheTree(t *testing.T) {
	ws := openTree(t)
	if err := os.Symlink(filepath.Join(ws.Root(), "..", "secret.txt"), filepath.Join(ws.Root(), "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..", filepath.Join(ws.Root(), "up")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"../secret.txt", "link", "up/secret.txt", ".git/config", "/../secret.txt"} {
		if b, err := ws.ReadFile(p); err == nil {
			t.Errorf("ReadFile(%q) = %q, want error", p, b)
		}
	}
	if _, err := ws.ListDir("up"); err == nil {
		t.Error("ListDir through a symlink to the parent succeeded")
	}
	hits, _ := ws.Grep("steal|TOKEN", "")
	if len(hits) != 0 {
		t.Errorf("Grep reached outside the tree or into .git: %+v", hits)
	}
}
