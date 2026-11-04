# CLAUDE.md — web/src/components/

## AppLayout.vue

Shell de layout com sidebar fixa (desktop) e drawer deslizante (mobile).

- Contém `<router-view />` no `<main>` — todas as views autenticadas renderizam aqui.
- Monta `<BackgroundMotion />` (fundo animado) e `<BrandLogo />` (sidebar e topbar).
- Menu em três seções (Operação, Configuração, Acesso), cada item com a tecla de atalho (`NAV_SHORTCUTS`, `brand/shortcuts.ts`). O atalho é ignorado quando o foco está em campo editável, inclusive o editor Tiptap (`isEditableTarget`) ou com Ctrl/Meta/Alt.
- Rodapé: usuário e papel (`formatRole`), botão de sair e botão de tema (`useTheme().toggleTheme`).
- `menuOpen: ref<boolean>` controla o drawer mobile (botão no topbar, sobreposição clicável fecha).
- Responsive: sidebar sempre visível ≥ 768 px; abaixo disso, topbar aparece e sidebar entra via `transform: translateX(-100%)`.
- `handleLogout` chama `auth.logout()` e navega para `/login`.

---

## StatsCards.vue

Cards de métricas globais.

**Props:** `{ stats: ReviewStats }`

Renderiza 5 indicadores: Revisões no total, Revisões em 30 dias, Tokens no total, Custo total, Custo em 30 dias (formatados por `formatCount`/`formatUsd`, `brand/format.ts`). Não faz chamadas à API — recebe `stats` do pai (`DashboardView`).

---

## ClientStats.vue

Lista "Por cliente": nome, revisões, tokens, custo e a parte do custo total (barra com `--spectrum` e percentual).

**Props:** `{ stats: ReviewStats['by_client'] }`

Sem dados, a lista fica vazia e o `DashboardView` mostra o estado vazio ("Sem dados ainda."). A parte é 0 quando o custo total é 0 (sem `NaN`).

---

## CostChart.vue

Gráfico de linha Chart.js com o custo diário.

**Props:** `{ client?: string; days?: number }` — `client` filtra por cliente; `days` é o período (padrão 30; o painel envia 1, 7, 30 ou 90).

- Registra os módulos Chart.js necessários uma vez: `CategoryScale`, `LinearScale`, `PointElement`, `LineElement`, `Tooltip`, `Filler`.
- Carrega dados em `onMounted` e via `watch` de `client` e `days`.
- O canvas não entende `var(--x)`: as cores saem de `getComputedStyle` num `computed` que depende de `useTheme().theme`, então o gráfico muda de cor na troca de tema. Linha em degradê `--danger` → `--merge-mid` → `--ok`.
- Valores com `formatUsd`; datas como "4 nov".

---

## LiveFeed.vue

Feed de reviews em tempo real via SSE.

- Abre `EventSource` em `onMounted` via `createEventSource(auth.token)`.
- Fecha o `EventSource` em `onUnmounted`.
- Mostra **todos os status** (publicada, ignorada e erro), não só os publicados. Carga inicial: 20 eventos via `listEvents`; depois, o SSE.
- Novos eventos entram no início da lista (`unshift`); máximo de 50 itens retidos. Só os que chegam por SSE ganham a passada de marca-texto (`novos`).
- Situação por sinal e rótulo (`statusMark`: `+` publicada, `−` erro, `~` ignorada), nunca só por cor. Erro do backend aparece como chega.
- Emite `connection(boolean)` via `es.onopen` / `es.onerror`; o `DashboardView` mostra "conectado" / "desconectado".
- Vazio: "Aguardando revisões…". Horário relativo por `formatRelative` ("agora", "há 14 min").

---

## EventsTable.vue

Tabela de eventos paginada.

**Props:** `{ events: ReviewEvent[], loading: boolean }`

- Mostra spinner enquanto `loading`.
- Pill por situação com sinal e rótulo (`statusMark`), nunca só cor.
- Mensagem de erro do backend abaixo do título do MR, como chega.
- Botão de re-análise por linha.
- Não gerencia paginação — isso é responsabilidade de `EventsView`.

---

## ClientForm.vue

Modal de criação/edição de um client de plataforma.

