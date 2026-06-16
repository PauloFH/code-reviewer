<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · perfis de revisão</div>
        <h1 class="page-title">Perfis de revisão &amp; atribuições</h1>
        <div class="page-sub">Defina os perfis de revisão e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <!-- Profiles card -->
      <div class="card">
        <div class="card-head">
          <div>
            <div class="card-title">Perfis</div>
            <div class="card-sub">{{ profiles.length }} perfis definidos</div>
          </div>
          <button class="btn primary" @click="openCreate">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14"/>
            </svg>
            Novo perfil
          </button>
        </div>

        <div v-if="loading" class="loading-row">
          <span class="spinner" /> Carregando…
        </div>
        <div v-else-if="profiles.length === 0" class="empty-state">
          <div class="empty-title">Nenhum perfil cadastrado</div>
          <div class="empty-sub">Crie um perfil para configurar as camadas de revisão do agente.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Nome</th>
              <th>Camadas</th>
              <th>Max iter.</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in profiles" :key="p.id">
              <td>
                <div style="display: flex; align-items: center; gap: 10px;">
                  <div class="profile-icon">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                      <path d="M2 17l10 5 10-5"/>
                      <path d="M2 12l10 5 10-5"/>
                    </svg>
                  </div>
                  <div>
                    <div style="font-weight: 500;">{{ p.name }}</div>
                    <div class="card-sub">criado em {{ fmtDate(p.created_at) }}</div>
                  </div>
                </div>
              </td>
              <td>
                <div class="badges">
                  <span v-if="p.include_related" class="badge related">relacionado</span>
                  <span v-if="p.agentic" class="badge agentic">agêntico</span>
                  <span v-if="p.execution" class="badge execution">sandbox</span>
                  <span v-if="!p.include_related && !p.agentic && !p.execution" class="badge none">básico</span>
                </div>
              </td>
              <td class="num">{{ p.max_iterations || '—' }}</td>
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
            <div class="card-sub">resolução do perfil por escopo · ordem: projeto → cliente → global</div>
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
              <th>Perfil</th>
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
              <td>{{ a.profile_name }}</td>
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

    <!-- Profile Form Modal -->
    <ReviewProfileForm
      v-if="showForm"
      :initial="editingProfile"
      ref="formRef"
      @close="showForm = false"
      @saved="handleSaved"
    />

    <!-- Assignment Form Modal -->
    <ReviewProfileAssignmentForm
      v-if="showAssignForm"
      :profiles="profiles"
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
import ReviewProfileForm from '@/components/ReviewProfileForm.vue'
import ReviewProfileAssignmentForm from '@/components/ReviewProfileAssignmentForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { useToastStore } from '@/stores/toast'
import {
  listReviewProfiles, createReviewProfile, updateReviewProfile, deleteReviewProfile,
  listAssignments, setAssignment, removeAssignment,
  type ReviewProfile, type ReviewProfileInput, type ReviewProfileAssignment, type AssignmentInput,
} from '@/api/reviewProfiles'

const toast = useToastStore()

const profiles = ref<ReviewProfile[]>([])
const assignments = ref<ReviewProfileAssignment[]>([])
const loading = ref(false)
const deleting = ref(false)
const showForm = ref(false)
const showAssignForm = ref(false)
const editingProfile = ref<ReviewProfile | null>(null)
const deleteTarget = ref<ReviewProfile | null>(null)
const formRef = ref<InstanceType<typeof ReviewProfileForm> | null>(null)
const assignFormRef = ref<InstanceType<typeof ReviewProfileAssignmentForm> | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [p, a] = await Promise.all([listReviewProfiles(), listAssignments()])
    profiles.value = p
    assignments.value = a
  } catch {
    toast.push('Erro ao carregar perfis', 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingProfile.value = null
  showForm.value = true
}

function openEdit(p: ReviewProfile) {
  editingProfile.value = p
  showForm.value = true
}

async function handleSaved(data: ReviewProfileInput & { _id?: number }) {
  formRef.value?.setSaving(true)
  try {
    if (data._id) {
      await updateReviewProfile(data._id, {
        name: data.name,
        include_related: data.include_related,
        agentic: data.agentic,
        execution: data.execution,
        max_iterations: data.max_iterations,
      })
      toast.push('Perfil atualizado', 'success')
    } else {
      await createReviewProfile({
        name: data.name,
        include_related: data.include_related,
        agentic: data.agentic,
        execution: data.execution,
        max_iterations: data.max_iterations,
      })
      toast.push('Perfil criado', 'success')
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
    await deleteReviewProfile(deleteTarget.value.id)
    toast.push(`Perfil "${deleteTarget.value.name}" excluído`, 'success')
    deleteTarget.value = null
    await load()
  } catch {
    toast.push('Erro ao excluir', 'error')
  } finally {
    deleting.value = false
  }
}

async function removeAssign(a: ReviewProfileAssignment) {
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

.profile-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  display: grid;
  place-items: center;
  color: var(--muted);
}

.badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.badge {
  font-size: 11px;
  font-family: var(--font-mono);
  padding: 2px 7px;
  border-radius: 4px;
  font-weight: 500;
  letter-spacing: 0.02em;
}

.badge.related {
  background: rgba(99, 102, 241, 0.12);
  color: #818cf8;
}

.badge.agentic {
  background: rgba(16, 185, 129, 0.12);
  color: #34d399;
}

.badge.execution {
  background: rgba(245, 158, 11, 0.12);
  color: #fbbf24;
}

.badge.none {
  background: var(--bg-2);
  color: var(--muted);
}
</style>
