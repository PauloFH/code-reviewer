# CLAUDE.md — internal/config

## Papel

Lê toda a configuração do serviço e a expõe em structs tipadas. Nenhuma outra camada acessa `os.Getenv` diretamente.

## Arquivos

### `config.go`
- `Load()` — lê env vars e retorna `*Config`
- `requireEnv(key)` — faz panic com mensagem clara se var obrigatória estiver ausente; falha rápido no boot
- `getEnv(key, fallback)` — retorna fallback se a var não estiver setada

Variáveis obrigatórias (sem elas o serviço não inicia): `AI_BASE_URL`, `AI_TOKEN`.

### `clients.go`
- `ClientConfig` — struct de uma entrada do `clients.json`
- `LoadClients(path)` — lê e valida o arquivo JSON de clientes
- `validateClients(clients)` — valida: nome único, plataforma conhecida, campos obrigatórios preenchidos

## Plataformas válidas

`"gitlab"`, `"gitlab-selfhosted"`, `"github"`, `"gitea"`, `"gitea-selfhosted"`

Ao adicionar nova plataforma, atualize o switch em `validateClients`.

## Como testar

```bash
go test ./internal/config/...
```

Os testes de `clients_test.go` criam arquivos JSON temporários via `os.CreateTemp` — não precisam de fixtures em disco.

## Notas

- `clients.json` **não é commitado** (está no `.gitignore`). Use `clients.example.json` como modelo.
- `.env` também não é commitado. Use `.env.example` como modelo.
- A struct `Config` não contém mais `GitLabConfig` — a configuração de plataformas foi movida para `clients.json` para suportar múltiplos clientes.
