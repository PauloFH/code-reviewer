package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestClientReviewDiff_CallsGeminiGenerateContent(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	var gotReq geminiRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-goog-api-key")

		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decoding request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{"text": "Revisao "},
							{"text": "ok"}
						],
						"role": "model"
					},
					"finishReason": "STOP",
					"index": 0
				}
			],
			"usageMetadata": {
				"promptTokenCount": 8,
				"candidatesTokenCount": 11,
				"totalTokenCount": 373
			}
		}`))
	}))
	defer server.Close()

	client := New("gemini", server.URL, "test-key", "gemini-flash-latest", time.Second, nil)

	result, err := client.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{
			Title:       "MR",
			ProjectRef:  "owner/repo",
			Description: "desc",
			WebURL:      "https://example.com/mr/1",
		},
		Diff: "+func main() {}",
	})
	if err != nil {
		t.Fatalf("ReviewDiff returned error: %v", err)
	}

	if gotPath != "/models/gemini-flash-latest:generateContent" {
		t.Fatalf("path = %q, want Gemini generateContent path", gotPath)
	}
	if gotAPIKey != "test-key" {
		t.Fatalf("X-goog-api-key = %q, want test-key", gotAPIKey)
	}
	if gotReq.SystemInstruction == nil || len(gotReq.SystemInstruction.Parts) != 1 {
		t.Fatalf("expected systemInstruction with one text part")
	}
	if gotReq.GenerationConfig.Temperature != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", gotReq.GenerationConfig.Temperature)
	}
	if len(gotReq.Contents) != 1 || gotReq.Contents[0].Role != "user" || len(gotReq.Contents[0].Parts) != 1 {
		t.Fatalf("expected one user content with one text part")
	}
	if !strings.Contains(gotReq.Contents[0].Parts[0].Text, "## Diff") {
		t.Fatalf("user prompt did not include diff section")
	}

	if result.Body != "Revisao ok" {
		t.Fatalf("Body = %q, want concatenated text", result.Body)
	}
	if result.Usage.PromptTokens != 8 || result.Usage.CompletionTokens != 11 || result.Usage.TotalTokens != 373 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
}

func TestClientReviewDiff_EmptyGeminiCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [
				{
					"content": {"parts": [], "role": "model"},
					"finishReason": "MAX_TOKENS"
				}
			]
		}`))
	}))
	defer server.Close()

	client := New("gemini", server.URL, "test-key", "gemini-flash-latest", time.Second, nil)

	_, err := client.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "MR"},
		Diff:         "+change",
	})
	if err == nil || !strings.Contains(err.Error(), "MAX_TOKENS") {
		t.Fatalf("expected empty response error with finish reason, got %v", err)
	}
}

