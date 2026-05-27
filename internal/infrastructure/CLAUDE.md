# CLAUDE.md — internal/infrastructure

## Papel

Implementações concretas dos contratos definidos em `domain`. Esta camada conhece tecnologias específicas: HTTP, SQLite, JSON, HMAC. A camada de usecase nunca importa daqui diretamente — só via interfaces.

## Subpacotes

### `platform/`
Implementações de `domain.PlatformClient` por plataforma Git. Veja `platform/CLAUDE.md` para detalhes.

### `ai/`
Resolve a configuração de IA e despacha o review para o transporte do provider.

**`resolver.go`** — `ResolvingReviewer` implementa `domain.AIReviewer`:
- `ResolveConfig(clientName, projectRef)` pega a `AIConfig` do escopo mais específico; se nenhuma, usa o `envClient` (fallback de env).
- Modo `account`: `ensureValidToken` faz refresh do token OAuth quando ausente/perto de expirar (margem de 1 min) e persiste via `UpdateOAuthTokens`.
- Constrói um `Client` por config e anexa os preços da config ao `ReviewResult`.
- `TestConfig(id)` roda um review de um diff de exemplo embutido contra uma config (usado pelo endpoint de teste).

**`client.go`** — `Client` (executor por config) escolhe o transporte:
- `transportKindForProvider`: gemini → gemini; claude → anthropic; openai/deepseek → chat_completions.
- `baseURLForConfig`/`modelForConfig`: defaults por provider, sobrescrevíveis pela config.
- `New(...)` é o construtor legado usado pelo fallback de env.
- `parseReviewPayload` extrai `review_markdown` + `inline_comments`; sem JSON válido, usa a resposta inteira.

**Transportes** (interface interna `transport`):
- **`gemini.go`** — `POST /models/{model}:generateContent`, header `X-goog-api-key`, `generationConfig.maxOutputTokens` opcional.
- **`chat_completions.go`** — `POST /chat/completions`, `Authorization: Bearer`, `max_tokens` opcional (OpenAI/DeepSeek).
- **`anthropic.go`** — API nativa `POST /v1/messages`. Modo api: header `x-api-key`. Modo account: `Authorization: Bearer` + `anthropic-beta: oauth-2025-04-20` + primeiro bloco system com a identidade "Claude Code" (exigida pelos tokens OAuth). `max_tokens` default 4096 (configurável). **Sem `temperature`** (deprecado no modelo de assinatura).

**`oauth_claude.go`** — `ClaudeOAuth`: fluxo OAuth estilo Claude Code (PKCE S256). `BuildAuthorizeURL`, `Exchange(state, code)` e `Refresh`. Verifiers pendentes ficam em memória, com TTL. Parâmetros públicos (client_id/endpoints) são área cinzenta — ver [[ai-account-oauth-validated]] na memória.

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
Implementa `domain.ReviewRepository`, `ClientRepository`, `ReviewEventRepository`, `PromptRepository` e `AIConfigRepository` usando SQLite.

**`sqlite_repository.go`**:
- `New(path)` — abre/cria o banco e roda as migrations versionadas (`migration001`..`migration006`)
- `AlreadyReviewed(...)` / `MarkAsReviewed(...)` — idempotência (`SELECT COUNT` / `INSERT OR IGNORE`)
- `UseEncryptionKey(base64Key)` — configura o encryptor para as credenciais de IA
- WAL mode + `MaxOpenConns=1` — SQLite não suporta múltiplos writers simultâneos

**`ai_config_repository.go`** — CRUD de `ai_configs`/`ai_config_assignments`, `ResolveConfig` (cascata projeto → client → global, igual aos prompts) e `UpdateOAuthTokens`. Cifra os campos sensíveis no write e decifra no read via `encryptField`/`decryptField`.

**`crypto.go`** — `encryptor` AES-256-GCM. `newEncryptor(base64Key)` (chave de 32 bytes), `encrypt`/`decrypt` no formato `base64(nonce || ciphertext)`. Sem chave configurada, ler/gravar segredo retorna `errNoEncryptionKey` — daí o boot faz fail-fast em `ListConfigs`. Ver [[ai-config-enc-key-must-be-stable]] na memória.

**`migrations.go`** — `migration005` cria `ai_configs` + `ai_config_assignments`; `migration006` adiciona a coluna `max_tokens`.

### `logger/`
**`logger.go`** — retorna `*slog.Logger` configurado com `slog.NewJSONHandler` para stdout. Nível configurável via `LOG_LEVEL`. Sem dependências externas.

## Regras desta camada

- Pode importar o pacote `domain` (para implementar as interfaces)
- Pode importar `config` (para receber configurações)
- **Não pode** ser importado pelo `usecase` ou `domain` diretamente
- Toda comunicação com o usecase se dá via interfaces injetadas no `main.go`
