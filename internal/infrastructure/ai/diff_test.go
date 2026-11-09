package ai

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

var sampleDiff = `--- a/src/main.go
+++ b/src/main.go
@@ -1,2 +1,3 @@
 package main
+// added comment
 func main() {}
--- a/package-lock.json
+++ b/package-lock.json
@@ -1 +1 @@
-  "version": "1.0.0"
+  "version": "1.0.1"
--- a/yarn.lock
+++ b/yarn.lock
@@ -0,0 +1 @@
+lodash@4.17.21
--- a/go.sum
+++ b/go.sum
@@ -0,0 +1 @@
+github.com/foo v1.0.0 h1:abc
--- a/dist/bundle.min.js
+++ b/dist/bundle.min.js
@@ -0,0 +1 @@
+!function(){}()
--- a/src/app.ts
+++ b/src/app.ts
@@ -1 +1,2 @@
 export const app = {}
+export const version = "2"
`

func TestFilterDiff_KeepsOnlyRelevantFiles(t *testing.T) {
	want := `--- a/src/main.go
+++ b/src/main.go
@@ -1,2 +1,3 @@
 package main
+// added comment
 func main() {}
--- a/src/app.ts
+++ b/src/app.ts
@@ -1 +1,2 @@
 export const app = {}
+export const version = "2"`
	if got := filterDiff(sampleDiff); got != want {
		t.Fatalf("filterDiff:\n%s\nwant:\n%s", got, want)
	}
}

func TestFilterDiff_RemovedSQLCommentIsNotAFileHeader(t *testing.T) {
	// "-- vendor/x" removed reads "--- vendor/x": it must not switch files.
	diff := "--- a/db/schema.sql\n+++ b/db/schema.sql\n@@ -1,2 +1,1 @@\n--- vendor/x\n select 1;\n"
	if got := filterDiff(diff); !strings.Contains(got, "select 1;") {
		t.Fatalf("removed SQL comment was read as a header and dropped the file:\n%s", got)
	}
}

func TestReviewDiff_Truncates(t *testing.T) {
	big := "--- a/x.go\n+++ b/x.go\n@@ -0,0 +1,1 @@\n+" + strings.Repeat("a", maxDiffBytes)
	got := reviewDiff(big)
	if len(got) <= maxDiffBytes || !strings.HasSuffix(got, "[diff truncado por exceder o limite de tamanho]") {
		t.Fatalf("expected a truncated diff with marker, got len %d", len(got))
	}
}

