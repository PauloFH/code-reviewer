package ai

import (
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const systemPrompt = `Você é um revisor sênior de código com ampla experiência em engenharia de software.
Sua tarefa é revisar o diff de um Merge Request de forma objetiva, direta e em português brasileiro.
Não faça comentários genéricos. Foque apenas no que está no diff recebido.
Você responde SEMPRE com um único objeto JSON, com campos de texto separados.
NUNCA escreve crases triplas (` + "```" + `) em nenhum campo: os diagramas Mermaid são
devolvidos como corpo puro e o sistema adiciona as cercas de código automaticamente.`

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

// buildPrompt renders the user message; diff is reviewDiff's output, the same
// text inline comments are later validated against.
func buildPrompt(input domain.ReviewInput, diff, outputTemplate string) string {
	mr := input.MergeRequest

	related := ""
	if input.Related != nil {
		related = renderRelatedContext(input.Related)
	}
	reanalysis := ""
	if input.PreviousReview != "" {
		reanalysis += "\n## Revisão anterior\n\n" + input.PreviousReview + "\n"
	}
	if input.ReviewJustification != "" {
		reanalysis += "\n## Justificativa do solicitante\n\n" + input.ReviewJustification + "\n"
	}

	extraGuidance := ""
	if outputTemplate != "" {
		extraGuidance = "\n\nDiretrizes adicionais de conteúdo (use como ênfase ao preencher os campos acima, sem alterar o formato JSON):\n\n" + outputTemplate
	}

	return fmt.Sprintf(`## Merge Request

**Título:** %s
**Projeto:** %s
**URL:** %s
**Descrição:**
%s

## Diff

%s
%s
%s

---

Por favor, revise o diff acima.

Retorne EXCLUSIVAMENTE um único objeto JSON válido (sem markdown, sem crases, sem texto antes ou depois), com exatamente estes campos de texto:

%s

Regras de formatação (OBRIGATÓRIAS):
- NUNCA use crases triplas em nenhum campo. Os diagramas Mermaid vêm como corpo puro; o sistema adiciona as cercas de código automaticamente.
- Se precisar mostrar um trecho de código dentro de qualquer campo textual, use til triplo (~~~), nunca crases.
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
		annotateDiffLineNumbers(diff),
		related,
		reanalysis,
		reviewJSONSchema(),
		maxInlineComments,
		extraGuidance,
	)
}
