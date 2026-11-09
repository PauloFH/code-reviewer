# CLAUDE.md — web/

Painel do **Mergeye** (nome visível; repositório, módulo Go e binário continuam `code-reviewer`). Vue 3 + TypeScript + Vite, Pinia, axios, Chart.js, Tiptap. Sem biblioteca de UI: CSS à mão sobre `src/assets/tokens.css`.

## Comandos

```bash
cd web
npm run dev          # :5173, proxy /api → :8080
npm run build        # vue-tsc + vite; grava em ../internal/transport/http/frontend/dist
npm run type-check
npm run lint
npm test             # node --test tests/*.test.ts
node tests/mock-api.mjs [--empty]   # API fictícia em :8080, sem backend
```

`npm test` roda no Node puro (type stripping): os testes só importam módulos `.ts` sem dependências (`src/domain.ts`, `src/brand/*`), nunca `.vue` nem algo que importe Vue ou `@/`. Regra testável vai para um desses módulos.

## Integração com o Go

O Go embute `frontend/dist` (`//go:embed`). Só `dist/index.html` é versionado (`dist/assets/` está no `.gitignore`), para `go build` funcionar sem npm.

## Contrato com o backend

- `src/domain.ts` espelha valores do Go (papéis, plataformas, escopos, situações, modos de IA) e a navegação. Mudou no backend, mude ali; rotas, menu, atalhos e rótulos derivam dessa tabela.
- Tipos em `src/api/*` espelham as structs Go (json snake_case). Provedores de IA vêm de `GET /ai-configs/providers`, não de lista fixa.
- Erro do backend é texto puro (`http.Error`): mostre com `apiErrorMessage(e, fallback)`, que só usa o corpo em 4xx `text/plain` (5xx e HTML da CDN caem no fallback).
- 401 faz logout e manda para `/login`, exceto em chamadas com `skipAuthRedirect` (login e `/auth/me`, onde o 401 é resposta esperada).
- Clientes só são listáveis por gestor/super: busque com `useClients()`, que não chama a API para os outros papéis.
- `tests/mock-api.mjs` acompanha o contrato: mudou um endpoint, mude o mock.

## Convenções

- Prompts, perfis e modelos de IA são bibliotecas atribuíveis por escopo: API via `libraryApi(prefix)` e página via `useLibrary`; o que é só de modelos de IA (provedores, OAuth, teste) fica na `AiConfigView`.
- Formulários em modal usam `FormModal` + `useSubmit`: o form recebe uma prop `save` assíncrona, aguarda e mostra o erro ele mesmo; o pai fecha o modal quando `save` resolve.
- Classe em `tokens.css` só quando mais de um componente usa (`.btn`, `.card`, `.table`, `.form`, `.field`, `.error`, `.page`, `.empty-state`, `.sr-only`, ...); classe com um único dono fica no `<style scoped>` dele.
- Cor literal (`#hex`, `rgba(`) só em `tokens.css`; o resto usa `var(--x)`. Canvas lê o token resolvido com `cssVar()` e redesenha ao trocar o tema.
- `<svg` só em `AppIcon.vue` e `LandingAbertura.vue`.
- Situação nunca só por cor: sinal + rótulo (`statusMark`).
- `localStorage` via `brand/storage.ts` (armazenamento bloqueado não lança).
- Interface toda em PT-BR. `tests/sweep.test.ts` barra marca antiga, cor literal em `.vue`, `<svg` fora do lugar e texto do glossário em inglês; `tests/contrast.test.ts` exige AA dos tokens nos dois temas.
- Acessibilidade: não zere `outline` sem outro indicador; botões de rodapé de modal usam `form="<id>"` para o navegador validar `required`; atalhos de uma tecla podem ser desligados (WCAG 2.1.4).
- `PrivacyView.vue` descreve o que o serviço coleta e com quem compartilha: mudou isso (novo provedor de IA, nova coleta), atualize a página e a data no topo.
