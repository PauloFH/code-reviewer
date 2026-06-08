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
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          Novo usuário
        </button>
      </div>
    </div>

    <!-- Users table -->
    <div class="card">
      <div v-if="loading" class="loading-row">
        <span class="spinner" /> Carregando…
      </div>
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
                  <div class="card-sub">{{ u.role }}</div>
                </div>
              </div>
            </td>
            <td>
              <span class="pill" :class="u.role === 'super' ? 'accent' : 'skip'">
                <span class="dot"></span>{{ u.role }}
              </span>
            </td>
            <td class="num">{{ fmtDate(u.created_at) }}</td>
            <td class="num muted">—</td>
            <td>
              <div class="row-actions">
                <button class="btn sm ghost" @click="editUser(u)">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17 3l4 4-12 12H5v-4z"/>
                    <path d="M14.5 5.5l4 4"/>
                  </svg>
                </button>
                <button class="btn sm ghost danger" @click="confirmDelete(u)">
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

    <!-- Create/Edit Modal -->
    <div v-if="showForm" class="overlay" @click.self="closeForm">
      <div class="modal">
        <div class="modal-head">
          <h2>{{ editing ? 'Editar Usuário' : 'Novo Usuário' }}</h2>
          <button class="icon-btn" @click="closeForm">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleSubmit" class="modal-body">
          <div class="field">
            <label>Usuário</label>
            <input v-model="form.username" type="text" required />
          </div>

          <div class="field">
            <label>Senha {{ editing ? '(deixe vazio para manter)' : '' }}</label>
            <input v-model="form.password" :type="editing ? 'password' : 'password'" :required="!editing" />
          </div>

          <div class="field">
            <label>Papel</label>
            <select v-model="form.role" required>
              <option value="super">Super</option>
              <option value="gestor">Gestor</option>
              <option value="padrao">Padrão</option>
            </select>
          </div>

          <p v-if="formError" class="error">{{ formError }}</p>

          <div class="modal-actions">
            <button type="button" class="btn" @click="closeForm">Cancelar</button>
            <button type="submit" class="btn primary" :disabled="saving">
              <span v-if="saving" class="spinner" />
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

/* Modal */
.modal {
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 420px;
  overflow: hidden;
}

.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 22px;
  border-bottom: 1px solid var(--hairline);
}

.modal-head h2 {
  font-size: 15px;
  font-weight: 500;
  color: var(--text);
  margin: 0;
}

.modal-body {
  padding: 22px;
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
  font-size: 12px;
  color: var(--muted);
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.08em;
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
  background: rgba(248, 113, 113, 0.08);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: var(--radius);
  margin: 0;
}

.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 8px;
}
</style>