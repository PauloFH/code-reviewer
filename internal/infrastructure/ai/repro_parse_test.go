package ai

import (
	"strings"
	"testing"
)

// reproValidPayload mimics the screenshot: structured fields, escaped \n,
// backticks, emoji, pipe tables. This MUST assemble into markdown, not post raw.
const reproValidPayload = `{
  "guia": "Este MR refatora o ciclo de vida de certificados mTLS.",
  "diagrama_sequencia": "sequenceDiagram\n    participant Admin\n    participant TokenService\n    Admin->>TokenService: gerarToken()",
  "diagrama_classes": "classDiagram\n    class ProvisioningTokenService {\n        +gerarToken(scanner)\n    }",
  "alteracoes": "| Arquivo | Mudança | Detalhes |\n|---|---|---|\n| docs/x.md | Doc | Specs |",
  "pontos_atencao": "- A1 🔴 — Signer.java — ` + "`ObjectMapper`" + ` estático inseguro",
  "sugestoes": "- M1 — Payload.java — usar ` + "`@JsonProperty`" + ` explícito",
  "riscos": "- R1 — remoção de /provision é breaking change",
  "testes": "- T1 — round-trip de assinatura",
  "conclusao": "⚠️ Aprovado com ressalvas",
  "inline_comments": []
}`

func TestRepro_ValidStructuredAssembles(t *testing.T) {
	body, _ := parseReviewPayload(reproValidPayload)
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("REGRESSION: posted raw JSON instead of markdown:\n%s", body)
	}
	if !strings.Contains(body, "## Guia do Revisor") {
		t.Fatalf("did not assemble markdown:\n%s", body)
	}
	if !strings.Contains(body, "```mermaid\nsequenceDiagram") {
		t.Fatalf("mermaid fence missing:\n%s", body)
	}
	t.Logf("OK assembled:\n%s", body)
}

func TestRepro_TruncatedPayloadFallsBackToRaw(t *testing.T) {
	// Simulate max_tokens cutting the JSON mid-string (invalid JSON).
	truncated := reproValidPayload[:len(reproValidPayload)-120]
	body, _ := parseReviewPayload(truncated)
	t.Logf("truncated -> body starts with %q", firstN(body, 40))
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Logf("CONFIRMED: truncated JSON is posted raw (the production symptom)")
	}
}

func TestRepro_LiteralNewlineInString(t *testing.T) {
	// Model emitted a real newline inside a JSON string instead of \n.
	bad := "{\"guia\": \"linha1\nlinha2\", \"conclusao\": \"ok\"}"
	body, _ := parseReviewPayload(bad)
	t.Logf("literal-newline -> body starts with %q", firstN(body, 40))
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Logf("CONFIRMED: unescaped newline makes JSON invalid -> posted raw")
	}
}

func firstN(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
