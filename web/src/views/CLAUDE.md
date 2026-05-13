# CLAUDE.md — web/src/views/

## LoginView.vue

Página pública de login (`/login`).

- Formulário com `username` e `password`.
- Chama `auth.login(username, password)` no submit.
- Mostra spinner no botão durante a chamada e mensagem de erro inline em falha.
- Redireciona para `/dashboard` em caso de sucesso via `router.push`.
- Se já autenticado ao montar, redireciona imediatamente para `/dashboard`.

---

## DashboardView.vue

Visão geral do sistema (`/dashboard`).

**Dados carregados em `onMounted`:**
- `getStats()` → `ReviewStats` — passado para `StatsCards` e `ClientStats`.

**Composição:**
```
StatsCards  — 5 cards de métricas globais
ClientStats — tabela breakdown por cliente (oculto se by_client vazio)
CostChart   — gráfico de custo diário 30 dias
LiveFeed    — SSE em tempo real
```

Todos os dados do Dashboard são independentes — nenhum componente filho espera dados do outro.

---

## EventsView.vue

Histórico paginado de reviews (`/events`).

**Estado:**
- `events: ReviewEvent[]`, `total: number`, `page: number`, `loading: boolean`
- `filterClient`, `filterSince`, `filterUntil` — inputs controlados
- `pageSize = 25` (fixo)

**Fluxo:**
1. `onMounted` → `load()`
2. `applyFilter()` reseta `page = 0` e chama `load()`
3. `prevPage()` / `nextPage()` incrementam `page` e chamam `load()`

`load()` constrói o filtro de datas com sufixo `T00:00:00Z` / `T23:59:59Z` para garantir cobertura do dia inteiro.

`totalPages = Math.max(1, Math.ceil(total / pageSize))` — mínimo 1 página mesmo sem dados.

`EventsTable` recebe a lista e o estado de loading; paginação é gerenciada pela view.

---

## ClientsView.vue

CRUD de clients de plataforma (`/clients`).

**Estado:**
- `clients: Client[]` — lista carregada do servidor
- `showForm: boolean` — controla visibilidade do modal `ClientForm`
- `editingClient: Client | null` — `null` = criação; `Client` = edição
- `deleteTarget: string` — nome do client a deletar; `''` = nenhum
- `formRef` — ref para a instância de `ClientForm` (para chamar `setSaving`/`setError`)

**Fluxo de criação/edição:**
1. `openCreate()` / `openEdit(c)` → define `editingClient` e abre `showForm`
2. `ClientForm` emite `saved(data)` com `_oldName` se edição
3. `handleSaved` chama `setSaving(true)` no form, faz a chamada à API
4. Sucesso → `showForm = false`, `load()`, toast success
5. Erro → `setError(msg)`, `setSaving(false)` — form permanece aberto com mensagem

**Fluxo de exclusão:**
1. Botão Delete → `deleteTarget = c.name` → modal de confirmação aparece
2. `doDelete()` chama `deleteClient(deleteTarget)`, toast, `load()`
3. Cancelar ou clicar fora → `deleteTarget = ''`

Toasts usam `useToastStore` — sucesso e erro aparecem no canto via `ToastContainer`.
