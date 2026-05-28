# Task: Diagnosticar 502 no teste Claude account

## Contexto

Ao clicar em "Testar" numa configuração Claude com conta conectada, a UI exibe um `502 Bad Gateway` com payload de erro da Cloudflare.

## Plano

1. [x] Revisar o fluxo do botão no frontend e do endpoint `POST /api/v1/ai-configs/{id}/test`.
2. [x] Revisar o transporte Anthropic/Claude account, incluindo headers, URL, tratamento de erro e refresh de OAuth.
3. [x] Reproduzir por testes unitários onde possível, sem depender de chamadas reais.
4. [x] Ajustar o request Claude account para usar o perfil OAuth/Claude Code esperado.
5. [x] Rodar testes relevantes.

## Notas

- A configuração `account` usa OAuth Claude estilo Claude Code e depende de endpoints/headers externos que podem mudar.
- O erro informado pelo usuário parece ser um erro HTTP vindo de um upstream atrás de Cloudflare, não necessariamente do Cloudflare que está na frente deste sistema.
- O transporte estava enviando apenas `oauth-2025-04-20` no header `anthropic-beta`; o perfil atual do Claude Code também usa `claude-code-20250219`.
- O Dockerfile estava em Go 1.24, mas o `go.mod` exige Go 1.25.

## Resultado

- `anthropic-beta` em modo `account` agora envia `claude-code-20250219,oauth-2025-04-20`.
- Testes cobrem o perfil de request do modo conta.
- `Dockerfile` usa `golang:1.25-alpine`.
- Validação executada com `mise exec -- go test ./...`.
