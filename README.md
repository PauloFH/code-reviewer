# Code Reviewer

Serviço em Go que monitora Merge Requests / Pull Requests em múltiplas plataformas Git e publica automaticamente um code review gerado por IA como comentário no próprio MR/PR.

Suporta **GitLab**, **GitLab self-hosted**, **GitHub** e **Gitea** (incluindo instâncias self-hosted) simultaneamente, via Strategy Pattern — cada plataforma é um cliente independente registrado em `clients.json`.

---

## Como funciona

```
Plataforma Git → POST /webhook/{clientName} → validação de secret
    → parse do payload → busca MR + diff via API
    → envia diff para IA → posta comentário no MR
    → persiste registro (SQLite) para garantir idempotência
```

O review é gerado **uma única vez por MR**, quando o usuário configurado (`username`) é marcado como reviewer ou assignee pela primeira vez. Reenvios de webhook, novos commits, alterações de título ou reinicializações do serviço não disparam um segundo review.

Além do comentário completo, o serviço tenta publicar comentários inline no diff para pontos acionáveis apontados pela IA. O suporte inline é best-effort: GitLab e GitHub usam comentários nativos no diff; Gitea continua recebendo apenas o comentário completo.

---

## Pré-requisitos

- Go 1.22+ (com CGo habilitado, necessário para SQLite)
- GCC / musl-dev (Linux) ou Xcode CLT (macOS) — para o driver SQLite
- Docker + Docker Compose (para execução containerizada)

---

## Configuração

### 1. Variáveis de ambiente

Copie o arquivo de exemplo e preencha:

```bash
cp .env.example .env
```

| Variável | Padrão | Descrição |
|---|---|---|
| `APP_PORT` | `8080` | Porta HTTP |
| `APP_ENV` | `development` | `development` ou `production` |
| `AI_PROVIDER` | inferido | Fallback de IA: `gemini` ou `chat_completions` (usado quando nenhuma config do banco resolve) |
| `AI_CONFIG_ENC_KEY` | — | Base64 de 32 bytes; cifra credenciais de IA configuradas pelo frontend. Obrigatória só quando há configs persistidas |
| `GEMINI_API_KEY` | — | Chave da API Gemini |
| `GEMINI_MODEL` | `gemini-flash-latest` | Modelo Gemini a ser usado |
| `GEMINI_BASE_URL` | `https://generativelanguage.googleapis.com/v1beta` | URL base da API Gemini |
| `AI_BASE_URL` | — | URL base para provider `chat_completions` |
| `AI_TOKEN` | — | Token para provider `chat_completions` |
| `AI_MODEL` | `claude-sonnet-4` | Modelo para provider `chat_completions` |
| `AI_TIMEOUT_SECONDS` | `120` | Timeout para chamadas à IA |
| `AI_PRICE_PER_M_INPUT_TOKENS` | `3.0` | Preço por milhão de tokens de entrada (USD) |
| `AI_PRICE_PER_M_OUTPUT_TOKENS` | `15.0` | Preço por milhão de tokens de saída (USD) |
| `DATABASE_PATH` | `./data/reviewer.db` | Caminho do arquivo SQLite |
| `CLIENTS_CONFIG_PATH` | `./clients.json` | Caminho para o arquivo de clientes |
| `DASHBOARD_USER` | `admin` | Usuário do dashboard |
| `DASHBOARD_PASSWORD` | — | Senha do dashboard (obrigatória para login) |
| `DASHBOARD_JWT_SECRET` | — | Segredo para assinar JWTs (recomendado: 32+ chars aleatórios) |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |

### 2. Arquivo de clientes (`clients.json`)

Cada entrada representa uma instância de plataforma monitorada. Múltiplas entradas permitem monitorar GitLab e GitHub ao mesmo tempo, ou múltiplas instâncias da mesma plataforma.

```bash
cp clients.example.json clients.json
# edite clients.json com seus dados reais
```

Formato de cada entrada:

```json
{
  "name": "identificador-unico",
  "platform": "gitlab | gitlab-selfhosted | github | gitea | gitea-selfhosted",
  "base_url": "URL base da API",
  "token": "token de acesso à API",
  "webhook_secret": "segredo configurado no webhook da plataforma",
  "username": "usuário a ser monitorado"
}
```

