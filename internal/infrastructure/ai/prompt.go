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
Não faça comentários genéricos. Foque apenas no que está no diff recebido.`

const defaultOutputFormat = `## Guia do Revisor

(Resumo de 2-3 frases explicando o que o MR faz e qual o impacto arquitetural)

### Diagrama de sequência

(Se aplicável — diagrama Mermaid mostrando o fluxo principal introduzido ou modificado)

### Diagrama de classes/componentes

(Se aplicável — diagrama Mermaid mostrando novos tipos, interfaces ou relações entre módulos)

### Alterações por arquivo

| Mudança | Detalhes | Arquivos |
|---------|----------|----------|
| (descrição curta) | - detalhe 1<br>- detalhe 2 | path/to/file.go |

### Pontos de atenção
- (bugs, vulnerabilidades, race conditions, edge cases não tratados)

### Sugestões de melhoria
- (refatorações, simplificações, melhor nomenclatura, padrões mais adequados)

### Riscos
- (breaking changes, impacto em performance, regressões possíveis)

### Testes sugeridos
- (cenários que deveriam ter cobertura de teste)

### Conclusão

(Veredicto: ✅ Aprovado | ⚠️ Aprovado com ressalvas | ❌ Solicitar alterações — com justificativa breve)`

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

func buildPrompt(input domain.ReviewInput, outputTemplate string) string {
	mr := input.MergeRequest
	diff := filterDiff(input.Diff)
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n[diff truncado por exceder o limite de tamanho]"
	}
	diff = annotateDiffLineNumbers(diff)

	outputFmt := defaultOutputFormat
	if outputTemplate != "" {
		outputFmt = outputTemplate
	}

	return fmt.Sprintf(`## Merge Request

**Título:** %s
**Plataforma:** %s
**URL:** %s
**Descrição:**
%s

## Diff

%s

---

Por favor, revise o diff acima.

Retorne exclusivamente um objeto JSON válido, sem markdown fences e sem texto antes ou depois, neste formato:

{
  "review_markdown": "review completo em Markdown, seguindo o template informado",
  "inline_comments": [
    {
      "path": "caminho/do/arquivo.ext",
      "line": 123,
      "body": "comentário curto e objetivo em português brasileiro"
    }
  ]
}

Regras para "inline_comments":
- Inclua no máximo %d comentários.
- Inclua apenas pontos acionáveis que recomendem mudança ou revisão.
- Use somente linhas adicionadas/modificadas do lado novo do diff, identificadas por marcadores [new:<linha>].
- Em "line", use o número exato do marcador [new:<linha>].
- Em "path", use o caminho do arquivo conforme aparece no cabeçalho "+++ b/path".
- Se não houver um caminho e linha exatos, omita o comentário inline.
- Não duplique no inline todos os pontos do review; priorize os mais importantes.

Template do campo "review_markdown":

%s`,
		mr.Title,
		mr.ProjectRef,
		mr.WebURL,
		mr.Description,
		diff,
		maxInlineComments,
		outputFmt,
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
