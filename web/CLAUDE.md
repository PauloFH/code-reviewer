# CLAUDE.md — web/

## Stack

Vue 3 + TypeScript + Vite. Pinia for state, axios for HTTP, Chart.js / vue-chartjs for the cost graph, Tiptap for rich text editing. No UI library — all styling is hand-written scoped CSS with a dark (`#0f172a` / Slate) palette.

## Comandos

```bash
cd web

# Desenvolvimento (hot-reload, proxies /api → :8080)
npm run dev

# Build para o binário Go (output em ../internal/transport/http/frontend/dist)
npm run build

# Type-check sem emitir
npm run type-check
```

## Integração com o Go

`vite.config.ts` aponta `build.outDir` para `../internal/transport/http/frontend/dist`. O frontend.go embute essa pasta via `//go:embed frontend/dist`. Logo:

1. `npm run build` gera os artefatos no lugar que o Go espera.
2. `go build ./...` ou `go run ./cmd/server` inclui o SPA automaticamente.
3. Em dev, use `npm run dev` (porta 5173) com o proxy Vite apontando para o servidor Go em `:8080`.

O diretório `dist/assets/` está no `.gitignore`; apenas `dist/index.html` é commitado para que `go build` funcione sem precisar de npm instalado.

## Estrutura

```
web/
├── index.html            # Template HTML root (Vite entry)
├── package.json
├── vite.config.ts        # outDir + proxy
├── tsconfig.json
└── src/
    ├── main.ts           # createApp + Pinia + router + mount
    ├── App.vue           # <router-view /> + <ToastContainer />
    ├── router/           # Rotas e navigation guard
    ├── stores/           # Pinia: auth, toast
    ├── api/              # axios + funções de API + tipos TypeScript
    ├── components/       # Componentes reutilizáveis
    └── views/            # Páginas carregadas pelo router
```

## Roteamento

| Path | Nome | Público | Componente |
|------|------|---------|------------|
| `/login` | login | sim | LoginView.vue |
| `/dashboard` | dashboard | não | DashboardView.vue |
| `/events` | events | não | EventsView.vue |
| `/clients` | clients | não | ClientsView.vue |
| `/prompts` | prompts | não | PromptsView.vue |
| `/ai-configs` | ai-configs | não (super/gestor) | AiConfigView.vue |
| `/` | — | — | redirect → /dashboard |

`/dashboard`, `/events` e `/clients` são filhos de `AppLayout.vue` (layout com sidebar).

O navigation guard em `router/index.ts` redireciona para `/login` quando `!auth.isAuthenticated` para rotas sem `meta.public`.

## Alias de path

`@` aponta para `src/`. Use `@/api/events`, `@/stores/auth`, etc.

## Convenções

- Componentes em PascalCase; arquivos `.vue` com nome idêntico ao componente.
- Tipos TypeScript exportados dos módulos `@/api/*` — nunca defina tipos inline nas views.
- Scoped CSS em todos os componentes; sem classes globais além do `body` em `index.html`.
- Props com tipos explícitos via `defineProps<{...}>()`. Emits via `defineEmits<{...}>()`.
- `defineExpose` apenas quando o pai precisar chamar métodos do filho (ex.: `ClientForm`).
