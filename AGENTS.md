# AGENTS.md — Code Reviewer (Mergeye)

Instruções para qualquer agente de código (Claude Code, Codex e outros).

**Leia o [`CLAUDE.md`](CLAUDE.md) antes de mudar qualquer coisa.** Ele é a fonte única do projeto: o que o serviço faz, quando uma revisão roda, arquitetura, convenções para estender (plataforma, provider de IA, banco), variáveis de ambiente e comandos. Este arquivo não repete nada disso: se algo mudar no projeto, atualize o `CLAUDE.md`.

## Regras de trabalho

- **Verifique antes de entregar.** Rode o mesmo que o CI, porque ele só roda em push no `main` (PRs não têm CI):
  ```bash
  go vet ./... && go test -race ./... && golangci-lint run ./...
  cd web && npm run lint && npm run type-check && npm test && npm run build
  ```
  O `npm run build` reescreve `internal/transport/http/frontend/dist/`; não commite essa mudança sem querer (`git checkout -- internal/transport/http/frontend/dist`).
- **Commits:** uma linha só, `tipo: mensagem` (ex.: `fix: ...`, `feat: ...`), sem escopo, sem corpo e sem linha de coautor.
- **Branches e PRs:** o remoto principal é o GitHub `PauloFH/code-reviewer`. Trabalhe num branch e abra PR; push no `main` dispara o deploy (após aprovação no ambiente `production`).
- **Testes** precisam falhar quando o comportamento quebra. Não escreva teste que copie a implementação ou que não possa falhar.
