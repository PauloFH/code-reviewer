# CLAUDE.md — web/src/components/

## AppLayout.vue

Shell de layout com sidebar fixa (desktop) e drawer deslizante (mobile).

- Contém `<router-view />` no `<main>` — todas as views autenticadas renderizam aqui.
- Contém `<ToastContainer />` — único ponto de montagem das notificações.
- `menuOpen: ref<boolean>` controla o drawer mobile (botão `☰` no topbar, sobreposição clicável fecha).
- Responsive: sidebar sempre visível ≥ 768 px; abaixo disso, topbar aparece e sidebar entra via `transform: translateX(-100%)`.
- `handleLogout` chama `auth.logout()` e navega para `/login`.

---

## StatsCards.vue

Cards de métricas globais.

**Props:** `{ stats: ReviewStats }`

Renderiza 5 cards: Total Reviews, Last 30d, Total Tokens, Total Cost, Cost (30d). Não faz chamadas à API — recebe `stats` do pai (`DashboardView`).

---

## ClientStats.vue

Tabela de métricas por cliente.

**Props:** `{ stats: ReviewStats['by_client'] }`

Visível apenas quando `stats.length > 0`. Colunas: Client, Reviews, Tokens (formatado com `fmtK`), Cost. Função `fmtK` converte para `k`/`M` sufixos.

---

## CostChart.vue

Gráfico de linha Chart.js com custo diário dos últimos 30 dias.

**Props:** `{ client?: string }` — filtra por cliente quando fornecido.

- Registra os módulos Chart.js necessários uma vez: `CategoryScale`, `LinearScale`, `PointElement`, `LineElement`, `Title`, `Tooltip`, `Filler`.
- Carrega dados em `onMounted` e via `watch(() => props.client, load)`.
- `chartData` e `chartOptions` são `computed` / literais — não armazenam referências ao objeto Chart.
- Tooltip exibe `$X.XXXXX`; eixo Y exibe `$X.XXXX`.

---

## LiveFeed.vue

Feed de reviews em tempo real via SSE.

- Abre `EventSource` em `onMounted` via `createEventSource(auth.token)`.
- Fecha o `EventSource` em `onUnmounted`.
- Novos eventos são inseridos no início da lista (`unshift`); máximo de 100 itens retidos.
- Status visual: dot verde (success), cinza (skipped), vermelho (error).
- `connected` badge: Live (verde) / Disconnected (vermelho) — atualizado via `es.onopen` / `es.onerror`.
- `relTime(ts)` formata timestamp como "Xs ago", "Xm ago", "Xh ago".

---

## EventsTable.vue

Tabela de eventos paginada.

**Props:** `{ events: ReviewEvent[], loading: boolean }`

- Mostra spinner enquanto `loading`.
- Badge colorido por status (`success`/`skipped`/`error`).
- Linha extra quando `error_message` presente.
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
- Cores: success = verde, error = vermelho, info = azul.
- Cada toast some após o TTL configurado em `toast.push()` — não há botão de fechar manual.

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
