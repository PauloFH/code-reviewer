package ai

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const systemPrompt = `Você é um revisor sênior de código com ampla experiência em engenharia de software.
Sua tarefa é revisar o diff de um Merge Request de forma objetiva, direta e em português brasileiro.
Não faça comentários genéricos. Foque apenas no que está no diff recebido.
Você responde SEMPRE com um único objeto JSON, com campos de texto separados.
NUNCA escreve crases triplas (` + "```" + `) em nenhum campo: os diagramas Mermaid são
devolvidos como corpo puro e o sistema adiciona as cercas de código automaticamente.`

const maxDiffBytes = 80_000
const maxInlineComments = 20

// irrelevantPatterns lists file paths/extensions to skip during diff filtering.
var irrelevantPatterns = []string{
	"package-lock.json",
	"pnpm-lock.yaml",
	"yarn.lock",
	"go.sum",
	"Cargo.lock",
	"composer.lock",
	"dist/",
	"build/",
	"target/",
	"node_modules/",
	"vendor/",
	".min.js",
	".min.css",
	".map",
	".pb.go", // generated protobuf
	"_generated",
}

// renderRelatedContext formats the related layer (issues, linked MRs, discussion)
// into a Markdown block, including an explicit compatibility question for issues.
func renderRelatedContext(rc *domain.RelatedContext) string {
	if rc.IsEmpty() {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Contexto relacionado\n")

	if len(rc.Issues) > 0 {
		b.WriteString("\n### Issues vinculadas\n")
		for _, is := range rc.Issues {
			fmt.Fprintf(&b, "\n**#%d — %s** (%s)\n%s\n", is.IID, is.Title, is.State, is.Description)
		}
		b.WriteString("\nAo revisar, verifique a **compatibilidade**: o MR cumpre o que cada issue acima pede? Aponte divergências explicitamente.\n")
	}

	if len(rc.LinkedMRs) > 0 {
		b.WriteString("\n### MRs relacionados\n")
		for _, m := range rc.LinkedMRs {
			fmt.Fprintf(&b, "- !%d — %s (%s) %s\n", m.IID, m.Title, m.State, m.WebURL)
		}
	}

	if len(rc.Discussions) > 0 {
		b.WriteString("\n### Discussão do MR\n")
		for _, d := range rc.Discussions {
			fmt.Fprintf(&b, "- **%s:** %s\n", d.Author, d.Body)
		}
	}

	return b.String()
}

func buildPrompt(input domain.ReviewInput, outputTemplate string) string {
	mr := input.MergeRequest
	diff := filterDiff(input.Diff)
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n[diff truncado por exceder o limite de tamanho]"
	}
	diff = annotateDiffLineNumbers(diff)

	related := ""
	if input.Related != nil {
		related = renderRelatedContext(input.Related)
	}

	extraGuidance := ""
	if outputTemplate != "" {
		extraGuidance = "\n\nDiretrizes adicionais de conteúdo (use como ênfase ao preencher os campos acima, sem alterar o formato JSON):\n\n" + outputTemplate
	}

	return fmt.Sprintf(`## Merge Request

**Título:** %s
**Plataforma:** %s
**URL:** %s
**Descrição:**
%s

## Diff

%s
%s

---

Por favor, revise o diff acima.

Retorne EXCLUSIVAMENTE um único objeto JSON válido (sem markdown, sem crases, sem texto antes ou depois), com exatamente estes campos de texto:

{
  "guia": "Resumo de 2-3 frases explicando o que o MR faz e qual o impacto arquitetural.",
  "diagrama_sequencia": "Corpo PURO de um diagrama Mermaid de sequência (normalmente começa com 'sequenceDiagram'). Apenas a sintaxe do diagrama, SEM crases e SEM a palavra 'mermaid'. Use \"\" (string vazia) se não se aplicar.",
  "diagrama_classes": "Corpo PURO de um diagrama Mermaid de classes/componentes (normalmente começa com 'classDiagram'). Apenas a sintaxe do diagrama, SEM crases e SEM a palavra 'mermaid'. Use \"\" se não se aplicar.",
  "alteracoes": "Markdown com uma tabela | Mudança | Detalhes | Arquivos | resumindo as alterações por arquivo.",
  "pontos_atencao": "Markdown em lista: bugs, vulnerabilidades, race conditions, edge cases não tratados.",
  "sugestoes": "Markdown em lista: refatorações, simplificações, melhor nomenclatura, padrões mais adequados.",
  "riscos": "Markdown em lista: breaking changes, impacto em performance, regressões possíveis.",
  "testes": "Markdown em lista: cenários que deveriam ter cobertura de teste.",
  "conclusao": "Veredicto: ✅ Aprovado | ⚠️ Aprovado com ressalvas | ❌ Solicitar alterações — com justificativa breve.",
  "inline_comments": [
    {
      "path": "caminho/do/arquivo.ext",
      "line": 123,
      "body": "comentário curto e objetivo em português brasileiro"
    }
  ]
}

Regras de formatação (OBRIGATÓRIAS):
- NUNCA use crases triplas em nenhum campo. Os diagramas Mermaid vêm como corpo puro; o sistema adiciona as cercas de código automaticamente.
- Se precisar mostrar um trecho de código dentro de qualquer campo textual, use til triplo (~~~), nunca crases.
- Os campos "diagrama_sequencia" e "diagrama_classes" contêm SOMENTE a sintaxe Mermaid (sem a palavra 'mermaid', sem cercas). Deixe "" quando não houver diagrama relevante.
- Escape quebras de linha dentro das strings JSON como \n.
- Use "" para qualquer seção que não se aplique a este MR.

Regras para "inline_comments":
- Inclua no máximo %d comentários.
- Inclua apenas pontos acionáveis que recomendem mudança ou revisão.
- Use somente linhas adicionadas/modificadas do lado novo do diff, identificadas por marcadores [new:<linha>].
- Em "line", use o número exato do marcador [new:<linha>].
- Em "path", use o caminho do arquivo conforme aparece no cabeçalho "+++ b/path".
- Se não houver um caminho e linha exatos, omita o comentário inline.
- Não duplique no inline todos os pontos do review; priorize os mais importantes.%s`,
		mr.Title,
		mr.ProjectRef,
		mr.WebURL,
		mr.Description,
		diff,
		related,
		maxInlineComments,
		extraGuidance,
	)
}

