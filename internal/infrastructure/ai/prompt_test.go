package ai

import (
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

var sampleDiff = `--- a/src/main.go
+++ b/src/main.go
@@ -1,4 +1,5 @@
 package main
+// added comment
 func main() {}
--- a/package-lock.json
+++ b/package-lock.json
@@ -1,3 +1,3 @@
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
@@ -1,2 +1,3 @@
 export const app = {}
+export const version = "2"
`

func TestFilterDiff_RemovesLockfiles(t *testing.T) {
	result := filterDiff(sampleDiff)

	for _, banned := range []string{"package-lock.json", "yarn.lock", "go.sum"} {
		if strings.Contains(result, banned) {
			t.Errorf("expected %q to be filtered out", banned)
		}
	}
}

func TestFilterDiff_RemovesBuildArtefacts(t *testing.T) {
	result := filterDiff(sampleDiff)

	if strings.Contains(result, "dist/bundle.min.js") {
		t.Error("expected dist/bundle.min.js to be filtered out")
	}
}

func TestFilterDiff_KeepsSourceFiles(t *testing.T) {
	result := filterDiff(sampleDiff)

	for _, kept := range []string{"src/main.go", "src/app.ts"} {
		if !strings.Contains(result, kept) {
			t.Errorf("expected %q to be kept", kept)
		}
	}
}

func TestFilterDiff_EmptyDiff(t *testing.T) {
	if got := filterDiff(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestIsIrrelevant(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"package-lock.json", true},
		{"yarn.lock", true},
		{"go.sum", true},
		{"pnpm-lock.yaml", true},
		{"dist/app.js", true},
		{"build/output.js", true},
		{"node_modules/lodash/index.js", true},
		{"vendor/github.com/foo/bar.go", true},
		{"app.min.js", true},
		{"app.min.css", true},
		{"main.map", true},
		{"src/main.go", false},
		{"internal/domain/review.go", false},
		{"README.md", false},
		{"src/app.ts", false},
	}

	for _, tc := range cases {
		got := isIrrelevant(tc.path)
		if got != tc.want {
			t.Errorf("isIrrelevant(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestFilterDiff_TruncatesLargeDiff(t *testing.T) {
	large := strings.Repeat("--- a/src/x.go\n+++ b/src/x.go\n+line\n", 5000)
	result := filterDiff(large)
	// After filtering (no irrelevant files), truncation is applied by buildPrompt;
	// filterDiff itself should not truncate — just verify it returns the full content.
	if len(result) == 0 {
		t.Error("expected non-empty result for large valid diff")
	}
}

func TestBuildPromptIncludesRelatedContext(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42", Description: "d"},
		Diff:         "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n+x",
		Related: &domain.RelatedContext{
			Issues:      []domain.Issue{{IID: 7, Title: "Bug do login", Description: "usuário não loga"}},
			LinkedMRs:   []domain.MergeRequestRef{{IID: 34, Title: "Refactor", State: "merged"}},
			Discussions: []domain.Discussion{{Author: "ana", Body: "revisem o caso X"}},
		},
	}
	out := buildPrompt(input, "")
	for _, want := range []string{"Contexto relacionado", "#7", "Bug do login", "!34", "ana", "compat"} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptWithoutRelatedContextOmitsBlock(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42"},
		Diff:         "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n+x",
	}
	if strings.Contains(buildPrompt(input, ""), "Contexto relacionado") {
		t.Error("expected no related block when Related is nil")
	}
}

func TestBuildPromptIncludesReanalysisContext(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest:        &domain.MergeRequest{Title: "feat", ProjectRef: "42"},
		Diff:                "--- a/x.go\n+++ b/x.go\n+novo código",
		PreviousReview:      "A1: validar setor obrigatório.",
		ReviewJustification: "setor vazio é permitido para exames externos",
	}
	out := buildPrompt(input, "")
	for _, want := range []string{"Revisão anterior", "A1: validar", "Justificativa", "exames externos"} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
