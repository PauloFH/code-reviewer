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
| `AI_BASE_URL` | — | URL base da API de IA (chat/completions) |
| `AI_TOKEN` | — | Token de autenticação da API de IA |
| `AI_MODEL` | `claude-sonnet-4` | Modelo a ser usado |
| `AI_TIMEOUT_SECONDS` | `120` | Timeout para chamadas à IA |
| `DATABASE_PATH` | `./data/reviewer.db` | Caminho do arquivo SQLite |
| `CLIENTS_CONFIG_PATH` | `./clients.json` | Caminho para o arquivo de clientes |
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
│   │   ├── ai/                                # Cliente chat/completions + prompt builder
│   │   ├── logger/                            # slog estruturado em JSON
│   │   ├── platform/
│   │   │   ├── factory.go                     # Cria PlatformClient pelo tipo
│   │   │   ├── registry.go                    # Registro de todos os clients
│   │   │   ├── gitlab/                        # GitLab cloud e self-hosted
│   │   │   ├── github/                        # GitHub cloud e Enterprise
│   │   │   └── gitea/                         # Gitea e Gitea self-hosted
│   │   └── storage/                           # SQLite repository
│   ├── transport/http/                        # Handler, middleware, router
│   └── usecase/                               # Pipeline de review (regra de negócio)
├── migrations/                                # SQL de referência
├── test/fixtures/                             # Payloads de exemplo por plataforma
├── clients.example.json                       # Modelo de clients.json
├── .env.example                               # Modelo de .env
├── Dockerfile                                 # Multi-stage build com CGo
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

## Limitações conhecidas

- **SQLite single-writer:** o banco usa WAL mode com `MaxOpenConns=1`. Para alta concorrência de webhooks, considere migrar para PostgreSQL implementando a interface `domain.ReviewRepository`.
- **Diff grande:** diffs maiores que 80 KB são truncados antes de enviar para a IA. Arquivos gerados, lockfiles e artefatos de build são filtrados automaticamente.
- **Retry de webhook:** se o serviço retornar `5xx`, a plataforma pode reenviar o webhook. A idempotência via SQLite previne comentários duplicados na maioria dos casos.
- **Sem suporte a GitLab Enterprise features:** o cliente GitLab usa apenas a API v4 pública.
- **Sem autenticação no `/healthz`:** público por design para facilitar probes de infraestrutura.
