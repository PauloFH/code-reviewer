# CLAUDE.md — internal/infrastructure/platform

## Papel

Implementações concretas de `domain.PlatformClient` para cada plataforma Git suportada. É o ponto de extensão principal do projeto — adicionar suporte a uma nova plataforma significa criar um novo subpacote aqui.

## Estrutura

```
platform/
├── registry.go      # Implementação de domain.ClientRegistry
├── factory.go       # Cria PlatformClient a partir de config.ClientConfig
├── gitlab/
│   ├── client.go    # API calls: GetMergeRequest, GetMergeRequestDiff, CreateNote
│   └── webhook.go   # ValidateWebhook (X-Gitlab-Token) + ParseWebhook
├── github/
│   ├── client.go    # API calls com Authorization: Bearer
│   └── webhook.go   # ValidateWebhook (HMAC-SHA256) + ParseWebhook
└── gitea/
    ├── client.go    # API calls com Authorization: token
    └── webhook.go   # ValidateWebhook (HMAC-SHA256) + ParseWebhook
```

## Strategy Pattern

Todos os pacotes implementam `domain.PlatformClient`:

```go
type PlatformClient interface {
    Platform() PlatformType          // identidade
    ClientName() string
    WatchedUsername() string
    ValidateWebhook(RawWebhook) bool // valida o secret do webhook
    ParseWebhook(RawWebhook) (*WebhookEvent, error) // normaliza o payload
    GetMergeRequest(projectRef string, mrIID int64) (*MergeRequest, error)
    GetMergeRequestDiff(projectRef string, mrIID int64) (string, error)
    CreateMergeRequestNote(projectRef string, mrIID int64, body string) error
    CreateMergeRequestInlineComment(projectRef string, mr *MergeRequest, comment InlineComment) error
}
```

Comentários inline:
- GitLab usa `POST /merge_requests/:iid/discussions` com `position` e `diff_refs`.
- GitHub usa `POST /pulls/:pull_number/comments` com `commit_id`, `path`, `line`, `side=RIGHT`.
- Gitea atualmente retorna `domain.ErrInlineCommentsUnsupported`.

## `projectRef` por plataforma

| Plataforma | Formato de `projectRef` | Exemplo |
|---|---|---|
| GitLab | ID numérico como string | `"42"` |
| GitHub | `"owner/repo"` | `"acme/backend-api"` |
| Gitea | `"owner/repo"` | `"acme/backend-api"` |

## Autenticação de webhook por plataforma

| Plataforma | Header | Método |
|---|---|---|
| GitLab | `X-Gitlab-Token` | Comparação direta com o secret |
| GitHub | `X-Hub-Signature-256` | HMAC-SHA256 do body com o secret |
| Gitea | `X-Gitea-Signature` | HMAC-SHA256 do body com o secret |

## Self-hosted vs cloud

`gitlab` e `gitlab-selfhosted` usam o mesmo código — a diferença está apenas no `PlatformType` retornado por `Platform()`, que aparece nos logs. Idem para `gitea` vs `gitea-selfhosted`. O `base_url` no `clients.json` define o endpoint real.

## factory.go — ponto de extensão

Para adicionar nova plataforma:
1. Crie o subpacote com `client.go` e `webhook.go`
2. Adicione o `case` no switch de `Build()`
3. Atualize `domain/platform.go` e `config/clients.go`

## registry.go

- Thread-safe via `sync.RWMutex`
- `Register` é chamado apenas no boot (wiring em `main.go`)
- `Get` e `All` são usados em runtime pelos handlers

## Como testar

```bash
go test ./internal/infrastructure/platform/...
```

Os testes de webhook são whitebox (mesmo pacote), testando `ValidateWebhook` e `ParseWebhook` diretamente sem HTTP.
