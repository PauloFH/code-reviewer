package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type sectionKind int

const (
	sectionText    sectionKind = iota
	sectionDiagram             // pure Mermaid body; the fence is added here, never by the model
	sectionPrompt              // copy-pasteable text block
)

// reviewSection is one text field of the review JSON. The ordered table drives
// the schema shown to the model, decoding and the assembled markdown.
type reviewSection struct {
	key, header string
	kind        sectionKind
	instruction string
}

var reviewSections = []reviewSection{
	{"guia", "## Guia do Revisor", sectionText, `Resumo de 2-3 frases explicando o que o MR faz e qual o impacto arquitetural.`},
	{"diagrama_sequencia", "### Diagrama de sequência", sectionDiagram, `Corpo PURO de um diagrama Mermaid de sequência (normalmente começa com 'sequenceDiagram'). Apenas a sintaxe do diagrama, SEM crases e SEM a palavra 'mermaid'. Use \"\" (string vazia) se não se aplicar.`},
	{"diagrama_classes", "### Diagrama de classes/componentes", sectionDiagram, `Corpo PURO de um diagrama Mermaid de classes/componentes (normalmente começa com 'classDiagram'). Apenas a sintaxe do diagrama, SEM crases e SEM a palavra 'mermaid'. Use \"\" se não se aplicar.`},
	{"alteracoes", "### Alterações por arquivo", sectionText, `Markdown com uma tabela | Mudança | Detalhes | Arquivos | resumindo as alterações por arquivo.`},
	{"pontos_atencao", "### Pontos de atenção", sectionText, `Markdown em lista: bugs, vulnerabilidades, race conditions, edge cases não tratados.`},
	{"sugestoes", "### Sugestões de melhoria", sectionText, `Markdown em lista: refatorações, simplificações, melhor nomenclatura, padrões mais adequados.`},
	{"riscos", "### Riscos", sectionText, `Markdown em lista: breaking changes, impacto em performance, regressões possíveis.`},
	{"testes", "### Testes sugeridos", sectionText, `Markdown em lista: cenários que deveriam ter cobertura de teste.`},
	{"conclusao", "### Conclusão", sectionText, `Veredicto: ✅ Aprovado | ⚠️ Aprovado com ressalvas | ❌ Solicitar alterações — com justificativa breve.`},
	{"prompt_correcao", "### 🔧 Prompt para corrigir (cole numa IA)", sectionPrompt, `Prompt pronto para colar num assistente de IA de código (ex.: Claude Code/Cursor) que corrija os problemas. Em modo imperativo, liste os pontos acionáveis (pontos de atenção + sugestões + riscos), cada um com o arquivo e, quando houver, a linha, e peça para rodar os testes após corrigir. Texto puro, SEM crases. Use \"\" se o MR não precisar de correções.`},
}

// reviewJSONSchema renders the example object the prompt asks the model to return.
func reviewJSONSchema() string {
	var b strings.Builder
	b.WriteString("{\n")
	for _, s := range reviewSections {
		fmt.Fprintf(&b, "  %q: \"%s\",\n", s.key, s.instruction)
	}
	b.WriteString(`  "inline_comments": [
    {
      "path": "caminho/do/arquivo.ext",
      "line": 123,
      "body": "comentário curto e objetivo em português brasileiro"
    }
  ]
}`)
	return b.String()
}

const (
	mermaidFenceOpen = "```mermaid"
	codeFenceClose   = "```"
)

// markdownSanitizer strips invisible characters that have broken Mermaid 11
// rendering: U+00A0 becomes a normal space, U+200B and U+FEFF are removed.
var markdownSanitizer = strings.NewReplacer(
	"\u00a0", " ",
	"\u200b", "",
	"\ufeff", "",
)

func sanitizeMarkdown(s string) string {
	return markdownSanitizer.Replace(s)
}

// reviewPayload is the decoded review JSON. markdown is the legacy
// review_markdown single-body fallback.
type reviewPayload struct {
	sections map[string]string
	markdown string
	comments []domain.InlineComment
}

// parseReviewPayload turns the model's answer into the markdown body and the
// raw (sanitised, not yet diff-validated) inline comments.
func parseReviewPayload(content string) (string, []domain.InlineComment) {
	trimmed := strings.TrimSpace(content)
	payload, ok := decodeReviewPayload(trimmed)
	if !ok {
		return sanitizeMarkdown(trimmed), nil
	}

	body := assembleReviewMarkdown(payload)
	if body == "" {
		body = sanitizeMarkdown(trimmed)
	}
	comments := make([]domain.InlineComment, 0, len(payload.comments))
	for _, c := range payload.comments {
		c.Path = strings.TrimSpace(c.Path)
		c.Body = sanitizeMarkdown(strings.TrimSpace(c.Body))
		comments = append(comments, c)
	}
	return body, comments
}

