package gitfetch

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\nfunc Run() {}\n")
	mustWrite(t, filepath.Join(root, "internal", "svc.go"), "package internal\n// TODO: fix\nfunc Do() {}\n")
	return root
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

func TestWorkspaceReadFile(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	b, err := ws.ReadFile("main.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) == "" || !contains(string(b), "func Run()") {
		t.Fatalf("unexpected content: %q", b)
	}
}

func TestWorkspaceListDir(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	entries, err := ws.ListDir(".")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	if !hasAll(names, "main.go", "internal") {
		t.Fatalf("ListDir = %v, want main.go and internal", names)
	}
}

func TestWorkspaceGrep(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	hits, err := ws.Grep("TODO", "")
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(hits) != 1 || hits[0].Path != filepath.FromSlash("internal/svc.go") || hits[0].Line != 2 {
		t.Fatalf("Grep hits = %+v, want internal/svc.go:2", hits)
	}
}

func TestWorkspaceGrepPathGlob(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	hits, err := ws.Grep("func", "*.go") // only top-level *.go
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	for _, h := range hits {
		if h.Path != "main.go" {
			t.Fatalf("glob *.go matched %q", h.Path)
		}
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit in main.go")
	}
}

func TestWorkspaceRejectsTraversal(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	if _, err := ws.ReadFile("../../etc/passwd"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

// tiny helpers (avoid extra imports in the test)
func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func hasAll(set []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, s := range set {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