func TestIsIrrelevant(t *testing.T) {
	cases := map[string]bool{
		"package-lock.json":            true,
		"yarn.lock":                    true,
		"go.sum":                       true,
		"pnpm-lock.yaml":               true,
		"dist/app.js":                  true,
		"build/output.js":              true,
		"node_modules/lodash/index.js": true,
		"vendor/github.com/foo/bar.go": true,
		"app.min.js":                   true,
		"app.min.css":                  true,
		"main.map":                     true,
		"api.pb.go":                    true,
		"src/main.go":                  false,
		"internal/domain/review.go":    false,
		"README.md":                    false,
	}
	for path, want := range cases {
		if got := isIrrelevant(path); got != want {
			t.Errorf("isIrrelevant(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestAnnotateDiffLineNumbers(t *testing.T) {
	diff := "--- a/a.go\n+++ b/a.go\n@@ -3,3 +3,3 @@\n keep\n-old\n+new\n tail"
	want := "--- a/a.go\n+++ b/a.go\n@@ -3,3 +3,3 @@\n [old:3 new:3]keep\n-[old:4] old\n+[new:4] new\n [old:5 new:5]tail"
	if got := annotateDiffLineNumbers(diff); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// contractDiff exercises the cases where annotation and validation used to
// disagree: a "\ No newline" marker and a removed "-- sql comment" line.
const contractDiff = `diff --git a/db/q.sql b/db/q.sql
--- a/db/q.sql
+++ b/db/q.sql
@@ -1,3 +1,3 @@
--- old comment
 select 1;
-select 2;
\ No newline at end of file
+select 3;
+select 4;
\ No newline at end of file
diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -10,0 +11,2 @@
+func a() {}
+func b() {}
`

func TestAnnotateAndValidateAgree(t *testing.T) {
	annotated := annotateDiffLineNumbers(contractDiff)

	// Collect (path, line) for every [new:N] marker the model is shown.
	marker := regexp.MustCompile(`^\+\[new:(\d+)\] `)
	var shown []domain.InlineComment
	path := ""
	for _, l := range strings.Split(annotated, "\n") {
		if p, ok := strings.CutPrefix(l, "+++ b/"); ok {
			path = p
			continue
		}
		if m := marker.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[1])
			shown = append(shown, domain.InlineComment{Path: path, Line: n, Body: "c" + m[1]})
		}
	}

	want := []domain.InlineComment{
		{Path: "db/q.sql", Line: 2, Body: "c2"},
		{Path: "db/q.sql", Line: 3, Body: "c3"},
		{Path: "main.go", Line: 11, Body: "c11"},
		{Path: "main.go", Line: 12, Body: "c12"},
	}
	if len(shown) != len(want) {
		t.Fatalf("markers shown to the model = %+v, want %+v\n%s", shown, want, annotated)
	}
	for i := range want {
		if shown[i] != want[i] {
			t.Fatalf("marker %d = %+v, want %+v\n%s", i, shown[i], want[i], annotated)
		}
	}
	if got := validateInlineComments(shown, contractDiff); len(got) != len(shown) {
		t.Fatalf("validation rejected lines the model was shown: kept %+v of %+v", got, shown)
	}
	if !strings.Contains(annotated, "-[old:1] -- old comment") {
		t.Fatalf("removed SQL comment not annotated as a removed line:\n%s", annotated)
	}
	off := []domain.InlineComment{{Path: "db/q.sql", Line: 4, Body: "past the end"}}
	if got := validateInlineComments(off, contractDiff); len(got) != 0 {
		t.Fatalf("line after the \\ No newline marker must not exist, got %+v", got)
	}
}

func TestValidateInlineComments_KeepsOnlyAddedLines(t *testing.T) {
	diff := "--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,3 @@\n package main\n-func old() {}\n+func run() {}\n func keep() {}\n"
	got := validateInlineComments([]domain.InlineComment{
		{Path: "main.go", Line: 2, Body: "ok"},
		{Path: "main.go", Line: 1, Body: "context line"},
		{Path: "main.go", Line: 99, Body: "missing"},
		{Path: "other.go", Line: 2, Body: "wrong file"},
		{Path: "main.go", Line: 2, Body: "  "},
	}, diff)
	if len(got) != 1 || got[0] != (domain.InlineComment{Path: "main.go", Line: 2, Body: "ok"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateInlineComments_NormalizesPathsAndDeduplicates(t *testing.T) {
	diff := "--- a/src/app.go\n+++ b/src/app.go\n@@ -10,0 +11,2 @@\n+func a() {}\n+func b() {}\n"
	got := validateInlineComments([]domain.InlineComment{
		{Path: "b/src/app.go", Line: 11, Body: "Rever nome."},
		{Path: "src/app.go", Line: 11, Body: "Rever nome."},
		{Path: "a/src/app.go", Line: 12, Body: "  Adicionar teste.  "},
	}, diff)
	want := []domain.InlineComment{
		{Path: "src/app.go", Line: 11, Body: "Rever nome."},
		{Path: "src/app.go", Line: 12, Body: "Adicionar teste."},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestValidateInlineComments_CapsAfterValidation(t *testing.T) {
	var b strings.Builder
	b.WriteString("--- a/x.go\n+++ b/x.go\n@@ -0,0 +1,30 @@\n")
	for i := 0; i < 30; i++ {
		b.WriteString("+x\n")
	}
	var comments []domain.InlineComment
	for i := 0; i < maxInlineComments; i++ { // invalid ones first must not use up the cap
		comments = append(comments, domain.InlineComment{Path: "x.go", Line: 1000 + i, Body: "bad"})
	}
	for i := 1; i <= 30; i++ {
		comments = append(comments, domain.InlineComment{Path: "x.go", Line: i, Body: "ok"})
	}
	got := validateInlineComments(comments, b.String())
	if len(got) != maxInlineComments || got[0].Line != 1 {
		t.Fatalf("got %d comments starting at %+v, want %d valid ones", len(got), got[0], maxInlineComments)
	}
}
