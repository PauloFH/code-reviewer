# Identidade Mergeye e repaginada do painel — plano de implementação

> **Para agentes:** SUB-SKILL OBRIGATÓRIA: use `superpowers:subagent-driven-development` (recomendado) ou `superpowers:executing-plans` para executar este plano tarefa por tarefa. Os passos usam checkbox (`- [ ]`).

**Objetivo:** aplicar a identidade Mergeye a todas as telas do painel Vue, nos temas escuro e claro, e publicar capturas no README, sem mudar o comportamento do sistema.

**Arquitetura:** os valores visuais ficam em `tokens.css`. Módulos TypeScript puros concentram o que tem lógica (status, formatação, atalhos), e um componente cuida dos ícones e outro das animações de fundo. As telas migram por etapas, com uma parada obrigatória para aprovação depois do Painel.

**Stack:** Vue 3, TypeScript, Vite, CSS próprio, Canvas 2D, Chart.js (já instalado), `node --test` (nativo do Node 24, sem dependência nova).

**Spec:** `docs/superpowers/specs/2026-11-04-identidade-visual-design.md`. Leia junto com este plano.

**Referência visual principal:** `.superpowers/brainstorm/637840-1793801627/content/painel-novo-design.html`, aprovada por Paulo. Abra no navegador e copie dela os valores de layout e a lógica do canvas de fundo.

## Restrições globais

