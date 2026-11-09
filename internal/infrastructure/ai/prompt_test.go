package ai

import (
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const promptDiff = "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n+x"

func TestBuildPrompt_LabelsProjectAndAnnotatesDiff(t *testing.T) {
	out := buildPrompt(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "acme/repo"},
	}, promptDiff, "")
	for _, want := range []string{"**Projeto:** acme/repo", "+[new:1] x", `"prompt_correcao": "`, "no máximo 20 comentários"} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptIncludesRelatedContext(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42", Description: "d"},
		Related: &domain.RelatedContext{
			Issues:      []domain.Issue{{IID: 7, Title: "Bug do login", Description: "usuário não loga"}},
			LinkedMRs:   []domain.MergeRequestRef{{IID: 34, Title: "Refactor", State: "merged"}},
			Discussions: []domain.Discussion{{Author: "ana", Body: "revisem o caso X"}},
		},
	}
	out := buildPrompt(input, promptDiff, "")
	for _, want := range []string{"Contexto relacionado", "#7", "Bug do login", "!34", "ana", "compat"} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptWithoutRelatedContextOmitsBlock(t *testing.T) {
	input := domain.ReviewInput{MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42"}}
	if strings.Contains(buildPrompt(input, promptDiff, ""), "Contexto relacionado") {
		t.Error("expected no related block when Related is nil")
	}
}

func TestBuildPromptIncludesReanalysisContextAndTemplate(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest:        &domain.MergeRequest{Title: "feat", ProjectRef: "42"},
		PreviousReview:      "A1: validar setor obrigatório.",
		ReviewJustification: "setor vazio é permitido para exames externos",
	}
	out := buildPrompt(input, promptDiff, "Foque em segurança.")
	for _, want := range []string{"Revisão anterior", "A1: validar", "Justificativa", "exames externos", "Diretrizes adicionais", "Foque em segurança."} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
