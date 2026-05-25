# CLAUDE.md — internal/infrastructure

## Papel

Implementações concretas dos contratos definidos em `domain`. Esta camada conhece tecnologias específicas: HTTP, SQLite, JSON, HMAC. A camada de usecase nunca importa daqui diretamente — só via interfaces.

## Subpacotes

### `platform/`
Implementações de `domain.PlatformClient` por plataforma Git. Veja `platform/CLAUDE.md` para detalhes.

### `ai/`
Cliente HTTP configurável para `gemini` ou `chat_completions`.

**`client.go`** — implementa `domain.AIReviewer`:
- Provider `gemini`: envia `POST /models/{model}:generateContent` com `systemInstruction`, `contents` e `generationConfig.temperature: 0.2`; usa header `X-goog-api-key`.
- Provider `chat_completions`: envia `POST /chat/completions` com `model`, `messages` e `temperature: 0.2`; usa header `Authorization: Bearer`.
- Retorna erro descritivo em status >= 300, prompt bloqueado ou response vazia.

**`prompt.go`** — monta o prompt e filtra o diff:
- `buildPrompt(input, outputTemplate)` — formata título, URL, descrição e diff em português; usa `outputTemplate` se fornecido, senão `defaultOutputFormat`
- `filterDiff(raw)` — remove hunks de arquivos irrelevantes (lockfiles, dist/, vendor/, *.min.js, etc.)
- `isIrrelevant(path)` — classifica um arquivo como relevante ou não
- `maxDiffBytes = 80_000` — limite de truncagem do diff
- `defaultOutputFormat` — formato padrão de saída (Resumo, Pontos de atenção, etc.)

**`html_to_text.go`** — converte HTML do editor Tiptap para texto plano:
- `htmlToPlainText(html)` — substitui tags de bloco por newlines, lista items por `- `, strip demais tags, decode entidades HTML

Para mudar a lista de arquivos ignorados, edite `irrelevantPatterns` em `prompt.go`.

### `storage/`
Implementa `domain.ReviewRepository` usando SQLite.

**`sqlite_repository.go`**:
- `New(path)` — abre/cria o banco e roda a migration inline
- `AlreadyReviewed(clientName, projectRef, mrIID, username)` — `SELECT COUNT`
- `MarkAsReviewed(...)` — `INSERT OR IGNORE` (safe em concorrência)
- WAL mode + `MaxOpenConns=1` — SQLite não suporta múltiplos writers simultâneos

### `logger/`
**`logger.go`** — retorna `*slog.Logger` configurado com `slog.NewJSONHandler` para stdout. Nível configurável via `LOG_LEVEL`. Sem dependências externas.

## Regras desta camada

- Pode importar o pacote `domain` (para implementar as interfaces)
- Pode importar `config` (para receber configurações)
- **Não pode** ser importado pelo `usecase` ou `domain` diretamente
- Toda comunicação com o usecase se dá via interfaces injetadas no `main.go`