- Só mudam aparência, textos e animações. Rotas, chamadas de API, permissões, formulários e regras continuam iguais. Exceção aprovada: o feed "Ao vivo" passa a mostrar todos os status.
- Fora de escopo: landing page, backend em Go, renomear repositório, módulo, binário ou imagem Docker, compra de domínio, registro de marca, demo pública.
- Nenhuma dependência nova.
- Contraste de texto em nível WCAG AA (4,5:1) nos dois temas.
- Evitar: índigo ou roxo, Geist, ícones de traço fino e arredondado, rótulos em mono maiúsculo, brilhos, sparkles, cérebros.
- Estado nunca só por cor: `+` publicada (`success`), `−` erro (`error`), `~` ignorada (`skipped`).
- Gradiente só em detalhes (barras, divisórias, gráfico), nunca sob texto corrido.
- Marca-texto só no item ativo do menu e em destaques pontuais, sempre com texto `--ink` por cima.
- Mensagens de erro vindas do backend aparecem como chegam.
- O nome visível é **Mergeye**, com essa grafia.
- Fontes: Bricolage Grotesque (texto e títulos) e DM Mono (dados, código e valores), pelo Google Fonts, como já é feito hoje.
- Todo commit termina com a linha `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Glossário de textos:** Dashboard → Painel; Events → Revisões; Clients → Clientes; Users → Usuários; Workspace → Operação; Overview → visão geral; Sign in → Entrar; Username → Usuário; Password → Senha; Total Reviews → Revisões no total; Reviews (30d) → Revisões em 30 dias; Total Tokens → Tokens no total; Total Cost → Custo total; Cost (30d) → Custo em 30 dias; all time → desde o início; Live feed → Ao vivo; live → conectado; Disconnected → desconectado; Waiting for reviews… → Aguardando revisões…; Daily cost → Custo diário; By client → Por cliente; share of total cost → parte do custo total; Time → Horário; Client → Cliente; MR → MR; Status → Situação; Cost → Custo; Duration → Duração; Cancel → Cancelar; Watch Username → Usuário monitorado; Loading stats… → Carregando indicadores…; No data yet → Sem dados ainda.

## Pontos de atenção na revisão

Cada item tem um teste na tarefa dona do código.

1. **Instalação nova, sem dados:** Painel, gráfico, Por cliente e Ao vivo mostram estados vazios legíveis, sem `NaN`, gráfico quebrado ou caixa em branco. (Tarefa 6)
2. **Digitar no editor de prompts ou em campos:** letras como `p` e `o` escrevem o texto e não trocam de tela. Hoje os atalhos ignoram `input`, `select` e `textarea`, mas não o editor Tiptap, que é `contenteditable`. (Tarefa 2)
3. **Títulos, projetos e erros muito longos, em tela estreita:** o texto quebra ou é cortado, sem rolagem horizontal na página. (Tarefas 6 e 7)
4. **Tema salvo antes do login, ou `localStorage` bloqueado:** o login já abre no tema salvo, sem piscar; com o armazenamento bloqueado, a tela abre no escuro e o botão continua funcionando na sessão. (Tarefa 1)
5. **Trocar de tema com o gráfico aberto:** gráfico e fundo animado mudam de cor na hora. O canvas não entende `var(--x)`, e o gráfico atual passa strings assim para o Chart.js. (Tarefas 4 e 6)

## Arquivos

| Arquivo | Responsabilidade |
|---|---|
| `web/src/assets/tokens.css` | Paleta, tipografia, raios e classes base |
| `web/src/assets/brand/*.png` | Logos usadas pelo painel |
| `web/index.html` | Fontes, tema inicial, título e favicon |
| `web/src/composables/useTheme.ts` | Estado único do tema |
| `web/src/brand/status.ts` | Sinal, rótulo e tom de cada status |
| `web/src/brand/format.ts` | Dólar, contagens e tempo relativo em pt-BR |
| `web/src/brand/shortcuts.ts` | Mapa de atalhos e detecção de campo editável |
| `web/src/components/AppIcon.vue` | Único lugar com desenhos de ícone |
| `web/src/components/BrandLogo.vue` | Logo certa para cada tema |
| `web/src/components/BackgroundMotion.vue` | Commits subindo e branches sendo puxadas |
| `web/tests/*.test.ts` | Testes com `node --test`, fora do `tsconfig` |
| `web/tests/mock-api.mjs` | API fictícia só para desenvolvimento e capturas |

---

### Tarefa 1: Base visual e tema

**Arquivos:**
- Criar: `web/tests/contrast.test.ts`, `web/src/composables/useTheme.ts`, `web/src/assets/brand/`
- Modificar: `web/src/assets/tokens.css:1-68`, `web/index.html`, `web/package.json`, `.gitignore`

**Interfaces:**
- Produz: tokens listados abaixo; `useTheme(): { theme: Ref<'dark' | 'light'>, toggleTheme(): void }`; script `npm test`.

- [ ] **Passo 1: Criar a branch e preparar o ambiente**

Rode `git switch -c feat/identidade-mergeye` e `cd web && npm ci`. Adicione `.superpowers/` e `.playwright-mcp/` ao `.gitignore`. Faça commit da spec, deste plano e de `docs/superpowers/specs/assets/`, com a mensagem `docs: spec e plano da identidade Mergeye`. Não inclua `exemplo.txt`.

- [ ] **Passo 2: Escrever o teste de contraste**

Em `web/tests/contrast.test.ts`, leia `tokens.css`, extraia as variáveis de `:root` e de `[data-theme="light"]` e componha cores com alfa sobre `--bg`. Exija razão ≥ 4,5 em cada tema para `--text`, `--text-2`, `--muted`, `--muted-2`, `--ok`, `--danger`, `--warn`, `--accent` sobre `--bg`, `--bg-2`, `--panel`, `--panel-solid`; e para `--ink` sobre `--hl-1` e `--hl-2`.

Adicione a `package.json` o script `"test": "node --test tests/*.test.ts"`.

- [ ] **Passo 3: Rodar o teste e ver falhar**

Rode `npm test` em `web/`. Esperado: FALHA, porque `--panel-solid`, `--hl-1`, `--hl-2` e `--ink` não existem.

- [ ] **Passo 4: Trocar os valores em `tokens.css`**

Mantenha os nomes existentes e acrescente os novos. Troque o cabeçalho "Atelier" por Mergeye.

| Token | Escuro | Claro |
|---|---|---|
| `--bg` | `#0b0c10` | `#fbf8ef` |
| `--bg-2` | `#111319` | `#f3eee0` |
| `--panel` | `rgba(20,22,29,.9)` | `rgba(255,254,249,.92)` |
| `--panel-solid` (novo) | `#14161d` | `#fffef9` |
| `--panel-2` | `#181b23` | `#f8f4e8` |
| `--hairline` | `#292c36` | `#dcd5c2` |
| `--hairline-strong` | `#3a3e4b` | `#c5bda6` |
| `--text` | `#f6f6f0` | `#16140e` |
| `--text-2` | `#d2d3cf` | `#38352a` |
| `--muted` | `#a4a7ae` | `#5e5a4d` |
| `--muted-2` | `#8b8e96` | `#6f6b5d` |
| `--accent` | `#ffc917` | `#8a6200` |
| `--accent-2` | `#fff15a` | `#6f4f00` |
| `--accent-soft` | `rgba(255,201,23,.12)` | `rgba(138,98,0,.1)` |
| `--ok` | `#1fd97c` | `#06773f` |
| `--danger` | `#ff4d3d` | `#cf2412` |
| `--warn` | `#ffc917` | `#8a6200` |
| `--skip` | `#a4a7ae` | `#5e5a4d` |
| `--info` | `#d2d3cf` | `#38352a` |
| `--hl-1` (novo) | `#fff15a` | `#fff15a` |
| `--hl-2` (novo) | `#ffc917` | `#ffd21f` |
| `--ink` (novo) | `#14130a` | `#14130a` |
| `--merge-mid` (novo) | `#ffc917` | `#e0a100` |

Comuns: `--radius: 2px`, `--radius-lg: 4px`, `--font-sans: 'Bricolage Grotesque', ui-sans-serif, system-ui, sans-serif`, `--font-mono: 'DM Mono', ui-monospace, monospace`, `--spectrum: linear-gradient(90deg, var(--danger), var(--merge-mid) 50%, var(--ok))`, `--highlight: linear-gradient(100deg, var(--hl-1), var(--hl-2))`.

Nas classes base do arquivo: `.btn.primary` usa `--highlight` e `--ink`; rótulos como `.kpi-label`, `.page-eyebrow`, `.nav-label` deixam de ser maiúsculos com letras espaçadas; `.page-title` fica em `clamp(30px, 3.4vw, 44px)`; `.kpis` usa `flex-wrap`, como na referência, com `--spectrum` na borda inferior; `.overlay` e o conteúdo de modais usam `--panel-solid`.

- [ ] **Passo 5: Rodar o teste e ver passar**

Rode `npm test`. Esperado: PASSA.

- [ ] **Passo 6: Atualizar `web/index.html`**

Título `Mergeye`; link das fontes novas; `body` sem `font-family` e cores fixas de Geist; favicon em SVG como data URI, com um olho de duas curvas (`#ff4d3d` em cima e `#1fd97c` embaixo) e um quadrado `#ffc917` no centro. Use data URI porque `dist/` só versiona o `index.html`.

Antes do módulo principal, um script inline define `data-theme` a partir de `localStorage`, dentro de `try/catch`, sem nenhum atributo quando o valor não for `light`.

- [ ] **Passo 7: Criar `useTheme` e copiar as logos**

`useTheme` compartilha um único `ref`, iniciado pelo atributo `data-theme`. `toggleTheme` atualiza o atributo e grava em `localStorage` dentro de `try/catch`. Copie de `docs/superpowers/specs/assets/mergeye/` para `web/src/assets/brand/` as versões `horizontal` e `simbolo`, nos dois temas.

- [ ] **Passo 8: Testar o ponto 4 no navegador e verificar**

Rode `npm run type-check`. Com `npm run dev`, grave `localStorage.theme = 'light'` e abra `/login`: esperado, `data-theme="light"` já presente antes de o Vue montar. Sobrescreva `localStorage.getItem` para lançar erro e recarregue: esperado, a tela abre no tema escuro sem erro no console.

- [ ] **Passo 9: Fazer commit**

`git add web .gitignore && git commit -m "feat(web): base visual e tema da identidade Mergeye"`

---

### Tarefa 2: Status, formatação e atalhos

**Arquivos:**
- Criar: `web/src/brand/status.ts`, `web/src/brand/format.ts`, `web/src/brand/shortcuts.ts`, `web/tests/brand.test.ts`

**Interfaces:**
- Produz:
  - `statusMark(status: string): { sign: '+' | '−' | '~'; label: string; tone: 'ok' | 'danger' | 'warn' }`
  - `formatUsd(value: number): string`
  - `formatCount(value: number): string`
  - `formatRelative(iso: string, now?: Date): string`
  - `NAV_SHORTCUTS: Record<string, string>`
  - `isEditableTarget(target: { tagName?: string; isContentEditable?: boolean } | null): boolean`

- [ ] **Passo 1: Escrever os testes**

Em `web/tests/brand.test.ts`, importe com a extensão `.ts` e confira:

```ts
assert.deepEqual(statusMark('success'), { sign: '+', label: 'publicada', tone: 'ok' })
assert.deepEqual(statusMark('error'), { sign: '−', label: 'erro', tone: 'danger' })
assert.deepEqual(statusMark('skipped'), { sign: '~', label: 'ignorada', tone: 'warn' })
assert.deepEqual(statusMark('novo'), { sign: '~', label: 'novo', tone: 'warn' })
assert.equal(formatUsd(38.17), 'US$ 38,17')
assert.equal(formatUsd(0), 'US$ 0,00')
assert.equal(formatUsd(0.00042), 'US$ 0,00042')
assert.equal(formatUsd(Number.NaN), 'US$ 0,00')
assert.equal(formatCount(1284), '1.284')
assert.equal(formatCount(12400), '12,4 mil')
assert.equal(formatCount(4200000), '4,2 mi')
assert.equal(formatCount(0), '0')
const now = new Date('2026-11-04T12:00:00Z')
assert.equal(formatRelative('2026-11-04T11:59:40Z', now), 'agora')
assert.equal(formatRelative('2026-11-04T11:46:00Z', now), 'há 14 min')
assert.equal(formatRelative('2026-11-04T09:00:00Z', now), 'há 3 h')
assert.equal(formatRelative('2026-11-01T12:00:00Z', now), 'há 3 dias')
assert.deepEqual(NAV_SHORTCUTS, { p: '/dashboard', r: '/events', c: '/clients', o: '/prompts', m: '/ai-configs', f: '/review-profiles', u: '/users' })
assert.equal(isEditableTarget({ tagName: 'INPUT' }), true)
assert.equal(isEditableTarget({ tagName: 'SELECT' }), true)
assert.equal(isEditableTarget({ tagName: 'TEXTAREA' }), true)
assert.equal(isEditableTarget({ tagName: 'DIV', isContentEditable: true }), true)
assert.equal(isEditableTarget({ tagName: 'DIV', isContentEditable: false }), false)
assert.equal(isEditableTarget(null), false)
```

- [ ] **Passo 2: Rodar e ver falhar**

Rode `npm test`. Esperado: FALHA, módulos inexistentes.

- [ ] **Passo 3: Implementar os três módulos**

`formatUsd` usa duas casas de valor ≥ 0,01 e até cinco abaixo disso, com vírgula decimal. `formatCount` usa `mi` a partir de 1.000.000, `mil` a partir de 10.000 e separador de milhar abaixo disso. Valores não finitos valem 0. Os módulos não importam Vue nem usam `@/`.

- [ ] **Passo 4: Rodar e ver passar**

Rode `npm test`. Esperado: PASSA.

- [ ] **Passo 5: Fazer commit**

`git add web && git commit -m "feat(web): status, formatação e atalhos em pt-BR"`

---

### Tarefa 3: Ícones, logo, menu e API fictícia

**Arquivos:**
- Criar: `web/src/components/AppIcon.vue`, `web/src/components/BrandLogo.vue`, `web/tests/mock-api.mjs`
- Modificar: `web/src/components/AppLayout.vue`

**Interfaces:**
- Consome: `useTheme`, `NAV_SHORTCUTS`, `isEditableTarget`.
- Produz:
  - `<AppIcon name="..." :size="14" />`, com `name: IconName`
  - `IconName = 'plus' | 'trash' | 'edit' | 'close' | 'refresh' | 'activity' | 'document' | 'layers' | 'check' | 'play' | 'settings' | 'merge' | 'external' | 'search' | 'calendar' | 'chevron-left' | 'chevron-right' | 'sun' | 'moon' | 'logout' | 'menu'`
  - `<BrandLogo variant="horizontal" | "simbolo" />`
  - `node web/tests/mock-api.mjs [--empty]`, escutando em `:8080`

- [ ] **Passo 1: Criar a API fictícia**

Servidor com `node:http`, sem dependência, reproduzindo as rotas usadas em `web/src/api/*.ts` e `web/src/stores/auth.ts`, incluindo login, estatísticas, eventos, SSE, clientes, prompts, configs, perfis e usuários. Use os dados fictícios da referência visual, mais um evento com título, projeto e `error_message` de 300 caracteres. Com `--empty`, todas as listas vêm vazias e os totais zerados. O arquivo fica fora do build e serve só para desenvolvimento.

- [ ] **Passo 2: Criar `AppIcon` e `BrandLogo`**

`AppIcon` guarda os desenhos em um único mapa, com `viewBox="0 0 24 24"`, `stroke-width="2"`, `stroke-linecap="square"`, `stroke-linejoin="miter"` e cantos retos, herdando `currentColor`. Redesenhe cada ícone nesse estilo, sem copiar os atuais. `merge` substitui o sparkle de `ClientsView.vue`.

`BrandLogo` usa `useTheme` para escolher o PNG, com `alt="Mergeye"`.

- [ ] **Passo 3: Repaginar `AppLayout.vue`**

Copie da referência o menu sem ícones, o item ativo com `--highlight` e `--ink`, a logo horizontal a 190px, os rótulos do glossário e as letras de `NAV_SHORTCUTS`. A barra lateral usa `--bg-2`. Remova `v 2.4 · prod`. O rodapé mostra usuário, função, botão de tema com o texto "Tema claro" ou "Tema escuro" e o botão de sair. Mantenha o drawer de celular, as verificações de permissão e as rotas.

Use `isEditableTarget` no `handleKeydown` e `useTheme` no lugar do estado local de tema.

- [ ] **Passo 4: Verificar o ponto 2 no navegador**

Rode `node web/tests/mock-api.mjs` e `npm run dev`. Em `/prompts`, abra o formulário e digite `pofrmcu` no editor. Esperado: o texto aparece e a rota não muda. Fora de campos, pressione `r`: esperado, abre `/events`.

- [ ] **Passo 5: Verificar e fazer commit**

Rode `npm run type-check`, `npm run lint` e `npm test`. Esperado: tudo passa.

`git add web && git commit -m "feat(web): ícones próprios, logo e menu Mergeye"`

---

### Tarefa 4: Animações de fundo

**Arquivos:**
- Criar: `web/src/components/BackgroundMotion.vue`
- Modificar: `web/src/components/AppLayout.vue`

**Interfaces:**
- Consome: `useTheme`.
- Produz: `<BackgroundMotion :intensity="1" />`. O valor `1` é o do painel; a Tarefa 5 usa `1.6` no login.

- [ ] **Passo 1: Implementar o componente**

Porte o canvas da função `draw` da referência visual: commits subindo pelas raias e branches sendo puxadas entre elas. Canvas fixo, com `z-index: 0`, `pointer-events: none` e `aria-hidden`. `intensity` multiplica a opacidade. As cores vêm de `getComputedStyle` a cada quadro, para acompanhar o tema. Limpe `requestAnimationFrame` e o listener de `resize` ao desmontar. Não desenhe com a aba oculta. Com `prefers-reduced-motion`, desenhe um único quadro estático e redesenhe ao trocar de tema ou redimensionar.

- [ ] **Passo 2: Colocar no `AppLayout`**

Fica atrás de todo o conteúdo autenticado, com o conteúdo em `z-index: 1`.

- [ ] **Passo 3: Verificar no navegador**

Com a API fictícia: troque o tema e veja o fundo mudar de cor na hora. Emule `prefers-reduced-motion: reduce` e capture dois quadros com 2 segundos de intervalo: esperado, idênticos. Navegue por três telas e confira que há um só canvas na página.

- [ ] **Passo 4: Verificar e fazer commit**

Rode `npm run type-check` e `npm run lint`. Esperado: passam.

`git add web && git commit -m "feat(web): animações translúcidas de commits e branches"`

---

### Tarefa 5: Login

**Arquivos:**
- Modificar: `web/src/views/LoginView.vue`

**Interfaces:**
- Consome: `BrandLogo`, `BackgroundMotion`.

- [ ] **Passo 1: Repaginar a tela**

Logo com o nome ao lado, `<BackgroundMotion :intensity="1.6" />` e cartão com `--panel-solid`. Textos: "Entrar", "Usuário", "Senha". Sem cores fixas e sem `v 2.4`. Mantenha validação, envio e mensagem de erro.

- [ ] **Passo 2: Verificar no navegador**

Com a API fictícia: login correto abre o Painel; login com erro mostra a mensagem na tela. Confira os dois temas, com a largura de 400px, sem rolagem horizontal.

- [ ] **Passo 3: Verificar e fazer commit**

Rode `npm run type-check` e `npm run lint`.

`git add web && git commit -m "feat(web): tela de login Mergeye"`

---

### Tarefa 6: Painel

**Arquivos:**
- Modificar: `web/src/views/DashboardView.vue`, `web/src/components/StatsCards.vue`, `web/src/components/CostChart.vue`, `web/src/components/ClientStats.vue`, `web/src/components/LiveFeed.vue`

**Interfaces:**
- Consome: `statusMark`, `formatUsd`, `formatCount`, `formatRelative`, `useTheme`, `AppIcon`.

- [ ] **Passo 1: Repaginar as seções**

Siga a referência e use os textos do glossário. Em `StatsCards`, use `formatCount` e `formatUsd`.

Em `ClientStats`, barras com `--spectrum`.

Em `LiveFeed`, cada linha mostra o sinal de `statusMark`, e o novo evento entra com a passada de marca-texto da referência. Remova os filtros de `LiveFeed.vue:57` e `:67`, que mostram só `success`, para o feed exibir os três status; essa é a única exceção de comportamento aprovada na spec. Remova o CSS morto `.status.failed` e mostre `error_message` nas linhas de erro. Títulos e metadados usam `overflow-wrap: anywhere`.

- [ ] **Passo 2: Corrigir as cores do gráfico**

Em `CostChart.vue`, resolva as cores com `getComputedStyle(document.documentElement)` em um `computed` que depende de `useTheme().theme`, usando gradiente horizontal de `--danger`, `--merge-mid` e `--ok` por função `scriptable` do Chart.js, e preenchimento translúcido. Remova as strings `var(--...)` e `rgba(129,140,248,...)` passadas ao Chart.js. Eixos e tooltip usam `formatUsd`. Remova o título interno em inglês, porque o card já tem título.

- [ ] **Passo 3: Verificar os pontos 1, 3 e 5 no navegador**

- `node web/tests/mock-api.mjs --empty`: indicadores em `0` e `US$ 0,00`, gráfico com "Sem dados ainda", Ao vivo com "Aguardando revisões…", nenhum `NaN`.
- API normal a 400px: com o evento de 300 caracteres, `document.documentElement.scrollWidth <= innerWidth`.
- Trocar o tema com o gráfico aberto: o traço muda de cor sem recarregar.
- Conferir a aparência contra a referência nos dois temas.

- [ ] **Passo 4: Verificar e fazer commit**

Rode `npm run type-check`, `npm run lint` e `npm test`.

`git add web && git commit -m "feat(web): Painel na identidade Mergeye"`

---

### Parada para aprovação

**Pare aqui.** Mostre a Paulo logo, cores, ícones, login e Painel rodando, nos dois temas, usando a API fictícia. As Tarefas 7 a 9 só começam depois da aprovação dele, e as decisões da spec ainda podem mudar nesse ponto.

---

### Tarefa 7: Revisões

**Arquivos:**
- Modificar: `web/src/views/EventsView.vue`, `web/src/components/EventsTable.vue`

**Interfaces:**
- Consome: `statusMark`, `formatUsd`, `formatCount`, `AppIcon`.

- [ ] **Passo 1: Repaginar**

Coluna Situação com sinal e rótulo de `statusMark`; colunas, filtros e paginação com os textos do glossário; valores em DM Mono com os formatadores; ícones por `AppIcon`. Mantenha os valores brutos de `e.status` na exportação em `EventsView.vue:165`, porque ela é dado, não interface.

- [ ] **Passo 2: Verificar no navegador**

Com a API normal e `--empty`, nos dois temas: filtro, paginação, Re-analisar e exportação continuam funcionando. A 400px, a tabela rola dentro de `.scroll-x` e a página não tem rolagem horizontal, inclusive com o evento de 300 caracteres.

- [ ] **Passo 3: Verificar e fazer commit**

Rode `npm run type-check` e `npm run lint`.

`git add web && git commit -m "feat(web): tela de Revisões na identidade Mergeye"`

---

### Tarefa 8: Componentes compartilhados, Clientes e Usuários

**Arquivos:**
- Modificar: `web/src/components/Modal.vue`, `ConfirmModal.vue`, `ToastContainer.vue`, `ClientForm.vue`, `web/src/views/ClientsView.vue`, `web/src/views/UsersView.vue`

**Interfaces:**
- Consome: `AppIcon`.

- [ ] **Passo 1: Repaginar**

Modais e toasts usam `--panel-solid`, `--ok`, `--danger` e `--warn`, sem cores fixas. Toasts indicam o tipo também por sinal, não só por cor. Aplique o glossário e `AppIcon`.

- [ ] **Passo 2: Verificar no navegador**

Com a API fictícia, nos dois temas: criar, editar e excluir cliente e usuário; ver toast de sucesso e de erro; confirmar que o erro do backend aparece como chegou. Confira que um usuário sem permissão não vê as rotas protegidas.

- [ ] **Passo 3: Verificar e fazer commit**

Rode `npm run type-check` e `npm run lint`.

`git add web && git commit -m "feat(web): modais, toasts, Clientes e Usuários na identidade Mergeye"`

---

### Tarefa 9: Prompts, Modelos e Perfis de revisão

**Arquivos:**
- Modificar: `web/src/views/PromptsView.vue`, `AiConfigView.vue`, `ReviewProfilesView.vue`; `web/src/components/PromptForm.vue`, `PromptAssignmentForm.vue`, `AiConfigForm.vue`, `AiConfigAssignmentForm.vue`, `ReviewProfileForm.vue`, `ReviewProfileAssignmentForm.vue`, `RichTextEditor.vue`

**Interfaces:**
- Consome: `AppIcon`.

- [ ] **Passo 1: Repaginar**

Troque cores fixas por tokens, ícones por `AppIcon` e textos em inglês conforme o glossário. O editor usa `--panel-solid` e `--text`.

- [ ] **Passo 2: Verificar no navegador**

Com a API fictícia, nos dois temas: criar e editar prompt, config e perfil; atribuir por escopo; usar Testar em Modelos; digitar no editor sem acionar atalho.

- [ ] **Passo 3: Verificar e fazer commit**

Rode `npm run type-check` e `npm run lint`.

`git add web && git commit -m "feat(web): Prompts, Modelos e Perfis na identidade Mergeye"`

---

### Tarefa 10: Varredura final, capturas e documentação

**Arquivos:**
- Criar: `docs/media/mergeye-painel-escuro.png`, `mergeye-painel-claro.png`, `mergeye-revisoes.png`, `mergeye-login.png`, `mergeye-painel.gif`, `web/tests/sweep.test.ts`
- Modificar: `README.md`, `web/CLAUDE.md`, `CLAUDE.md`, `internal/transport/http/frontend/dist/index.html`

- [ ] **Passo 1: Escrever a varredura**

`web/tests/sweep.test.ts` percorre `web/src/**/*.{vue,ts,css}` e exige:

- nenhuma ocorrência de `Geist`, `Code Reviewer`, `v 2.4`, `818cf8`, `6366f1`, `4f46e5` ou `<svg` fora de `AppIcon.vue`;
- nenhuma cor hexadecimal ou `rgba(` em arquivos `.vue`, exceto `rgba(0, 0, 0, ...)` em sombras e overlays;
- nenhum dos textos em inglês do glossário dentro de tags (`>Sign in<`, `>Total Reviews<`, `>Cancel<` e demais).

- [ ] **Passo 2: Rodar e corrigir o que sobrar**

Rode `npm test`. Corrija cada item encontrado até passar.

- [ ] **Passo 3: Gerar as capturas**

Com a API fictícia, capture Painel nos dois temas, Revisões e login a 1440 × 900, e um GIF de até 5 MB e cerca de 8 segundos do Painel, com `ffmpeg`, mostrando o fundo em movimento e um evento novo entrando. Salve em `docs/media/`.

- [ ] **Passo 4: Atualizar a documentação**

- `README.md`: título Mergeye com `mergeye-logo-horizontal-escuro.png`, nota de que o repositório continua `code-reviewer`, e as capturas e o GIF na seção "Dashboard de monitoramento".
- `web/CLAUDE.md`: troque a paleta Slate e a regra "sem classes globais" pela descrição de `tokens.css`, `AppIcon`, `brand/`, `npm test` e a API fictícia.
- `CLAUDE.md`: registre que Mergeye é o nome visível do produto.

- [ ] **Passo 5: Verificação final**

Rode em `web/`: `npm test`, `npm run lint` e `npm run build`. Na raiz: `go test ./...`. Esperado: tudo passa, e `dist/index.html` traz `Mergeye` e as fontes novas. Rode `go run ./cmd/server` e confira que o painel embutido abre.

- [ ] **Passo 6: Fazer commit**

`git add web docs README.md CLAUDE.md internal/transport/http/frontend/dist/index.html && git commit -m "docs: capturas do Mergeye e varredura final da identidade"`
