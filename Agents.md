# AGENTS.md — GitLab MR AI Reviewer

## Objetivo

Criar um serviço em Go que monitora Merge Requests do GitLab onde o usuário foi marcado como reviewer/assignee ou mencionado, lê o diff do MR, envia o conteúdo para um provider de IA configurável (Gemini ou API compatível com chat/completions) e publica automaticamente um comentário de code review inicial no próprio Merge Request.

O serviço deve ser bem estruturado, seguindo Clean Architecture, boas práticas de engenharia de software, código limpo, baixo acoplamento, alta testabilidade e configuração por variáveis de ambiente.

---

## Stack obrigatória

- Linguagem: Go
- HTTP server nativo ou Gin/Fiber, preferencialmente `net/http` para simplicidade
- Estrutura baseada em Clean Architecture
- Configuração via environment variables
- Cliente HTTP próprio para GitLab
- Cliente HTTP próprio para API de IA
- Logs estruturados
- Testes unitários nas regras principais
- Dockerfile
- docker-compose para execução local

---

## Funcionalidade principal

O serviço deve receber webhooks do GitLab relacionados a Merge Requests.

Quando um MR for criado ou atualizado, o serviço deve verificar se o usuário configurado foi marcado pela primeira vez no MR.

Se sim:

1. Validar o webhook recebido.
2. Identificar o projeto e o merge request.
3. Buscar detalhes completos do MR via API do GitLab.
4. Buscar o diff completo do MR.
5. Montar um prompt de code review.
6. Enviar o prompt para a API de IA.
7. Receber a resposta.
8. Publicar um comentário no MR com o review gerado.
9. Registrar que aquele MR já foi revisado para evitar comentários duplicados.

---

## Regra importante de disparo

O review automático deve acontecer somente uma vez por Merge Request, quando o usuário for marcado pela primeira vez.

Deve funcionar nos dois casos:

1. MR criado já marcando o usuário.
2. MR já existente onde o usuário foi marcado depois.

Não deve rodar novamente se:

- o MR recebeu novos commits;
- o título foi alterado;
- a descrição foi alterada;
- outro usuário foi marcado;
- o webhook foi reenviado;
- o serviço reiniciou.

Para isso, implementar persistência de estado.

---

## Persistência

Criar uma camada de storage para registrar MRs já processados.

Começar com SQLite por simplicidade.

Tabela sugerida:

```sql
CREATE TABLE reviewed_merge_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gitlab_project_id INTEGER NOT NULL,
    gitlab_merge_request_iid INTEGER NOT NULL,
    gitlab_merge_request_id INTEGER,
    reviewed_user TEXT NOT NULL,
    reviewed_at DATETIME NOT NULL,
    UNIQUE(gitlab_project_id, gitlab_merge_request_iid, reviewed_user)
);
```

---

## Variáveis de ambiente

```env
APP_PORT=8080
APP_ENV=development

GITLAB_BASE_URL=https://gitlab.com
GITLAB_TOKEN=seu_token_gitlab
GITLAB_WEBHOOK_SECRET=segredo_webhook
GITLAB_USERNAME=paulorfholanda

AI_PROVIDER=gemini
GEMINI_API_KEY=sua_chave_gemini
GEMINI_MODEL=gemini-flash-latest
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
AI_TIMEOUT_SECONDS=120

DATABASE_PATH=./data/reviewer.db

LOG_LEVEL=info
```

---

## Estrutura de pastas esperada