// decodeReviewPayload tries the whole (fence-stripped) text, then the first
// balanced {…} object — agentic models often prepend prose like "Vou montar a
// revisão:" — and finally recovers the completed fields of a truncated object.
func decodeReviewPayload(trimmed string) (reviewPayload, bool) {
	candidates := []string{stripOuterCodeFence(trimmed)}
	if obj, ok := extractJSONObject(trimmed); ok {
		candidates = append(candidates, obj)
	}
	for _, c := range candidates {
		if p, ok := unmarshalReviewPayload(c); ok {
			return p, true
		}
	}
	return recoverTruncatedReviewPayload(trimmed)
}

func unmarshalReviewPayload(s string) (reviewPayload, bool) {
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &keyed); err != nil {
		return reviewPayload{}, false
	}
	raw := make(map[string]json.RawMessage, len(keyed))
	for k, v := range keyed {
		raw[strings.ToLower(k)] = v
	}
	p := reviewPayload{sections: make(map[string]string, len(reviewSections))}
	field := func(key string, dst any) bool {
		v, ok := raw[key]
		return !ok || json.Unmarshal(v, dst) == nil
	}
	for _, s := range reviewSections {
		var v string
		if !field(s.key, &v) {
			return reviewPayload{}, false
		}
		p.sections[s.key] = v
	}
	if !field("review_markdown", &p.markdown) || !field("inline_comments", &p.comments) {
		return reviewPayload{}, false
	}
	return p, true
}

func (p reviewPayload) hasStructuredContent() bool {
	for _, v := range p.sections {
		if v != "" {
			return true
		}
	}
	return false
}

// recoverTruncatedReviewPayload handles an answer cut off by max_tokens: it
// keeps the longest prefix of top-level fields that still decodes.
func recoverTruncatedReviewPayload(s string) (reviewPayload, bool) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return reviewPayload{}, false
	}
	s = s[start:]
	var recovered reviewPayload
	ok := false
	walkJSON(s, func(i, depth int) bool {
		if s[i] == ',' && depth == 1 {
			if p, decoded := unmarshalReviewPayload(s[:i] + "}"); decoded && p.hasStructuredContent() {
				recovered, ok = p, true
			}
		}
		return true
	})
	return recovered, ok
}

// extractJSONObject returns the first complete, balanced {…} object in s.
func extractJSONObject(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return "", false
	}
	s = s[start:]
	end := -1
	walkJSON(s, func(i, depth int) bool {
		if s[i] == '}' && depth == 0 {
			end = i
			return false
		}
		return true
	})
	if end < 0 {
		return "", false
	}
	return s[:end+1], true
}

// walkJSON calls fn for every byte of s outside JSON string literals, with the
// brace depth after that byte; fn returns false to stop. Scanning bytes is safe
// because '{', '}', '"' and '\\' never appear in UTF-8 continuation bytes.
func walkJSON(s string, fn func(i, depth int) bool) {
	depth := 0
	inStr, escaped := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			continue
		case '{':
			depth++
		case '}':
			depth--
		}
		if !fn(i, depth) {
			return
		}
	}
}

// assembleReviewMarkdown renders the sections in table order. A payload with
// only the legacy review_markdown body returns it as-is. Always sanitised.
func assembleReviewMarkdown(p reviewPayload) string {
	if !p.hasStructuredContent() {
		return sanitizeMarkdown(strings.TrimSpace(p.markdown))
	}

	var b strings.Builder
	for _, s := range reviewSections {
		raw := strings.TrimSpace(p.sections[s.key])
		switch s.kind {
		case sectionDiagram:
			// Drop any fence the model emitted anyway, leaving pure Mermaid.
			raw = strings.TrimSpace(stripOuterCodeFence(raw))
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "mermaid"))
		case sectionPrompt:
			raw = strings.TrimSpace(stripOuterCodeFence(raw))
		}
		if raw == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(s.header)
		b.WriteString("\n\n")
		switch s.kind {
		case sectionDiagram:
			b.WriteString(mermaidFenceOpen + "\n" + raw + "\n" + codeFenceClose)
		case sectionPrompt:
			// "~~~" so a prompt that itself contains "```" can't break out of the block.
			b.WriteString("~~~text\n" + raw + "\n~~~")
		default:
			b.WriteString(raw)
		}
	}
	return sanitizeMarkdown(b.String())
}

// stripOuterCodeFence removes a single outer ``` or ~~~ wrapper (with an
// optional language tag) when the whole string is wrapped in one; interior
// fences are preserved.
func stripOuterCodeFence(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```", "~~~"} {
		if !strings.HasPrefix(s, fence) || !strings.HasSuffix(s, fence) {
			continue
		}
		nl := strings.IndexByte(s, '\n')
		if nl == -1 {
			continue
		}
		inner := s[nl+1:]
		if !strings.HasSuffix(inner, fence) {
			continue
		}
		return strings.TrimSpace(inner[:len(inner)-len(fence)])
	}
	return s
}