func TestClientReviewDiff_CallsChatCompletionsProvider(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotReq chatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decoding request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{"message": {"content": "review via chat completions"}}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30
			}
		}`))
	}))
	defer server.Close()

	client := New("chat_completions", server.URL, "test-token", "claude-sonnet-4", time.Second, nil)

	result, err := client.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "MR"},
		Diff:         "+change",
	})
	if err != nil {
		t.Fatalf("ReviewDiff returned error: %v", err)
	}

	if gotPath != "/chat/completions" {
		t.Fatalf("path = %q, want /chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
	if gotReq.Model != "claude-sonnet-4" {
		t.Fatalf("Model = %q, want claude-sonnet-4", gotReq.Model)
	}
	if len(gotReq.Messages) != 2 || gotReq.Messages[0].Role != "system" || gotReq.Messages[1].Role != "user" {
		t.Fatalf("unexpected chat messages: %+v", gotReq.Messages)
	}
	if result.Body != "review via chat completions" {
		t.Fatalf("Body = %q, want chat response", result.Body)
	}
	if result.Usage.PromptTokens != 10 || result.Usage.CompletionTokens != 20 || result.Usage.TotalTokens != 30 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
}

func TestParseReviewPayload_JSONWithInlineComments(t *testing.T) {
	body, comments := parseReviewPayload(`{
		"review_markdown": "## Revisao\n\nTexto",
		"inline_comments": [
			{"path": "main.go", "line": 10, "body": "Adicionar teste."},
			{"path": "", "line": 20, "body": "ignorado"},
			{"path": "x.go", "line": 0, "body": "ignorado"}
		]
	}`)

	if body != "## Revisao\n\nTexto" {
		t.Fatalf("unexpected body: %q", body)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if comments[0].Path != "main.go" || comments[0].Line != 10 || comments[0].Body != "Adicionar teste." {
		t.Fatalf("unexpected inline comment: %+v", comments[0])
	}
}

func TestParseReviewPayload_FallbackToPlainText(t *testing.T) {
	body, comments := parseReviewPayload("review plain text")

	if body != "review plain text" {
		t.Fatalf("body = %q, want plain text", body)
	}
	if len(comments) != 0 {
		t.Fatalf("expected no inline comments, got %d", len(comments))
	}
}

func TestParseReviewPayload_StructuredAddsMermaidFences(t *testing.T) {
	body, _ := parseReviewPayload(`{
		"guia": "Adiciona endpoint de health.",
		"diagrama_sequencia": "sequenceDiagram\n  A->>B: req",
		"diagrama_classes": "",
		"pontos_atencao": "- nenhum",
		"conclusao": "✅ Aprovado"
	}`)

	// The model never emits fences; we add exactly one ```mermaid block here.
	if !strings.Contains(body, "### Diagrama de sequência\n\n```mermaid\nsequenceDiagram\n  A->>B: req\n```") {
		t.Fatalf("sequence diagram not fenced as expected:\n%s", body)
	}
	if strings.Contains(body, "Diagrama de classes") {
		t.Fatalf("empty diagram section should be omitted:\n%s", body)
	}
	if !strings.HasPrefix(body, "## Guia do Revisor\n\nAdiciona endpoint de health.") {
		t.Fatalf("guia section missing or misordered:\n%s", body)
	}
	if !strings.Contains(body, "### Conclusão\n\n✅ Aprovado") {
		t.Fatalf("conclusao section missing:\n%s", body)
	}
}

func TestParseReviewPayload_DiagramFenceStrippedDefensively(t *testing.T) {
	// Even if the model wrongly wraps the diagram, we keep a single clean fence.
	body, _ := parseReviewPayload(`{
		"guia": "x",
		"diagrama_sequencia": "` + "```mermaid" + `\nsequenceDiagram\n  A->>B: req\n` + "```" + `"
	}`)

	if strings.Count(body, "```mermaid") != 1 || strings.Count(body, "```") != 2 {
		t.Fatalf("expected exactly one mermaid fence pair:\n%s", body)
	}
	if strings.Contains(body, "mermaid\nmermaid") {
		t.Fatalf("language tag leaked into body:\n%s", body)
	}
}

func TestParseReviewPayload_SanitizesInvisibleChars(t *testing.T) {
	// U+00A0 (nbsp) -> space, U+200B and U+FEFF removed.
	in := "{\"guia\": \"a b\u00a0c\ufeff\", \"inline_comments\": [{\"path\":\"a.go\",\"line\":1,\"body\":\"x\u200by\"}]}"
	body, comments := parseReviewPayload(in)

	if strings.ContainsAny(body, "\u00a0\u200b\ufeff") {
		t.Fatalf("invisible chars survived in body: %q", body)
	}
	if !strings.Contains(body, "a b c") {
		t.Fatalf("nbsp not normalised to space: %q", body)
	}
	if len(comments) != 1 || comments[0].Body != "xy" {
		t.Fatalf("inline comment not sanitised: %+v", comments)
	}
}

func TestStripOuterCodeFence(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"json fence", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"bare fence", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"no fence", `{"a":1}`, `{"a":1}`},
		{"preserves inner fences", "```json\n{\"x\":\"~~~\"}\n```", `{"x":"~~~"}`},
	}
	for _, tc := range cases {
		if got := stripOuterCodeFence(tc.in); got != tc.want {
			t.Errorf("%s: stripOuterCodeFence(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestParseReviewPayload_UnwrapsJSONInCodeFence(t *testing.T) {
	body, _ := parseReviewPayload("```json\n{\"guia\": \"olá\"}\n```")
	if body != "## Guia do Revisor\n\nolá" {
		t.Fatalf("fenced JSON not unwrapped/assembled: %q", body)
	}
}
