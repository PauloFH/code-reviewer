package usecase

import (
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestFilterInlineComments_KeepsOnlyAddedLines(t *testing.T) {
	diff := `--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
-func old() {}
+func run() {}
 func keep() {}
`

	got := filterInlineComments([]domain.InlineComment{
		{Path: "main.go", Line: 2, Body: "ok"},
		{Path: "main.go", Line: 1, Body: "context line should be ignored"},
		{Path: "main.go", Line: 99, Body: "missing"},
		{Path: "other.go", Line: 2, Body: "wrong file"},
	}, diff)

	if len(got) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(got))
	}
	if got[0].Path != "main.go" || got[0].Line != 2 || got[0].Body != "ok" {
		t.Fatalf("unexpected comment: %+v", got[0])
	}
}

func TestFilterInlineComments_NormalizesPathsAndDeduplicates(t *testing.T) {
	diff := `--- a/src/app.go
+++ b/src/app.go
@@ -10,0 +11,2 @@
+func a() {}
+func b() {}
`

	got := filterInlineComments([]domain.InlineComment{
		{Path: "b/src/app.go", Line: 11, Body: "Rever nome."},
		{Path: "src/app.go", Line: 11, Body: "Rever nome."},
		{Path: "a/src/app.go", Line: 12, Body: "  Adicionar teste.  "},
	}, diff)

	if len(got) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(got))
	}
	if got[0].Path != "src/app.go" {
		t.Fatalf("expected normalized path, got %q", got[0].Path)
	}
	if got[1].Body != "Adicionar teste." {
		t.Fatalf("expected trimmed body, got %q", got[1].Body)
	}
}