```txt
gitlab-ai-reviewer/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── review_service.go
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── merge_request.go
│   │   ├── review.go
│   │   └── repository.go
│   ├── infrastructure/
│   │   ├── gitlab/
│   │   │   ├── client.go
│   │   │   ├── webhook.go
│   │   │   └── mapper.go
│   │   ├── ai/
│   │   │   ├── client.go
│   │   │   └── prompt.go
│   │   ├── storage/
│   │   │   └── sqlite_repository.go
│   │   └── logger/
│   │       └── logger.go
│   ├── transport/
│   │   └── http/
│   │       ├── handler.go
│   │       ├── middleware.go
│   │       └── router.go
│   └── usecase/
│       └── process_merge_request.go
├── migrations/
│   └── 001_create_reviewed_merge_requests.sql
├── test/
│   └── fixtures/
│       └── gitlab_merge_request_webhook.json
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

---

## Clean Architecture

Separar responsabilidades:

### Domain

Contém entidades e contratos.

Não deve depender de GitLab, HTTP, banco ou IA.

### Usecase

Contém a regra principal:

- receber evento;
- verificar se o usuário foi marcado;
- verificar se já foi revisado;
- buscar diff;
- gerar review;
- publicar comentário;
- salvar estado.

### Infrastructure

Implementações externas:

- GitLab API client;
- AI API client;
- SQLite repository;
- logger.

### Transport

Camada HTTP:

- recebe webhook;
- valida secret;
- transforma payload em comando interno;
- chama usecase.

---

## Cliente GitLab

Implementar métodos:

```go
type GitLabClient interface {
    GetMergeRequest(projectID int64, mrIID int64) (*MergeRequest, error)
    GetMergeRequestDiff(projectID int64, mrIID int64) (string, error)
    CreateMergeRequestNote(projectID int64, mrIID int64, body string) error
}
```

Endpoints esperados:

```txt
GET /api/v4/projects/:id/merge_requests/:merge_request_iid
GET /api/v4/projects/:id/merge_requests/:merge_request_iid/changes
POST /api/v4/projects/:id/merge_requests/:merge_request_iid/notes
```

Autenticação:

```txt
PRIVATE-TOKEN: <GITLAB_TOKEN>
```

---

## Cliente de IA

Criar uma interface genérica:

```go
type AIReviewer interface {
    ReviewDiff(input ReviewInput) (*ReviewResult, error)
}
```

A implementação deve usar:

```env
AI_PROVIDER
GEMINI_API_KEY
GEMINI_MODEL
GEMINI_BASE_URL
AI_BASE_URL
AI_TOKEN
AI_MODEL
```

Assumir API HTTP JSON compatível com Gemini `generateContent` quando `AI_PROVIDER=gemini`, ou `chat/completions` quando `AI_PROVIDER=chat_completions`.

Payload sugerido para Gemini:

```json
{
  "systemInstruction": {
    "parts": [
      {
        "text": "Você é um revisor sênior de código..."
      }
    ]
  },
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "Revise o seguinte diff..."
        }
      ]
    }
  ],
  "generationConfig": {
    "temperature": 0.2
  }
}
```

Headers:

```txt
X-goog-api-key: <GEMINI_API_KEY>
Content-Type: application/json
```

Para `chat_completions`, usar payload `{ "model": "...", "messages": [...] }` e header `Authorization: Bearer <AI_TOKEN>`.

---

## Prompt base do review

O prompt deve pedir:

- análise objetiva;
- identificação da stack usada;
- riscos de bug;
- problemas de segurança;
- problemas de performance;
- legibilidade;
- arquitetura;
- testes ausentes;
- possíveis regressões;
- sugestões práticas;
- evitar comentários genéricos;
- comentar apenas sobre o diff recebido;
- retornar em português brasileiro.

Formato da resposta:

```md
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

## Regras para diff grande

Se o diff for muito grande:

1. limitar por tamanho configurável;
2. separar arquivos relevantes;
3. ignorar arquivos binários, lockfiles e artefatos gerados;
4. priorizar código-fonte.

Arquivos a ignorar por padrão:

```txt
package-lock.json
pnpm-lock.yaml
yarn.lock
go.sum
dist/
build/
target/
node_modules/
vendor/
*.min.js
*.map
```

---

## Validação do webhook

Validar o header:

```txt
X-Gitlab-Token
```

Comparar com:

```env
GITLAB_WEBHOOK_SECRET
```

Se inválido, retornar `401`.

---

## Idempotência

Antes de gerar review:

```go
exists := repository.AlreadyReviewed(projectID, mrIID, username)
```

Se já existir, retornar sem postar comentário.

Depois de postar comentário com sucesso:

```go
repository.MarkAsReviewed(projectID, mrIID, username)
```

---

## Logs

Usar logs estruturados.

Eventos importantes:

- webhook recebido;
- webhook ignorado;
- usuário marcado;
- MR já revisado;
- buscando diff;
- enviando para IA;
- comentário publicado;
- erro ao chamar GitLab;
- erro ao chamar IA;
- erro no banco.

Não logar tokens.

---

## Testes obrigatórios

Criar testes para:

- detectar se usuário foi marcado no MR;
- ignorar MR já revisado;
- montar prompt corretamente;
- ignorar arquivos irrelevantes no diff;
- validar webhook secret;
- tratar erro da API de IA;
- tratar erro da API do GitLab;
- garantir idempotência.

---

## README

O README deve conter:

- objetivo do projeto;
- como configurar `.env`;
- como rodar localmente;
- como configurar webhook no GitLab;
- exemplo de payload;
- como rodar com Docker;
- como rodar testes;
- limitações conhecidas.

---

## Docker

Criar Dockerfile multi-stage.

Exemplo esperado:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gitlab-ai-reviewer ./cmd/server

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/gitlab-ai-reviewer .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./gitlab-ai-reviewer"]
```

---

## docker-compose

Criar compose com volume para SQLite:

```yaml
services:
  gitlab-ai-reviewer:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    volumes:
      - ./data:/app/data
```

---

## Critérios de aceite

O projeto estará pronto quando:

- receber webhook do GitLab;
- validar o secret;
- identificar MR onde `GITLAB_USERNAME` foi marcado;
- buscar diff via GitLab API;
- enviar diff para API de IA;
- receber review;
- comentar no MR;
- não comentar duas vezes no mesmo MR;
- possuir testes unitários;
- possuir README;
- rodar via Docker;
- não expor tokens em logs;
- seguir estrutura limpa e modular.

---

## Observação importante

Não implementar lógica acoplada diretamente no handler HTTP.

O handler deve apenas validar entrada e chamar o usecase.

Toda regra de negócio deve ficar no usecase.

Toda integração externa deve ficar em infrastructure.
