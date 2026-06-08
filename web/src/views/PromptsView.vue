<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · prompts</div>
        <h1 class="page-title">Prompts &amp; atribuições</h1>
        <div class="page-sub">Defina os prompts usados pelo agente e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <!-- Prompts card -->
      <div class="card">
        <div class="card-head">
          <div>
            <div class="card-title">Prompts</div>
            <div class="card-sub">{{ prompts.length }} prompts definidos</div>
          </div>
          <button class="btn primary" @click="openCreate">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14"/>
            </svg>
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
                <div style="display: flex; align-items: center; gap: 10px;">
                  <div class="prompt-icon">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                      <path d="M14 2v6h6"/>
                      <path d="M8 13h8M8 17h6"/>
                    </svg>
                  </div>
                  <div>
                    <div style="font-weight: 500;">{{ p.name }}</div>
                    <div class="card-sub">criado em {{ fmtDate(p.created_at) }}</div>
                  </div>
                </div>
              </td>
              <td class="num">{{ fmtDate(p.created_at) }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn sm ghost" @click="openEdit(p)">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M17 3l4 4-12 12H5v-4z"/>
                      <path d="M14.5 5.5l4 4"/>
                    </svg>
                  </button>
                  <button class="btn sm ghost danger" @click="deleteTarget = p">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M3 6h18"/>
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                      <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
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
            <div class="card-title">Atribuições</div>
            <div class="card-sub">resolução do prompt por escopo · ordem: projeto → cliente → global</div>
          </div>
          <button class="btn primary" @click="showAssignForm = true">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14"/>
            </svg>
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
              <th style="width: 110px;">Escopo</th>
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
                  <button class="btn sm ghost danger" @click="removeAssign(a)">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M3 6h18"/>
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                      <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
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

.prompt-icon {
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