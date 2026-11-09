# CLAUDE.md — web/

## Stack

Vue 3 + TypeScript + Vite. Pinia for state, axios for HTTP, Chart.js / vue-chartjs for the cost graph, Tiptap for rich text editing. No UI library — o estilo é CSS escrito à mão sobre os tokens de `src/assets/tokens.css` (identidade **Mergeye**, nome visível do produto; o repositório, o módulo Go e o binário continuam `code-reviewer`).

## Comandos

```bash
cd web

# Desenvolvimento (hot-reload, proxies /api → :8080)
npm run dev

# Build para o binário Go (output em ../internal/transport/http/frontend/dist)
npm run build

# Type-check sem emitir
npm run type-check

# Lint
npm run lint

# Testes (node --test em tests/*.test.ts): formatação, status, contraste AA dos tokens,
# armazenamento, tema e a varredura final (tests/sweep.test.ts)
npm test

# API fictícia em :8080 para desenvolver a interface sem o backend (use --empty para o estado vazio)
node tests/mock-api.mjs
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
├── tests/                # node --test + mock-api.mjs (API fictícia, fora do build)
└── src/
    ├── main.ts           # createApp + Pinia + router + mount; importa fonts.css e tokens.css e inicia a coleta de Web Vitals
    ├── App.vue           # <router-view /> + <ToastContainer />
    ├── assets/           # tokens.css (tokens, temas e classes globais), brand/ (logos WebP) e fonts/ (woff2 self-hosted)
    ├── brand/            # funções puras: formatação pt-BR, statusMark, atalhos, readStorage, geometria da abertura da landing
    ├── composables/      # useTheme (tema claro/escuro)
    ├── router/           # Rotas e navigation guard
    ├── stores/           # Pinia: auth, toast
    ├── api/              # axios + funções de API + tipos TypeScript
    ├── components/       # Componentes reutilizáveis (AppIcon, BrandLogo, BackgroundMotion, ...)
    └── views/            # Páginas carregadas pelo router
```

## Identidade visual (Mergeye)

- **Tokens:** `src/assets/tokens.css` define as variáveis CSS (`--bg`, `--panel`, `--text`, `--accent`, `--ok`, `--danger`, `--ink`, `--spectrum`, ...). O tema escuro é o `:root`; o claro é `[data-theme="light"]`, que só redefine o que muda. Cores literais (`#hex`, `rgba(`) ficam só nesse arquivo; as views e os componentes usam `var(--x)`.
- **Fontes:** Bricolage Grotesque (texto e títulos) e DM Mono (dados, código e valores), servidas pelo próprio app: `src/assets/fonts.css` + `src/assets/fonts/*.woff2` (OFL), importados no `main.ts`. Sem CDN externo (`tests/fonts.test.ts` garante).
- **Classes globais:** `tokens.css` também traz as classes compartilhadas (`.btn`, `.pill`, `.card`, `.table`, `.page-head`, `.kpis`, `.filterbar`, `.seg`, `.feed-*`, `.empty`, ...). Estilo específico de um componente continua em `<style scoped>`.
- **Tema:** `useTheme()` alterna e persiste (`localStorage`, chave `theme`); o script inline de `index.html` aplica `data-theme` antes do Vue montar. O canvas (gráfico e fundo animado) não entende `var(--x)`: lê as cores com `getComputedStyle` e refaz ao trocar de tema.
- **Ícones:** só `AppIcon.vue` pode conter `<svg`; o resto usa `<AppIcon name="..." />`. A exceção é `LandingAbertura.vue`, o SVG animado da abertura da landing (redesenho do símbolo oficial, decorativo e com `aria-hidden`); não crie outro por script para escapar da varredura.
- **Marca:** `BrandLogo.vue` escolhe o WebP de `assets/brand/` pelo tema (600 px de largura, 3x o tamanho exibido; os masters PNG ficam em `docs/superpowers/specs/assets/mergeye/`); `BackgroundMotion.vue` é o fundo animado (respeita `prefers-reduced-motion`).
- **Situação nunca só por cor:** `statusMark()` (`brand/status.ts`) devolve sinal e rótulo: `+` publicada, `−` erro, `~` ignorada.
- **Armazenamento:** use `readStorage`/`writeStorage`/`removeStorage` (`brand/storage.ts`); `localStorage` bloqueado não lança, o valor vale só na sessão.
- **Textos:** toda a interface em PT-BR com acentuação. A varredura `tests/sweep.test.ts` falha se sobrar a marca antiga, `<svg` fora de `AppIcon` e `LandingAbertura`, cor literal em `.vue` (exceto `rgba(0, 0, 0, ...)`) ou texto do glossário em inglês.

## Roteamento

| Path | Nome | Público | Componente |
|------|------|---------|------------|
| `/` | landing | sim | LandingView.vue (quem tem sessão é redirecionado para `/dashboard` pelo guard) |
| `/login` | login | sim | LoginView.vue |
| `/dashboard` | dashboard | não | DashboardView.vue |
| `/events` | events | não | EventsView.vue |
| `/clients` | clients | não | ClientsView.vue |
| `/prompts` | prompts | não | PromptsView.vue |
| `/ai-configs` | ai-configs | não (super/gestor) | AiConfigView.vue |

`/dashboard`, `/events` e `/clients` são filhos de `AppLayout.vue` (layout com sidebar).

O navigation guard em `router/index.ts` redireciona para `/login` quando `!auth.isAuthenticated` para rotas sem `meta.public`, e `/` manda quem tem sessão para `/dashboard`.

## Alias de path

`@` aponta para `src/`. Use `@/api/events`, `@/stores/auth`, etc.

## Convenções

- Componentes em PascalCase; arquivos `.vue` com nome idêntico ao componente.
- Tipos TypeScript exportados dos módulos `@/api/*` — nunca defina tipos inline nas views.
- Scoped CSS nos componentes; estilos compartilhados vêm das classes globais e dos tokens de `src/assets/tokens.css`. Nada de cor literal nos `.vue`.
- Props com tipos explícitos via `defineProps<{...}>()`. Emits via `defineEmits<{...}>()`.
- `defineExpose` apenas quando o pai precisar chamar métodos do filho (ex.: `ClientForm`).