O campo `name` é usado na URL do webhook: `POST /webhook/{name}`.

---

## Rodando localmente

```bash
# 1. Instale dependências
go mod download

# 2. Configure o ambiente
cp .env.example .env && cp clients.example.json clients.json
# edite os dois arquivos com seus dados reais

# 3. Crie o diretório de dados
mkdir -p data

# 4. Inicie o servidor
go run ./cmd/server
```

O servidor estará disponível em `http://localhost:8080`.

Para verificar que está rodando:

```bash
curl http://localhost:8080/healthz
# → 200 OK
```

---

## Dashboard de monitoramento

Acesse `http://localhost:8080` para abrir o dashboard Vue 3.

### Login

Defina as variáveis de ambiente e faça login:

```
DASHBOARD_USER=admin
DASHBOARD_PASSWORD=sua-senha
DASHBOARD_JWT_SECRET=segredo-longo-aleatório
```

O dashboard inclui:
- **Dashboard** — stats globais, custo diário (gráfico 30 dias), breakdown por cliente, live feed SSE
- **Events** — tabela paginada com filtros por cliente e período; tokens + custo por review
- **Clients** — CRUD completo de plataformas monitoradas sem precisar editar arquivos
- **Prompts** — biblioteca de prompts reutilizáveis com editor rich text (Tiptap) para system prompt e template de saída; atribuição hierárquica (global → client → projeto)
- **IA** — configuração de provedores de IA (OpenAI, Claude, Gemini, DeepSeek) em dois modos (chave de API ou login de conta Claude via OAuth); atribuição hierárquica e botão "Testar" que roda um diff de exemplo

### Usando a API REST

```bash
# Login → JWT
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"sua-senha"}' | jq -r .token)

# Stats globais
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/events/stats

# Custo diário (últimos 30 dias)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/events/daily

# Eventos paginados
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/events?limit=25&offset=0"

# Live feed SSE
curl -N "http://localhost:8080/api/v1/events/stream?token=$TOKEN"
```

### Endpoints da API

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/v1/auth/login` | Retorna JWT |
| `GET` | `/api/v1/events` | Lista eventos (paginado; filtros: `client`, `since`, `until`) |
| `GET` | `/api/v1/events/stats` | Agregados globais e por cliente |
| `GET` | `/api/v1/events/daily` | Custo diário (`?days=30`, `?client=nome`) |
| `GET` | `/api/v1/events/stream` | Live feed SSE (`?token=JWT`) |
| `GET` | `/api/v1/clients` | Lista clients (token/secret redacted) |
| `POST` | `/api/v1/clients` | Cria client |
| `GET` | `/api/v1/clients/{name}` | Detalhe de um client |
| `PUT` | `/api/v1/clients/{name}` | Atualiza client (suporta rename) |
| `DELETE` | `/api/v1/clients/{name}` | Remove client |
| `GET` | `/api/v1/prompts` | Lista prompts |
| `POST` | `/api/v1/prompts` | Cria prompt |
| `GET` | `/api/v1/prompts/{id}` | Busca prompt por ID |
| `PUT` | `/api/v1/prompts/{id}` | Atualiza prompt |
| `DELETE` | `/api/v1/prompts/{id}` | Remove prompt |
| `GET` | `/api/v1/prompts/assignments` | Lista atribuições |
| `PUT` | `/api/v1/prompts/assignments` | Cria/atualiza atribuição (upsert) |
| `DELETE` | `/api/v1/prompts/assignments` | Remove atribuição |
| `GET` | `/api/v1/ai-configs` | Lista configs de IA (segredos mascarados) |
| `POST` | `/api/v1/ai-configs` | Cria config de IA |
| `GET/PUT/DELETE` | `/api/v1/ai-configs/{id}` | Lê / atualiza / remove config |
| `POST` | `/api/v1/ai-configs/{id}/test` | Roda um review de diff de exemplo na config |
| `GET/PUT/DELETE` | `/api/v1/ai-configs/assignments` | Atribuições de config (global → client → projeto) |
| `POST` | `/api/v1/ai-configs/oauth/claude/start` | Inicia OAuth (retorna authorize_url + state) |
| `POST` | `/api/v1/ai-configs/oauth/claude/exchange` | Troca código por tokens e persiste |

Rotas de `clients`, `prompts` e `ai-configs` exigem JWT + papel `super`/`gestor`; `users` exige `super`.

---

## Rodando com Docker

```bash
# Build e start
docker compose up --build

