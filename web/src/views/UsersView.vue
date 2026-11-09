<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Acesso</div>
        <h1 class="page-title">Usuários</h1>
        <div class="page-sub">Operadores com acesso ao painel. Permissões herdadas do papel atribuído.</div>
      </div>
      <div class="page-actions">
        <button class="btn primary" @click="openCreate">
          <AppIcon name="plus" :size="13" />
          Novo usuário
        </button>
      </div>
    </div>

    <!-- Users table -->
    <div class="card">
      <SkeletonRows v-if="loading && !users.length" :cols="6" />
      <div v-else-if="users.length === 0" class="empty-state">
        <div class="empty-title">Nenhum usuário encontrado</div>
        <div class="empty-sub">Crie um usuário para dar acesso ao painel.</div>
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>Usuário</th>
            <th>Papel</th>
            <th>Criado em</th>
            <th>Última atividade</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>
              <div style="display: flex; align-items: center; gap: 12px;">
                <div class="avatar">{{ u.username.slice(0, 2).toUpperCase() }}</div>
                <div>
                  <div style="font-weight: 500;">{{ u.username }}</div>
                  <div class="card-sub">{{ formatRole(u.role) }}</div>
                </div>
              </div>
            </td>
            <td>
              <span class="pill" :class="u.role === 'super' ? 'accent' : 'skip'">
                <span class="dot"></span>{{ formatRole(u.role) }}
              </span>
            </td>
            <td class="num">{{ fmtDate(u.created_at) }}</td>
            <td class="num muted">—</td>
            <td>
              <div class="row-actions">
                <button class="btn sm ghost" title="Editar" aria-label="Editar" @click="editUser(u)">
                  <AppIcon name="edit" :size="12" />
                </button>
                <button class="btn sm ghost danger" title="Excluir" aria-label="Excluir" @click="confirmDelete(u)">
                  <AppIcon name="trash" :size="12" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Modal -->
    <Modal v-if="showForm" :title="editing ? 'Editar usuário' : 'Novo usuário'" max-width="420px" @close="closeForm">
      <form id="user-form" class="form" @submit.prevent="handleSubmit">
        <div class="field">
          <label for="user-username">Usuário</label>
          <input id="user-username" v-model="form.username" type="text" required />
        </div>

        <div class="field">
          <label for="user-password">Senha {{ editing ? '(deixe vazio para manter)' : '' }}</label>
          <input id="user-password" v-model="form.password" type="password" :required="!editing" />
        </div>

        <div class="field">
          <label for="user-role">Papel</label>
          <select id="user-role" v-model="form.role" required>
            <option value="super">Super</option>
            <option value="gestor">Gestor</option>
            <option value="padrao">Padrão</option>
          </select>
        </div>

        <p v-if="formError" class="error" role="alert">{{ formError }}</p>
      </form>
      <template #footer>
        <button type="button" class="btn" @click="closeForm">Cancelar</button>
        <button type="submit" form="user-form" class="btn primary" :disabled="saving">
          <span v-if="saving" class="spinner" />
          {{ saving ? 'Salvando…' : 'Salvar' }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import { listUsers, createUser, updateUser, deleteUser, type User } from '@/api/users'
import { useToastStore } from '@/stores/toast'
import Modal from '@/components/Modal.vue'
import AppIcon from '@/components/AppIcon.vue'
import { formatRole } from '@/brand/format'

const toast = useToastStore()
const users = ref<User[]>([])
const loading = ref(true)
const showForm = ref(false)
const editing = ref<User | null>(null)
const saving = ref(false)
const formError = ref('')

const form = ref({
  username: '',
  password: '',
  role: 'padrao' as 'super' | 'gestor' | 'padrao',
})

async function load() {
  loading.value = true
  try {
    users.value = await listUsers()
  } catch {
    toast.push('Erro ao carregar usuários', 'error')
  } finally {
    loading.value = false
  }
}

function editUser(u: User) {
  editing.value = u
  form.value = { username: u.username, password: '', role: u.role }
  formError.value = ''
  showForm.value = true
}

function openCreate() {
  editing.value = null
  form.value = { username: '', password: '', role: 'padrao' }
  formError.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editing.value = null
  form.value = { username: '', password: '', role: 'padrao' }
  formError.value = ''
}

async function handleSubmit() {
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateUser(editing.value.id, {
        username: form.value.username,
        password: form.value.password || undefined,
        role: form.value.role,
      })
      toast.push('Usuário atualizado', 'success')
    } else {
      await createUser({
        username: form.value.username,
        password: form.value.password,
        role: form.value.role,
      })
      toast.push('Usuário criado', 'success')
    }
    closeForm()
    await load()
  } catch (err: any) {
    formError.value = err.response?.data || 'Erro ao salvar'
  } finally {
    saving.value = false
  }
}

function confirmDelete(u: User) {
  if (!confirm(`Excluir usuário "${u.username}"?`)) return
  doDelete(u)
}

async function doDelete(u: User) {
  try {
    await deleteUser(u.id)
    toast.push('Usuário excluído', 'success')
    await load()
  } catch (err: any) {
    toast.push(err.response?.data || 'Erro ao excluir', 'error')
  }
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString('pt-BR')
}

onMounted(load)
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

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 13px;
  color: var(--muted);
}

.field input,
.field select {
  padding: 10px 12px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: var(--font-sans);
  transition: border-color 0.12s;
}

.field input:focus,
.field select:focus {
  outline: none;
  border-color: var(--accent);
}

.field select {
  cursor: pointer;
}

.error {
  color: var(--danger);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--bg-2);
  border: 1px solid var(--danger);
  border-radius: var(--radius);
  margin: 0;
  overflow-wrap: anywhere;
}
</style>