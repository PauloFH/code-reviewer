package ai

import (
	"strings"
	"testing"
)

// validStructuredPayload mimics a real review: structured fields, escaped \n,
// backticks, emoji, pipe tables. It MUST assemble into markdown, never post raw.
const validStructuredPayload = `{
  "guia": "Este MR refatora o ciclo de vida de certificados mTLS.",
  "diagrama_sequencia": "sequenceDiagram\n    participant Admin\n    participant TokenService\n    Admin->>TokenService: gerarToken()",
  "diagrama_classes": "classDiagram\n    class ProvisioningTokenService {\n        +gerarToken(scanner)\n    }",
  "alteracoes": "| Arquivo | Mudança | Detalhes |\n|---|---|---|\n| docs/x.md | Doc | Specs |",
  "pontos_atencao": "- A1 🔴 — Signer.java — ` + "`ObjectMapper`" + ` estático inseguro",
  "conclusao": "⚠️ Aprovado com ressalvas",
  "inline_comments": []
}`

func TestParseReviewPayload_ValidStructuredAssembles(t *testing.T) {
	body, _ := parseReviewPayload(validStructuredPayload)
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("posted raw JSON instead of markdown:\n%s", body)
	}
	if !strings.Contains(body, "## Guia do Revisor") || !strings.Contains(body, "```mermaid\nsequenceDiagram") {
		t.Fatalf("did not assemble markdown:\n%s", body)
	}
}

// TestParseReviewPayload_ProseWrappedAgenticJSON reproduces the agentic-mode
// failure: the model prepends a sentence and wraps the JSON in a ```json fence.
// The envelope must still be recovered and assembled, not posted raw.
func TestParseReviewPayload_ProseWrappedAgenticJSON(t *testing.T) {
	content := "Já tenho material suficiente. Vou montar a revisão.\n\n```json\n" +
		validStructuredPayload + "\n```"

	body, _ := parseReviewPayload(content)

	if strings.Contains(body, "Vou montar a revisão") {
		t.Fatalf("prose preamble leaked into the posted body:\n%s", body)
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "## Guia do Revisor") {
		t.Fatalf("agentic prose-wrapped JSON not recovered:\n%s", body)
	}
	if strings.Contains(body, "```json") {
		t.Fatalf("raw json fence leaked into the body:\n%s", body)
	}
}

func TestParseReviewPayload_TruncatedFallsBackToRaw(t *testing.T) {
	// max_tokens cutting the JSON mid-string yields no balanced object: we must
	// not crash; we post the raw text. The real mitigation is an adequate
	// max_tokens (this just documents graceful degradation).
	truncated := validStructuredPayload[:len(validStructuredPayload)-120]
	body, _ := parseReviewPayload(truncated)
	if body == "" {
		t.Fatal("expected non-empty fallback body for truncated input")
	}
}

func TestParseReviewPayload_TruncatedStructuredPayloadUsesCompletedFields(t *testing.T) {
	truncated := validStructuredPayload[:len(validStructuredPayload)-120]
	body, _ := parseReviewPayload(truncated)

	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("posted raw truncated JSON instead of recovered markdown:\n%s", body)
	}
	if !strings.Contains(body, "## Guia do Revisor") {
		t.Fatalf("did not recover the completed structured fields:\n%s", body)
	}
}

func TestParseReviewPayload_AppendsCorrectionPrompt(t *testing.T) {
	payload := `{"guia":"x","prompt_correcao":"Corrija A1 em foo.go linha 10.","conclusao":"OK"}`
	body, _ := parseReviewPayload(payload)
	if !strings.Contains(body, "### 🔧 Prompt para corrigir (cole numa IA)") {
		t.Fatalf("missing correction-prompt header:\n%s", body)
	}
	if !strings.Contains(body, "~~~text\nCorrija A1 em foo.go linha 10.\n~~~") {
		t.Fatalf("correction prompt not wrapped in a copyable block:\n%s", body)
	}
}

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name, in, want string
		ok             bool
	}{
		{"plain", `{"a":1}`, `{"a":1}`, true},
		{"prose prefix and suffix", "bla {\"a\":1} fim", `{"a":1}`, true},
		{"json fence", "```json\n{\"a\":1}\n```", `{"a":1}`, true},
		{"braces inside string", `{"x":"a}b{c","y":2}`, `{"x":"a}b{c","y":2}`, true},
		{"escaped quote inside string", `{"x":"a\"}b"}`, `{"x":"a\"}b"}`, true},
		{"no object", "sem json aqui", "", false},
	}
	for _, tc := range cases {
		got, ok := extractJSONObject(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: extractJSONObject(%q) = (%q,%v), want (%q,%v)", tc.name, tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
