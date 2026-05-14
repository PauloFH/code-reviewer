<template>
  <div class="page">
    <!-- Prompts Section -->
    <div class="section">
      <div class="header">
        <h2>Prompts</h2>
        <button @click="openCreate">+ Novo Prompt</button>
      </div>

      <div v-if="loading" class="loading-row">
        <span class="spinner" /> Carregando…
      </div>
      <div v-else class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Criado em</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in prompts" :key="p.id">
              <td>{{ p.name }}</td>
              <td class="date">{{ fmtDate(p.created_at) }}</td>
              <td class="actions">
                <button @click="openEdit(p)">Editar</button>
                <button class="danger" @click="deleteTarget = p">Excluir</button>
              </td>
            </tr>
            <tr v-if="prompts.length === 0">
              <td colspan="3" class="empty">Nenhum prompt cadastrado</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Assignments Section -->
    <div class="section">
      <div class="header">
        <h2>Atribuições</h2>
        <button @click="showAssignForm = true">+ Nova Atribuição</button>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Escopo</th>
              <th>Client</th>
              <th>Projeto</th>
              <th>Prompt</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in assignments" :key="a.id">
              <td><span class="badge" :class="a.scope">{{ scopeLabel(a.scope) }}</span></td>
              <td>{{ a.client_name || '—' }}</td>
              <td>{{ a.project_ref || '—' }}</td>
              <td>{{ a.prompt_name }}</td>
              <td class="actions">
                <button class="danger" @click="removeAssign(a)">Remover</button>
              </td>
            </tr>
            <tr v-if="assignments.length === 0">
              <td colspan="5" class="empty">Nenhuma atribuição configurada</td>
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
    <div v-if="deleteTarget" class="overlay" @click.self="deleteTarget = null">
      <div class="confirm-modal">
        <p>Excluir prompt <strong>{{ deleteTarget.name }}</strong>?</p>
        <p class="warn">Atribuições vinculadas serão removidas.</p>
        <div class="confirm-actions">
          <button @click="deleteTarget = null">Cancelar</button>
          <button class="danger" :disabled="deleting" @click="doDelete">
            <span v-if="deleting" class="spinner" />
            {{ deleting ? 'Excluindo…' : 'Excluir' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PromptForm from '@/components/PromptForm.vue'
import PromptAssignmentForm from '@/components/PromptAssignmentForm.vue'
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
  if (scope === 'client') return 'Client'
  return 'Projeto'
}

function fmtDate(s: string) {
  const d = new Date(s)
  return d.toLocaleDateString('pt-BR')
}
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 2rem; }
.section { display: flex; flex-direction: column; gap: 1rem; }
h2 { font-size: 1.3rem; font-weight: 700; color: #f8fafc; }
.header { display: flex; align-items: center; justify-content: space-between; }
.header button {
  padding: .5rem 1rem; background: #6366f1; color: white;
  border: none; border-radius: 6px; cursor: pointer; font-size: .875rem;
}
.loading-row { display: flex; align-items: center; gap: .5rem; color: #64748b; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .6rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; white-space: nowrap; }
td { padding: .6rem .75rem; border-bottom: 1px solid #0f172a; color: #cbd5e1; }
.date { font-size: .8rem; color: #94a3b8; }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; }
.badge.global { background: #1e3a5f; color: #93c5fd; }
.badge.client { background: #1e3b1e; color: #86efac; }
.badge.project { background: #3b1e3b; color: #d8b4fe; }
.actions { display: flex; gap: .5rem; }
.actions button {
  padding: .3rem .6rem; border: 1px solid #334155; background: #1e293b;
  color: #e2e8f0; border-radius: 4px; cursor: pointer; font-size: .8rem;
}
.actions button.danger { color: #f87171; border-color: #7f1d1d; }
.empty { text-align: center; color: #475569; padding: 2rem; }
.overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.confirm-modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; max-width: 360px; width: 100%; text-align: center;
}
.confirm-modal p { color: #e2e8f0; margin-bottom: .5rem; }
.confirm-modal .warn { color: #fbbf24; font-size: .8rem; margin-bottom: 1rem; }
.confirm-actions { display: flex; justify-content: center; gap: .75rem; }
.confirm-actions button {
  padding: .5rem 1.1rem; border-radius: 6px; cursor: pointer; font-size: .875rem;
  border: 1px solid #334155; background: #334155; color: #e2e8f0;
  display: flex; align-items: center; gap: .4rem;
}
.confirm-actions button.danger { background: #7f1d1d; border-color: #7f1d1d; color: #fca5a5; }
.confirm-actions button:disabled { opacity: .6; cursor: not-allowed; }
.spinner {
  width: 12px; height: 12px;
  border: 2px solid rgba(255,255,255,.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