# Em background
docker compose up --build -d

# Logs
docker compose logs -f

# Parar
docker compose down
```

O banco SQLite é persistido em `./data/reviewer.db` via volume.

---

## Configurando webhooks nas plataformas

### GitLab

1. Acesse **Settings → Webhooks** no projeto.
2. Configure:
   - **URL:** `https://seu-servidor.com/webhook/nome-do-client`
   - **Secret token:** valor de `webhook_secret` no `clients.json`
   - **Trigger:** marque apenas **Merge request events**
3. Clique em **Add webhook**.

### GitHub

1. Acesse **Settings → Webhooks** no repositório.
2. Configure:
   - **Payload URL:** `https://seu-servidor.com/webhook/nome-do-client`
   - **Content type:** `application/json`
   - **Secret:** valor de `webhook_secret` no `clients.json`
   - **Events:** selecione **Let me select individual events** → marque **Pull requests**
3. Clique em **Add webhook**.

### Gitea

1. Acesse **Settings → Webhooks** no repositório.
2. Configure:
   - **Target URL:** `https://seu-servidor.com/webhook/nome-do-client`
   - **Secret:** valor de `webhook_secret` no `clients.json`
   - **Trigger On:** Pull Request
3. Clique em **Add Webhook**.

> **Importante:** O servidor precisa ser acessível publicamente para receber os webhooks. Em desenvolvimento, use [ngrok](https://ngrok.com) ou [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/).

---

## Exemplo de comentário gerado

```markdown
## Revisão automática do Merge Request

### Resumo
...

### Pontos de atenção
- ...

### Possíveis melhorias
- ...

### Riscos encontrados
- ...

### Testes sugeridos
- ...

### Conclusão
...
```

---

## Respostas HTTP do webhook

| Status | Significado |
|---|---|
| `200 OK` | Evento processado ou ignorado intencionalmente |
| `401 Unauthorized` | Secret de webhook inválido |
| `404 Not Found` | Client name não registrado |
| `500 Internal Server Error` | Erro ao chamar API da plataforma ou da IA |

---

## Rodando os testes

```bash
# Todos os testes
go test ./...

# Com output detalhado
go test ./... -v

# Com cobertura
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Estrutura do projeto

```
.
├── cmd/server/main.go                         # Entry point e wiring
├── internal/
│   ├── config/                                # Leitura de env vars e clients.json
│   ├── domain/                                # Entidades e interfaces (Strategy)
│   ├── infrastructure/
│   │   ├── ai/                                # Resolver + transportes (gemini/chat/anthropic) + OAuth + prompt builder
│   │   ├── logger/                            # slog estruturado em JSON
│   │   ├── platform/
│   │   │   ├── factory.go                     # Cria PlatformClient pelo tipo
│   │   │   ├── registry.go                    # Registro de todos os clients
│   │   │   ├── gitlab/                        # GitLab cloud e self-hosted
│   │   │   ├── github/                        # GitHub cloud e Enterprise
│   │   │   └── gitea/                         # Gitea e Gitea self-hosted
│   │   └── storage/                           # SQLite repository (migrations versionadas)
│   ├── transport/http/                        # Webhooks, API REST, SSE, SPA embed
│   └── usecase/                               # Pipeline de review + CRUD de clients + custo
├── web/                                       # Frontend Vue 3 + TypeScript + Vite
│   └── src/
│       ├── views/                             # Login, Dashboard, Events, Clients, Prompts, AiConfig
│       ├── components/                        # StatsCards, CostChart, LiveFeed, RichTextEditor, AiConfigForm, etc.
│       ├── stores/                            # Pinia: auth, toast
│       └── api/                               # Axios: events, clients, auth, prompts, aiConfigs
├── clients.example.json                       # Modelo de clients.json
├── .env.example                               # Modelo de .env
├── Dockerfile                                 # Multi-stage: Node (Vite) + Go (CGo)
└── docker-compose.yml
```

---

## Adicionando uma nova plataforma

1. Crie o pacote `internal/infrastructure/platform/<nome>/`.
2. Implemente `client.go` satisfazendo `domain.PlatformClient`.
3. Implemente `webhook.go` com `ValidateWebhook` e `ParseWebhook`.
4. Adicione a constante em `domain/platform.go`.
5. Adicione o `case` em `infrastructure/platform/factory.go`.
6. Atualize a validação em `config/clients.go`.

Nenhuma outra parte do código precisa ser alterada.

---

## Sistema de Prompts

O system prompt e o template de saída do review são configuráveis via dashboard, com hierarquia de prioridade:

```
Projeto específico  >  Client (provedor Git)  >  Global
```

### Como funciona

1. Acesse a aba **Prompts** no dashboard.
2. Crie um prompt com nome, system prompt e template de saída usando o editor rich text.
3. Atribua o prompt a um escopo:
   - **Global** — usado como padrão quando nenhum prompt mais específico está configurado
   - **Client** — vale para todos os repositórios de um provedor Git (ex: "gitlab-cloud")
   - **Projeto** — vale para um repositório específico (ex: "gitlab-cloud" + "org/meu-repo")

Quando um webhook chega, o sistema resolve o prompt mais específico disponível. Se nenhum prompt estiver configurado, usa o prompt padrão hardcoded.

Um mesmo prompt pode ser reutilizado em múltiplas atribuições — funciona como uma biblioteca.

---

## Configuração de IA

Os provedores de IA são configurados pela aba **IA** do dashboard, com a mesma hierarquia dos prompts (projeto > client > global). Se nenhuma config for resolvida, o serviço usa o **fallback por env** (`AI_PROVIDER`).

Dois modos:

- **API (chave):** `openai` e `deepseek` (via `/chat/completions`), `gemini` (`generateContent`) e `claude` (API nativa `/v1/messages`). Cada provider tem defaults de base URL e modelo, sobrescrevíveis por config.
- **Conta (OAuth):** login na assinatura **Claude** (fluxo estilo Claude Code — PKCE + colagem manual do código). O access token expira e é **renovado automaticamente** via refresh token antes de cada review; não é preciso reconectar.

Cada config também aceita **preço por milhão de tokens** (para o cálculo de custo) e **máximo de tokens de saída** (0 = padrão do provedor; Claude usa 4096).

### Segurança das credenciais

Chaves de API e tokens OAuth são cifrados em repouso com **AES-256-GCM**, usando a chave-mestra `AI_CONFIG_ENC_KEY` (base64 de 32 bytes). A API nunca devolve os segredos — só flags `has_api_key` / `account_connected`. Mantenha a chave estável: trocá-la torna os segredos gravados ilegíveis e o boot falha rápido ao tentar lê-los.

### Testando uma config

O botão **Testar** (ou `POST /api/v1/ai-configs/{id}/test`) roda um review de um diff de exemplo contra a config — valida credencial, refresh de token e geração, sem precisar de um webhook real.

---

## Limitações conhecidas

- **SQLite single-writer:** o banco usa WAL mode com `MaxOpenConns=1`. Para alta concorrência de webhooks, considere migrar para PostgreSQL implementando a interface `domain.ReviewRepository`.
- **Diff grande:** diffs maiores que 80 KB são truncados antes de enviar para a IA. Arquivos gerados, lockfiles e artefatos de build são filtrados automaticamente.
- **Retry de webhook:** se o serviço retornar `5xx`, a plataforma pode reenviar o webhook. A idempotência via SQLite previne comentários duplicados na maioria dos casos.
- **Sem suporte a GitLab Enterprise features:** o cliente GitLab usa apenas a API v4 pública.
- **Sem autenticação no `/healthz`:** público por design para facilitar probes de infraestrutura.
