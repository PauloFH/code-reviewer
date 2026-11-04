<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · prompts</div>
        <h1 class="page-title">Prompts e atribuições</h1>
        <div class="page-sub">Defina os prompts usados pelo agente e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <!-- Prompts card -->
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Prompts</h2>
            <div class="card-sub">{{ prompts.length }} {{ prompts.length === 1 ? 'definido' : 'definidos' }}</div>
          </div>
          <button class="btn primary" @click="openCreate">
            <AppIcon name="plus" :size="13" />
            Novo prompt
          </button>
        </div>

        <div v-if="loading" class="loading-row">
          <span class="spinner" /> Carregando…
        </div>
        <div v-else-if="prompts.length === 0" class="empty-state">
          <div class="empty-title">Nenhum prompt cadastrado</div>
          <div class="empty-sub">Crie um prompt para customizar como o agente analisa o código.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Nome</th>
              <th>Criado em</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in prompts" :key="p.id">
              <td>
                <div class="name-cell">
                  <div class="prompt-icon">
                    <AppIcon name="document" :size="14" />
                  </div>
                  <div>
                    <div class="name">{{ p.name }}</div>
                    <div class="card-sub">criado em {{ fmtDate(p.created_at) }}</div>
                  </div>
                </div>
              </td>
              <td class="num">{{ fmtDate(p.created_at) }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn sm ghost" type="button" title="Editar" :aria-label="`Editar ${p.name}`" @click="openEdit(p)">
                    <AppIcon name="edit" :size="12" />
                  </button>
                  <button class="btn sm ghost danger" type="button" title="Excluir" :aria-label="`Excluir ${p.name}`" @click="deleteTarget = p">
                    <AppIcon name="trash" :size="12" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Assignments card -->
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Atribuições</h2>
            <div class="card-sub">resolução do prompt por escopo · ordem: projeto → cliente → global</div>
          </div>
          <button class="btn primary" @click="showAssignForm = true">
            <AppIcon name="plus" :size="13" />
            Nova atribuição
          </button>
        </div>

        <div v-if="assignments.length === 0" class="empty-state">
          <div class="empty-title">Nenhuma atribuição configurada</div>
          <div class="empty-sub">O sistema está usando a configuração padrão do ambiente. Crie uma atribuição para sobrepor o fallback em clientes ou projetos específicos.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th class="col-scope">Escopo</th>
              <th>Cliente</th>
              <th>Projeto</th>
              <th>Prompt</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in assignments" :key="a.id">
              <td>
                <span class="pill" :class="a.scope === 'global' ? 'accent' : 'skip'">
                  <span class="dot"></span>{{ scopeLabel(a.scope) }}
                </span>
              </td>
              <td class="muted">{{ a.client_name || '—' }}</td>
              <td class="muted">{{ a.project_ref || '—' }}</td>
              <td>{{ a.prompt_name }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn sm ghost danger" type="button" title="Remover atribuição" aria-label="Remover atribuição" @click="removeAssign(a)">
                    <AppIcon name="trash" :size="12" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Prompt Form Modal -->
    <PromptForm
      v-if="showForm"
      :initial="editingPrompt"
      ref="formRef"
      @close="showForm = false"
      @saved="handleSaved"
    />

    <!-- Assignment Form Modal -->
    <PromptAssignmentForm
      v-if="showAssignForm"
      ref="assignFormRef"
      @close="showAssignForm = false"
      @saved="handleAssignSaved"
    />

    <!-- Delete confirm -->
    <ConfirmModal
      v-if="deleteTarget"
      :name="deleteTarget.name"
      warning="Atribuições vinculadas serão removidas."
      :loading="deleting"
      @close="deleteTarget = null"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppIcon from '@/components/AppIcon.vue'
import PromptForm from '@/components/PromptForm.vue'
import PromptAssignmentForm from '@/components/PromptAssignmentForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { useToastStore } from '@/stores/toast'
import {
  listPrompts, createPrompt, updatePrompt, deletePrompt,
  listAssignments, setAssignment, removeAssignment,
  type Prompt, type PromptInput, type PromptAssignment, type AssignmentInput,
} from '@/api/prompts'

const toast = useToastStore()

const prompts = ref<Prompt[]>([])
const assignments = ref<PromptAssignment[]>([])
const loading = ref(false)
const deleting = ref(false)
const showForm = ref(false)
const showAssignForm = ref(false)
const editingPrompt = ref<Prompt | null>(null)
const deleteTarget = ref<Prompt | null>(null)
const formRef = ref<InstanceType<typeof PromptForm> | null>(null)
const assignFormRef = ref<InstanceType<typeof PromptAssignmentForm> | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [p, a] = await Promise.all([listPrompts(), listAssignments()])
    prompts.value = p
    assignments.value = a
  } catch {
    toast.push('Erro ao carregar prompts', 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingPrompt.value = null
  showForm.value = true
}

function openEdit(p: Prompt) {
  editingPrompt.value = p
  showForm.value = true
}

async function handleSaved(data: PromptInput & { _id?: number }) {
  formRef.value?.setSaving(true)
  try {
    if (data._id) {
      await updatePrompt(data._id, { name: data.name, system_prompt: data.system_prompt, output_template: data.output_template })
      toast.push('Prompt atualizado', 'success')
    } else {
      await createPrompt({ name: data.name, system_prompt: data.system_prompt, output_template: data.output_template })
      toast.push('Prompt criado', 'success')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'Erro ao salvar'
    formRef.value?.setError(msg)
  }
}

async function handleAssignSaved(data: AssignmentInput) {
  assignFormRef.value?.setSaving(true)
  try {
    await setAssignment(data)
    toast.push('Atribuição salva', 'success')
    showAssignForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'Erro ao salvar'
    assignFormRef.value?.setError(msg)
  }
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deletePrompt(deleteTarget.value.id)
    toast.push(`Prompt "${deleteTarget.value.name}" excluído`, 'success')
    deleteTarget.value = null
    await load()
  } catch {
    toast.push('Erro ao excluir', 'error')
  } finally {
    deleting.value = false
  }
}

async function removeAssign(a: PromptAssignment) {
  try {
    await removeAssignment(a.scope, a.client_name, a.project_ref)
    toast.push('Atribuição removida', 'success')
    await load()
  } catch {
    toast.push('Erro ao remover', 'error')
  }
}

function scopeLabel(scope: string) {
  if (scope === 'global') return 'Global'
  if (scope === 'client') return 'Cliente'
  return 'Projeto'
}

function fmtDate(s: string) {
  const d = new Date(s)
  return d.toLocaleDateString('pt-BR')
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
}

.loading-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  padding: 24px;
}

.empty-state {
  padding: 56px 32px;
  text-align: center;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.name {
  font-weight: 500;
  overflow-wrap: anywhere;
}

.col-scope {
  width: 110px;
}

.prompt-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  display: grid;
  place-items: center;
  color: var(--muted);
}
</style>