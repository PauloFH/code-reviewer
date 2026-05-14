package ai

import (
	"fmt"
	"path/filepath"
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
	".pb.go",    // generated protobuf
	"_generated",
}

func buildPrompt(input domain.ReviewInput, outputTemplate string) string {
	mr := input.MergeRequest
	diff := filterDiff(input.Diff)
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n[diff truncado por exceder o limite de tamanho]"
	}

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

Por favor, revise o diff acima e retorne sua análise no seguinte formato:

%s`,
		mr.Title,
		mr.ProjectRef,
		mr.WebURL,
		mr.Description,
		diff,
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
