package ai

import (
	"strings"
	"testing"
)

// fullPayload mimics a real review: escaped \n, backticks, emoji, pipe tables.
const fullPayload = `{
  "guia": "Este MR refatora o ciclo de vida de certificados mTLS.",
  "diagrama_sequencia": "sequenceDiagram\n    participant Admin\n    Admin->>TokenService: gerarToken()",
  "diagrama_classes": "classDiagram\n    class ProvisioningTokenService",
  "alteracoes": "| Arquivo | Mudança |\n|---|---|\n| docs/x.md | Doc |",
  "pontos_atencao": "- A1 🔴 — Signer.java — ` + "`ObjectMapper`" + ` estático inseguro",
  "conclusao": "⚠️ Aprovado com ressalvas",
  "prompt_correcao": "Corrija A1 em Signer.java.",
  "inline_comments": []
}`

func TestParseReviewPayload_AssemblesSectionsInOrder(t *testing.T) {
	body, _ := parseReviewPayload(fullPayload)
	want := "## Guia do Revisor\n\nEste MR refatora o ciclo de vida de certificados mTLS." +
		"\n\n### Diagrama de sequência\n\n```mermaid\nsequenceDiagram\n    participant Admin\n    Admin->>TokenService: gerarToken()\n```" +
		"\n\n### Diagrama de classes/componentes\n\n```mermaid\nclassDiagram\n    class ProvisioningTokenService\n```" +
		"\n\n### Alterações por arquivo\n\n| Arquivo | Mudança |\n|---|---|\n| docs/x.md | Doc |" +
		"\n\n### Pontos de atenção\n\n- A1 🔴 — Signer.java — `ObjectMapper` estático inseguro" +
		"\n\n### Conclusão\n\n⚠️ Aprovado com ressalvas" +
		"\n\n### 🔧 Prompt para corrigir (cole numa IA)\n\n~~~text\nCorrija A1 em Signer.java.\n~~~"
	if body != want {
		t.Fatalf("body mismatch:\ngot:\n%s\nwant:\n%s", body, want)
	}
}

func TestParseReviewPayload_LegacyMarkdownAndComments(t *testing.T) {
	body, comments := parseReviewPayload(`{
		"review_markdown": "## Revisao\n\nTexto",
		"inline_comments": [{"path": " main.go ", "line": 10, "body": " Adicionar teste. "}]
	}`)
	if body != "## Revisao\n\nTexto" {
		t.Fatalf("unexpected body: %q", body)
	}
	if len(comments) != 1 || comments[0].Path != "main.go" || comments[0].Line != 10 || comments[0].Body != "Adicionar teste." {
		t.Fatalf("unexpected comments: %+v", comments)
	}
}

func TestParseReviewPayload_FallsBackToRawText(t *testing.T) {
	for _, in := range []string{
		"review plain text",
		`{"guia": 3, "conclusao": "x"}`,  // wrong field type: not our envelope
		"{\"guia\": \"linha1\nlinha2\"}", // literal newline in a JSON string
	} {
		body, comments := parseReviewPayload(in)
		if body != in || comments != nil {
			t.Errorf("parseReviewPayload(%q) = (%q, %v), want raw text", in, body, comments)
		}
	}
}

func TestParseReviewPayload_DiagramFenceStrippedDefensively(t *testing.T) {
	body, _ := parseReviewPayload("{\"guia\": \"x\", \"diagrama_sequencia\": \"```mermaid\\nsequenceDiagram\\n  A->>B: req\\n```\"}")
	want := "## Guia do Revisor\n\nx\n\n### Diagrama de sequência\n\n```mermaid\nsequenceDiagram\n  A->>B: req\n```"
	if body != want {
		t.Fatalf("got:\n%s\nwant:\n%s", body, want)
	}
}

func TestParseReviewPayload_SanitizesInvisibleChars(t *testing.T) {
	in := "{\"guia\": \"a b\u00a0c\ufeff\", \"inline_comments\": [{\"path\":\"a.go\",\"line\":1,\"body\":\"x\u200by\"}]}"
	body, comments := parseReviewPayload(in)
	if body != "## Guia do Revisor\n\na b c" {
		t.Fatalf("body = %q", body)
	}
	if len(comments) != 1 || comments[0].Body != "xy" {
		t.Fatalf("inline comment not sanitised: %+v", comments)
	}
}

func TestParseReviewPayload_UnwrapsJSONInCodeFence(t *testing.T) {
	body, _ := parseReviewPayload("```json\n{\"guia\": \"olá\"}\n```")
	if body != "## Guia do Revisor\n\nolá" {
		t.Fatalf("fenced JSON not unwrapped/assembled: %q", body)
	}
}

func TestParseReviewPayload_ProseWrappedAgenticJSON(t *testing.T) {
	plain, _ := parseReviewPayload(fullPayload)
	wrapped, _ := parseReviewPayload("Já tenho material suficiente. Vou montar a revisão.\n\n```json\n" + fullPayload + "\n```")
	if wrapped != plain {
		t.Fatalf("prose-wrapped JSON not recovered:\n%s", wrapped)
	}
}

func TestParseReviewPayload_TruncatedKeepsCompletedFields(t *testing.T) {
	cut := strings.Index(fullPayload, `"pontos_atencao"`) + 30
	body, _ := parseReviewPayload(fullPayload[:cut])
	if !strings.HasPrefix(body, "## Guia do Revisor\n\nEste MR") || !strings.HasSuffix(body, "| docs/x.md | Doc |") {
		t.Fatalf("expected the fields completed before the cut, got:\n%s", body)
	}
}

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name, in, want string
		ok             bool
	}{
		{"plain", `{"a":1}`, `{"a":1}`, true},
		{"prose prefix and suffix", "bla {\"a\":1} fim", `{"a":1}`, true},
		{"nested", `x {"a":{"b":1}} y`, `{"a":{"b":1}}`, true},
		{"braces inside string", `{"x":"a}b{c","y":2}`, `{"x":"a}b{c","y":2}`, true},
		{"escaped quote inside string", `{"x":"a\"}b"}`, `{"x":"a\"}b"}`, true},
		{"unbalanced", `{"a":{"b":1}`, "", false},
		{"no object", "sem json aqui", "", false},
	}
	for _, tc := range cases {
		got, ok := extractJSONObject(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: extractJSONObject(%q) = (%q,%v), want (%q,%v)", tc.name, tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestStripOuterCodeFence(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"json fence", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"tilde fence", "~~~\nx\n~~~", "x"},
		{"no fence", `{"a":1}`, `{"a":1}`},
		{"preserves inner fences", "```json\n{\"x\":\"~~~\"}\n```", `{"x":"~~~"}`},
	}
	for _, tc := range cases {
		if got := stripOuterCodeFence(tc.in); got != tc.want {
			t.Errorf("%s: stripOuterCodeFence(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}
