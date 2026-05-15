<template>
  <div class="users-view">
    <div class="header">
      <h1>Usuários</h1>
      <button class="btn-primary" @click="showForm = true">+ Novo Usuário</button>
    </div>

    <table v-if="users.length" class="table">
      <thead>
        <tr>
          <th>Usuário</th>
          <th>Role</th>
          <th>Criado em</th>
          <th>Ações</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="u in users" :key="u.id">
          <td>{{ u.username }}</td>
          <td><span :class="['badge', 'badge-' + u.role]">{{ u.role }}</span></td>
          <td>{{ formatDate(u.created_at) }}</td>
          <td>
            <button class="btn-sm" @click="editUser(u)">Editar</button>
            <button class="btn-sm btn-danger" @click="confirmDelete(u)">Excluir</button>
          </td>
        </tr>
      </tbody>
    </table>

    <p v-else-if="!loading" class="empty">Nenhum usuário encontrado.</p>

    <!-- Create/Edit Modal -->
    <div v-if="showForm" class="modal-overlay" @click.self="closeForm">
      <div class="modal">
        <h2>{{ editing ? 'Editar Usuário' : 'Novo Usuário' }}</h2>
        <form @submit.prevent="handleSubmit">
          <label>
            Usuário
            <input v-model="form.username" type="text" required />
          </label>
          <label>
            Senha {{ editing ? '(deixe vazio para manter)' : '' }}
            <input v-model="form.password" type="password" :required="!editing" />
          </label>
          <label>
            Role
            <select v-model="form.role" required>
              <option value="super">Super</option>
              <option value="gestor">Gestor</option>
              <option value="padrao">Padrão</option>
            </select>
          </label>
          <p v-if="formError" class="error">{{ formError }}</p>
          <div class="form-actions">
            <button type="button" class="btn-secondary" @click="closeForm">Cancelar</button>
            <button type="submit" class="btn-primary" :disabled="saving">
              {{ saving ? 'Salvando...' : 'Salvar' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listUsers, createUser, updateUser, deleteUser, type User } from '@/api/users'
import { useToastStore } from '@/stores/toast'

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

async function confirmDelete(u: User) {
  if (!confirm(`Excluir usuário "${u.username}"?`)) return
  try {
    await deleteUser(u.id)
    toast.push('Usuário excluído', 'success')
    await load()
  } catch (err: any) {
    toast.push(err.response?.data || 'Erro ao excluir', 'error')
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('pt-BR')
}

onMounted(load)
</script>

<style scoped>
.users-view { max-width: 900px; }
.header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
h1 { font-size: 1.5rem; color: #e2e8f0; margin: 0; }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td { padding: .6rem .75rem; text-align: left; border-bottom: 1px solid #334155; }
.table th { color: #94a3b8; font-size: .8rem; text-transform: uppercase; }
.table td { color: #e2e8f0; font-size: .9rem; }

.badge { padding: .2rem .5rem; border-radius: 4px; font-size: .75rem; font-weight: 600; }
.badge-super { background: #6366f1; color: #fff; }
.badge-gestor { background: #0ea5e9; color: #fff; }
.badge-padrao { background: #475569; color: #e2e8f0; }

.btn-primary { background: #6366f1; color: #fff; border: none; padding: .5rem 1rem; border-radius: 6px; cursor: pointer; font-size: .85rem; }
.btn-primary:hover { background: #4f46e5; }
.btn-primary:disabled { opacity: .6; cursor: not-allowed; }
.btn-secondary { background: none; border: 1px solid #475569; color: #94a3b8; padding: .5rem 1rem; border-radius: 6px; cursor: pointer; font-size: .85rem; }
.btn-sm { background: none; border: 1px solid #475569; color: #94a3b8; padding: .25rem .5rem; border-radius: 4px; cursor: pointer; font-size: .8rem; margin-right: .25rem; }
.btn-sm:hover { background: #334155; }
.btn-danger { border-color: #ef4444; color: #ef4444; }
.btn-danger:hover { background: #ef4444; color: #fff; }

.empty { color: #64748b; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 300; }
.modal { background: #1e293b; border: 1px solid #334155; border-radius: 8px; padding: 1.5rem; width: 100%; max-width: 400px; }
.modal h2 { margin: 0 0 1rem; color: #e2e8f0; font-size: 1.1rem; }
.modal label { display: block; margin-bottom: .75rem; color: #94a3b8; font-size: .85rem; }
.modal input, .modal select { display: block; width: 100%; margin-top: .25rem; padding: .5rem; background: #0f172a; border: 1px solid #334155; border-radius: 4px; color: #e2e8f0; font-size: .9rem; }
.form-actions { display: flex; gap: .5rem; justify-content: flex-end; margin-top: 1rem; }
.error { color: #ef4444; font-size: .85rem; margin: .5rem 0 0; }
</style>
