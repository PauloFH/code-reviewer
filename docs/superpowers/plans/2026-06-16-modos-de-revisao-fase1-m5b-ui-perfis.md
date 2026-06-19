# Modos de Revisão — Fase 1 / Milestone 5b: UI de perfis (Vue) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adicionar à SPA Vue a tela de gerenciamento de perfis de revisão (CRUD + atribuição por escopo), espelhando exatamente o fluxo de Prompts.

**Architecture:** Frontend Vue 3 + TS. Espelha o trio Prompts: módulo de API (`reviewProfiles.ts`), `view` (`ReviewProfilesView.vue`), e dois `components` de formulário (`ReviewProfileForm.vue`, `ReviewProfileAssignmentForm.vue`), mais a rota e o link de navegação. A diferença essencial: o formulário de perfil NÃO usa rich text — usa **checkboxes** (`include_related`, `agentic`, `execution`) e um **number** (`max_iterations`).

**Tech Stack:** Vue 3 (`<script setup>` + TS), axios via `@/api/http`, Pinia (`useToastStore`). Sem libs de UI — CSS scoped, paleta dark, como os demais componentes.

**Verificação:** este é frontend — não há teste Go. Verifique com `cd web && npm run type-check` (tsc sem emitir) e `npm run build`.

**Pré-requisitos:** M5a (API REST `/api/v1/review-profiles` no backend).

**Arquivos-referência a ESPELHAR (leia cada um e produza o equivalente para review profiles):**
- `web/src/api/prompts.ts` → `reviewProfiles.ts`
- `web/src/views/PromptsView.vue` → `ReviewProfilesView.vue`
- `web/src/components/PromptForm.vue` → `ReviewProfileForm.vue` (trocar rich text por checkboxes + number)
- `web/src/components/PromptAssignmentForm.vue` → `ReviewProfileAssignmentForm.vue`
- `web/src/router/index.ts` (adicionar rota)
- `web/src/components/AppLayout.vue` (adicionar link na seção "Configuração", com `v-if="auth.canManageResources"`)
- Convenções: `web/CLAUDE.md` e `web/src/api/CLAUDE.md`.

---

## File Structure

- **Create:** `web/src/api/reviewProfiles.ts`
- **Create:** `web/src/components/ReviewProfileForm.vue`
- **Create:** `web/src/components/ReviewProfileAssignmentForm.vue`
- **Create:** `web/src/views/ReviewProfilesView.vue`
- **Modify:** `web/src/router/index.ts`
- **Modify:** `web/src/components/AppLayout.vue`

---

## Task 1: Módulo de API `reviewProfiles.ts`

**Files:**
- Create: `web/src/api/reviewProfiles.ts`

- [ ] **Step 1: Implementar** (espelha `web/src/api/prompts.ts`, endpoints `/review-profiles`)

```ts
import { http } from './http'

export interface ReviewProfile {
  id: number
  name: string
  include_related: boolean
  agentic: boolean
  execution: boolean
  max_iterations: number
  created_at: string
  updated_at: string
}

export type ReviewProfileInput = Pick<
  ReviewProfile,
  'name' | 'include_related' | 'agentic' | 'execution' | 'max_iterations'
>

export interface ReviewProfileAssignment {
  id: number
  profile_id: number
  profile_name: string
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export interface AssignmentInput {
  profile_id: number
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export async function listReviewProfiles(): Promise<ReviewProfile[]> {
  const res = await http.get('/review-profiles')
  return res.data
}

export async function getReviewProfile(id: number): Promise<ReviewProfile> {
  const res = await http.get(`/review-profiles/${id}`)
  return res.data
}

export async function createReviewProfile(data: ReviewProfileInput): Promise<ReviewProfile> {
  const res = await http.post('/review-profiles', data)
  return res.data
}

export async function updateReviewProfile(id: number, data: ReviewProfileInput): Promise<ReviewProfile> {
  const res = await http.put(`/review-profiles/${id}`, data)
  return res.data
}

export async function deleteReviewProfile(id: number): Promise<void> {
  await http.delete(`/review-profiles/${id}`)
}

export async function listAssignments(): Promise<ReviewProfileAssignment[]> {
  const res = await http.get('/review-profiles/assignments')
  return res.data
}

export async function setAssignment(data: AssignmentInput): Promise<void> {
  await http.put('/review-profiles/assignments', data)
}

export async function removeAssignment(scope: string, clientName: string, projectRef: string): Promise<void> {
  await http.delete('/review-profiles/assignments', { params: { scope, client_name: clientName, project_ref: projectRef } })
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/api/reviewProfiles.ts
git commit -m "feat(web): módulo de API reviewProfiles"
```

---

## Task 2: Componente `ReviewProfileForm.vue`

**Files:**
- Create: `web/src/components/ReviewProfileForm.vue`

- [ ] **Step 1: Implementar**

Leia `web/src/components/PromptForm.vue` e produza o equivalente, com a MESMA estrutura (props `editing?: ReviewProfile | null`, emits `close`/`saved`, `defineExpose({ setSaving, setError })`, modal via `Modal.vue`, CSS scoped no mesmo estilo). Diferenças de campos:
- `name`: input text (obrigatório).
- `include_related`, `agentic`, `execution`: **checkboxes** (labels: "Incluir contexto relacionado (issues/MRs/discussão)", "Revisão agêntica (checkout + navegação)", "Execução em sandbox (Fase 2)").
- `max_iterations`: input `type="number"` min 0 (label: "Máximo de iterações agênticas (0 = padrão)").
- O `saved` emite `ReviewProfileInput & { _id?: number }` (igual ao padrão do PromptForm: inclui `_id` quando editando).

