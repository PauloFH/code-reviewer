<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Acesso</div>
        <h1 class="page-title">Usuários</h1>
        <div class="page-sub">Operadores com acesso ao painel. Permissões herdadas do papel atribuído.</div>
      </div>
      <div class="page-actions">
        <button class="btn primary" type="button" @click="openForm(null)">
          <AppIcon name="plus" :size="13" />
          Novo usuário
        </button>
      </div>
    </div>

    <div class="card">
      <SkeletonRows v-if="loading && !users.length" :cols="COLUMNS.length" />
      <div v-else-if="users.length === 0" class="empty-state">
        <div class="empty-title">Nenhum usuário encontrado</div>
        <div class="empty-sub">Crie um usuário para dar acesso ao painel.</div>
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th v-for="c in COLUMNS" :key="c">{{ c }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>
              <div class="name-cell">
                <div class="avatar">{{ u.username.slice(0, 2).toUpperCase() }}</div>
                <div class="name">{{ u.username }}</div>
              </div>
            </td>
            <td>
              <span class="pill" :class="u.role === 'super' ? 'accent' : 'skip'">
                <span class="dot"></span>{{ roleLabel(u.role) }}
              </span>
            </td>
            <td class="num">{{ formatDate(u.created_at) }}</td>
            <td>
              <div class="row-actions">
                <button class="btn sm ghost" type="button" title="Editar" :aria-label="`Editar ${u.username}`" @click="openForm(u)">
                  <AppIcon name="edit" :size="12" />
                </button>
                <button class="btn sm ghost danger" type="button" title="Excluir" :aria-label="`Excluir ${u.username}`" @click="deleteTarget = u">
                  <AppIcon name="trash" :size="12" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <UserForm v-if="showForm" :initial="editing" :save="save" @close="showForm = false" />

    <ConfirmModal
      v-if="deleteTarget"
      :name="deleteTarget.username"
      :loading="deleting"
      @close="deleteTarget = null"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import UserForm from '@/components/UserForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import AppIcon from '@/components/AppIcon.vue'
import { listUsers, createUser, updateUser, deleteUser, type User, type UserInput } from '@/api/users'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'
import { formatDate } from '@/brand/format'
import { roleLabel } from '@/domain'

const COLUMNS = ['Usuário', 'Papel', 'Criado em', '']

const toast = useToastStore()
const users = ref<User[]>([])
const loading = ref(true)
const showForm = ref(false)
const editing = ref<User | null>(null)
const deleteTarget = ref<User | null>(null)
const deleting = ref(false)

async function load() {
  loading.value = true
  try {
    users.value = await listUsers()
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao carregar usuários'), 'error')
  } finally {
    loading.value = false
  }
}

function openForm(u: User | null) {
  editing.value = u
  showForm.value = true
}

async function save(data: UserInput) {
  if (editing.value) await updateUser(editing.value.id, data)
  else await createUser(data)
  toast.push(editing.value ? 'Usuário atualizado' : 'Usuário criado', 'success')
  showForm.value = false
  await load()
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deleteUser(deleteTarget.value.id)
    toast.push('Usuário excluído', 'success')
    deleteTarget.value = null
    await load()
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao excluir'), 'error')
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--avatar-bg);
  border: 1px solid var(--avatar-border);
  display: grid;
  place-items: center;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--avatar-text);
  flex-shrink: 0;
}
</style>