**Props:** `{ initial?: Client | null }` — `null` = modo criação; `Client` = modo edição.

**Emits:**
- `close` — fecha o modal sem salvar.
- `saved(data: ClientInput & { _oldName?: string })` — emitido no submit. `_oldName` presente apenas no modo edição (usado pelo pai para chamar `updateClient(oldName, data)`).

**Expostos via `defineExpose`:**
- `setSaving(v: boolean)` — pai chama após iniciar a chamada à API.
- `setError(e: string)` — pai chama quando a API retorna erro (exibe mensagem no modal).

Em modo edição, `token` e `webhook_secret` iniciam vazios; enviar vazio significa "manter valor atual" (o servidor ignora strings vazias nesses campos). O campo `name` fica desabilitado (rename via `_oldName`).

---

## ToastContainer.vue

Renderiza os toasts do `useToastStore` em `Teleport to="body"`.

- `TransitionGroup` com animação de slide + fade.
- Tipo por sinal e cor: success `+` (`--ok`), error `−` (`--danger`), info `~`. Erro usa `role="alert"`.
- Cada toast some após o TTL configurado em `toast.push()`; clicar nele também o remove.

---

## AiConfigForm.vue

Modal de criação/edição de uma config de IA.

**Props:** `{ initial?: AIConfig | null }` — `null` = criação.

**Emits:** `close`, `saved(data: AIConfigInput & { _id?: number })`, `connected` (após exchange OAuth bem-sucedido).

**Expostos:** `setSaving(v)` / `setError(e)` (mesmo padrão de `ClientForm`/`PromptForm`).

- Seleção de **provider** e **mode**; modo `account` só habilita para `claude`.
- Modo `api`: campo de chave (placeholder indica "manter atual" na edição).
- Modo `account` (só em edição, com `id`): botão **Conectar conta Claude** → `startClaudeOAuth` abre o `authorize_url` em nova aba → campo para colar o código → `exchangeClaudeOAuth`. Mostra status conectado/expira.
- Campos de preço, **máximo de tokens** (0 = padrão) e timeout. Placeholders de base URL/model refletem o provider.

## AiConfigAssignmentForm.vue

Modal de atribuição de uma config a um escopo (global/client/projeto), espelhando `PromptAssignmentForm`. Carrega configs e clients no `onMounted`; emite `saved(data: AIAssignmentInput)`.

---

## Identidade Mergeye

- **`AppIcon.vue`** — único componente de ícones com `<svg` (a outra exceção é `LandingAbertura`). Props `name` e `size`; ícones de traço reto, sem arredondar. Para um ícone novo, adicione-o aqui.
- **`BrandLogo.vue`** — logo PNG de `assets/brand/`, versão escura ou clara pelo tema. Prop `variant`: `horizontal` | `simbolo`.
- **`LandingAbertura.vue`** — SVG animado da abertura da landing: o símbolo se monta e vira os fios vermelho e verde que se juntam na linha amarela das seções. Props `raiz`, `heroi`, `topo`, `logoTopo` e `texto` (elementos da `LandingView`); a geometria e o tempo vêm de `brand/abertura.ts`. Toca uma vez por sessão do navegador; clique, rolagem, toque ou tecla pulam para o estado final, e `prefers-reduced-motion` já abre no estado final. `requestAnimationFrame` só roda durante a abertura.
- **`BackgroundMotion.vue`** — fundo animado em canvas (raias vermelha e verde, branch sendo puxada). Prop `intensity`. Lê as cores dos tokens a cada quadro, então acompanha o tema; com `prefers-reduced-motion` desenha um quadro estático.
- **`Modal.vue`** — casca de modal em `Teleport`. Props `title` e `maxWidth`; emite `close`; slot padrão e slot `footer`. Ao montar, foca o `.modal` (`tabindex="-1"`) e, ao desmontar, devolve o foco ao elemento anterior; Esc e trap de Tab não existem. Os formulários (`ClientForm`, `PromptForm`, ...) são montados dentro dele.
- **`ConfirmModal.vue`** — confirmação de exclusão sobre `Modal`. Props `name`, `warning?`, `loading?`; emite `close` e `confirm`. Usado por Prompts, Modelos e Perfis de revisão.
