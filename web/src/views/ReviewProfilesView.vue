<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · perfis de revisão</div>
        <h1 class="page-title">Perfis de revisão e atribuições</h1>
        <div class="page-sub">Defina os perfis de revisão e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <!-- Profiles card -->
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Perfis</h2>
            <div class="card-sub">{{ profiles.length }} {{ profiles.length === 1 ? 'definido' : 'definidos' }}</div>
          </div>
          <button class="btn primary" @click="openCreate">
            <AppIcon name="plus" :size="13" />
            Novo perfil
          </button>
        </div>

        <SkeletonRows v-if="loading && !profiles.length" :cols="5" />
        <div v-else-if="profiles.length === 0" class="empty-state">
          <div class="empty-title">Nenhum perfil cadastrado</div>
          <div class="empty-sub">Crie um perfil para configurar as camadas de revisão do agente.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Nome</th>
              <th>Camadas</th>
              <th>Máx. iterações</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in profiles" :key="p.id">
              <td>
                <div class="name-cell">
                  <div class="profile-icon">
                    <AppIcon name="layers" :size="14" />
                  </div>
                  <div>
                    <div class="name">{{ p.name }}</div>
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
            <div class="card-sub">resolução do perfil por escopo · ordem: projeto → cliente → global</div>
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
import SkeletonRows from '@/components/SkeletonRows.vue'
import AppIcon from '@/components/AppIcon.vue'
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

.profile-icon {
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

.badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.badge {
  font-size: 11.5px;
  font-family: var(--font-mono);
  padding: 2px 7px;
  border-radius: var(--radius);
  border: 1px solid var(--hairline-strong);
  font-weight: 500;
  color: var(--text-2);
  background: var(--bg-2);
}

.badge.agentic {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}

/* Camada mais sensível (executa comandos): marca-texto pontual, sempre com --ink. */
.badge.execution {
  color: var(--ink);
  border-color: var(--hl-2);
  background: var(--highlight);
}

.badge.none {
  color: var(--muted);
  border-color: var(--hairline);
}
</style>
