# CLAUDE.md — web/src/views/

## LandingView.vue

Landing pública do Mergeye (`/`): topo, herói, quatro seções ligadas por uma linha amarela que desce pela página (como funciona, níveis de revisão, painel por capturas, como é feito) e rodapé.

- Rota pública; o guard redireciona para `/dashboard` quem tem sessão. Não faz chamada de API nem usa `BackgroundMotion`.
- `LandingAbertura` desenha a abertura animada sobre o herói; ela escreve as variáveis `--abertura-*` em `.landing`, e sem ela valem os valores do estado final.
- As capturas (`assets/landing/`) seguem o tema de `useTheme()`. Níveis Relacionado, Agêntica e Execução aparecem com a ressalva de que hoje são só GitLab.

---

## LoginView.vue

Página pública de login (`/login`).

- Formulário "Entrar" com usuário e senha; logo (`BrandLogo`) e fundo animado (`BackgroundMotion`) no tema salvo.
- Chama `auth.login(username, password)` no submit.
- Mostra spinner no botão durante a chamada e a mensagem "Credenciais inválidas" inline em falha.
- Redireciona para `/dashboard` em caso de sucesso via `router.push`.
- Se já autenticado ao montar, redireciona imediatamente para `/dashboard`.

---

## DashboardView.vue

Painel (`/dashboard`): visão geral do sistema.

**Estado:** `period` (`24h` | `7d` | `30d` | `90d`, padrão `7d`; vira `days` do `CostChart`), `filterClient` (filtra gráfico e indicadores), `connected` (vem do `LiveFeed`).

**Dados:** `getStats()` em `onMounted`, ao trocar cliente e no botão "Atualizar" (que também recarrega o gráfico via `:key`); trocar o período só recarrega o `CostChart`, que observa `days`. Os filhos não são desmontados durante recargas: o skeleton aparece só na primeira carga. `listClients()` alimenta o seletor.

**Composição:**
```
StatsCards  — 5 indicadores globais
CostChart   — "Custo diário" (período e cliente escolhidos)
ClientStats — "Por cliente" (estado vazio "Sem dados ainda." quando by_client é vazio)
LiveFeed    — "Ao vivo": SSE com todos os status, selo conectado/desconectado
```

Instalação nova (sem dados): indicadores em zero, gráfico e listas vazios e legíveis, sem `NaN`.

---

## EventsView.vue

Histórico paginado de reviews (`/events`).

**Estado:**
- `events: ReviewEvent[]`, `total: number`, `page: number`, `loading: boolean`
- `filterSearch`, `filterClient`, `filterStatus`, `filterSince`, `filterUntil` — inputs controlados
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

Lista e edição de clients de plataforma (`/clients`). Não há exclusão pela interface: `deleteClient` existe em `@/api/clients`, mas nenhuma view o chama.

**Estado:**
- `clients: Client[]` — lista carregada do servidor
- `showForm: boolean` — controla visibilidade do modal `ClientForm`
- `editingClient: Client | null` — `null` = criação; `Client` = edição
- `formRef` — ref para a instância de `ClientForm` (para chamar `setSaving`/`setError`)

**Fluxo de criação/edição:**
1. `openCreate()` / `openEdit(c)` → define `editingClient` e abre `showForm`
2. `ClientForm` emite `saved(data)` com `_oldName` se edição
3. `handleSaved` chama `setSaving(true)` no form, faz a chamada à API
4. Sucesso → `showForm = false`, `load()`, toast success
5. Erro → `setError(msg)`, `setSaving(false)` — form permanece aberto com mensagem

O botão "Sincronizar" apenas recarrega a lista (`load()`) e mostra um toast; as métricas por cliente vêm de `getStats()`.

Toasts usam `useToastStore` — sucesso e erro aparecem no canto via `ToastContainer`.

---

## AiConfigView.vue

Gerência de configs de IA (`/ai-configs`), espelhando a estrutura de `PromptsView`: lista de configs + lista de atribuições, com modais `AiConfigForm` e `AiConfigAssignmentForm`.

**Particularidades:**
- Coluna **Credencial** mostra "Conectado"/"Chave definida" conforme `mode`.
- Ao criar uma config em modo **account**, a view recarrega e reabre em modo edição (a conexão OAuth precisa do `id`).
- Botão **Testar** chama `testConfig(id)` e abre um modal com o `review_markdown` + contagem de tokens; erro de upstream aparece no modal.
- `onConnected` (emitido pelo form após o exchange OAuth) recarrega a lista.