// filterDiff removes hunks that belong to irrelevant files (lockfiles, generated
// code, build artefacts) so the AI only sees meaningful source changes.
func filterDiff(raw string) string {
	var result strings.Builder
	var skip bool

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			path := extractPath(line)
			skip = isIrrelevant(path)
		}
		if !skip {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}
	return strings.TrimSpace(result.String())
}

func extractPath(line string) string {
	// handles "--- a/path/to/file" and "+++ b/path/to/file"
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ""
	}
	p := parts[1]
	p = strings.TrimPrefix(p, "a/")
	p = strings.TrimPrefix(p, "b/")
	return p
}

func isIrrelevant(path string) bool {
	base := filepath.Base(path)
	for _, pat := range irrelevantPatterns {
		if strings.HasSuffix(pat, "/") {
			if strings.Contains(path, pat) {
				return true
			}
		} else if strings.HasPrefix(pat, ".") {
			if strings.HasSuffix(base, pat) {
				return true
			}
		} else {
			if base == pat || strings.Contains(path, pat) {
				return true
			}
		}
	}
	return false
}

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func annotateDiffLineNumbers(diff string) string {
	if diff == "" {
		return ""
	}

	var result strings.Builder
	var oldLine, newLine int
	inHunk := false

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			oldLine, newLine = parseHunkStart(line)
			inHunk = oldLine > 0 || newLine > 0
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}

		if !inHunk || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			result.WriteString(line)
			result.WriteByte('\n')
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			fmt.Fprintf(&result, "+[new:%d] %s\n", newLine, strings.TrimPrefix(line, "+"))
			newLine++
		case strings.HasPrefix(line, "-"):
			fmt.Fprintf(&result, "-[old:%d] %s\n", oldLine, strings.TrimPrefix(line, "-"))
			oldLine++
		default:
			fmt.Fprintf(&result, " [old:%d new:%d]%s\n", oldLine, newLine, strings.TrimPrefix(line, " "))
			oldLine++
			newLine++
		}
	}

	return strings.TrimSpace(result.String())
}

func parseHunkStart(header string) (oldStart, newStart int) {
	matches := hunkHeaderPattern.FindStringSubmatch(header)
	if len(matches) != 3 {
		return 0, 0
	}
	oldStart, _ = strconv.Atoi(matches[1])
	newStart, _ = strconv.Atoi(matches[2])
	return oldStart, newStart
}