Tipos importados de `@/api/reviewProfiles` (NUNCA inline — convenção do projeto). NÃO use `RichTextEditor`.

- [ ] **Step 2: Commit**

```bash
git add web/src/components/ReviewProfileForm.vue
git commit -m "feat(web): ReviewProfileForm (checkboxes de camadas + max_iterations)"
```

---

## Task 3: Componente `ReviewProfileAssignmentForm.vue`

**Files:**
- Create: `web/src/components/ReviewProfileAssignmentForm.vue`

- [ ] **Step 1: Implementar**

Leia `web/src/components/PromptAssignmentForm.vue` e produza o equivalente trocando "prompt" por "profile": props recebem a lista de `ReviewProfile[]` (para o `<select>` de perfil) — espelhe exatamente como o PromptAssignmentForm recebe a lista de prompts. Campos: select de perfil (`profile_id`), select de `scope` (global/client/project), inputs `client_name` e `project_ref` (condicionais ao escopo, igual ao de prompts). Emits `close`/`saved` com `AssignmentInput`; `defineExpose({ setSaving, setError })`. Tipos de `@/api/reviewProfiles`.

- [ ] **Step 2: Commit**

```bash
git add web/src/components/ReviewProfileAssignmentForm.vue
git commit -m "feat(web): ReviewProfileAssignmentForm (atribuição por escopo)"
```

---

## Task 4: View `ReviewProfilesView.vue`

**Files:**
- Create: `web/src/views/ReviewProfilesView.vue`

- [ ] **Step 1: Implementar**

Leia `web/src/views/PromptsView.vue` e produza o equivalente:
- lista de perfis (tabela) mostrando nome + badges das camadas ativas (ex.: "related", "agêntico") + max_iterations; botões editar/excluir (com `ConfirmModal`).
- card de atribuições (tabela) com escopo/client/project/perfil; botão de adicionar; remover atribuição.
- modais `ReviewProfileForm` e `ReviewProfileAssignmentForm` (passe a lista de perfis ao assignment form).
- usa `useToastStore`, `listReviewProfiles`/`listAssignments` em `Promise.all` no `load()`, e os handlers `handleSaved`/`handleAssignSaved` espelhando a PromptsView (incluindo `formRef.value?.setSaving/setError`).

Importe tipos e funções de `@/api/reviewProfiles`.

- [ ] **Step 2: Commit**

```bash
git add web/src/views/ReviewProfilesView.vue
git commit -m "feat(web): ReviewProfilesView (lista + atribuições + modais)"
```

---

## Task 5: Rota + navegação

**Files:**
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/AppLayout.vue`

- [ ] **Step 1: Adicionar a rota**

Em `web/src/router/index.ts`, dentro de `children` (após o bloco `prompts`), adicionar:

```ts
        {
          path: 'review-profiles',
          name: 'review-profiles',
          component: () => import('@/views/ReviewProfilesView.vue'),
          meta: { roles: ['super', 'gestor'] },
        },
```

- [ ] **Step 2: Adicionar o link de navegação**

Em `web/src/components/AppLayout.vue`, na seção de navegação "Configuração" (junto aos links de Prompts/AI configs), adicionar um `router-link` para `/review-profiles` seguindo o MESMO padrão visual dos vizinhos (com `v-if="auth.canManageResources"`, um ícone svg análogo e um `<span>Perfis de revisão</span>`). Espelhe a marcação de um link existente da mesma seção.

- [ ] **Step 3: Verificação**

```bash
cd web && npm run type-check && npm run build
```
Expected: type-check sem erros; build conclui e grava em `../internal/transport/http/frontend/dist`.

Voltar à raiz e confirmar que o Go ainda compila com o SPA embutido:

```bash
go build ./...
```
Expected: sucesso.

- [ ] **Step 4: Commit**

```bash
git add web/src/router/index.ts web/src/components/AppLayout.vue internal/transport/http/frontend/dist/index.html
git commit -m "feat(web): rota e navegação de perfis de revisão"
```

NOTA: `dist/assets/` está no `.gitignore`; apenas `dist/index.html` é commitado (convenção do projeto). Inclua só o `index.html` no commit se ele mudar.

---

## Verificação final do milestone

- [ ] `cd web && npm run type-check` → sem erros.
- [ ] `cd web && npm run build` → sucesso.
- [ ] `go build ./...` → sucesso (SPA embutida).
- [ ] **Sanidade:** logado como super/gestor, o menu mostra "Perfis de revisão"; a tela lista/cria/edita/exclui perfis e gerencia atribuições por escopo, consumindo `/api/v1/review-profiles`.

## Notas para o executor

- **Espelhe, não invente:** leia cada arquivo-referência de prompts e replique estrutura, props, emits, `defineExpose`, classes CSS e handlers. A única diferença de domínio é o conjunto de campos (checkboxes + number em vez de rich text).
- **Tipos sempre de `@/api/reviewProfiles`** — nunca inline (convenção `web/CLAUDE.md`).
- **`auth.canManageResources`** é o gate usado pelos links de Prompts/AI configs; use o mesmo.
- **Não toque em `dist/assets/`** (gitignored); só `dist/index.html` entra no commit, se mudar.
- Se `npm run type-check` acusar algo, corrija antes de commitar — é o substituto do "teste" aqui.
