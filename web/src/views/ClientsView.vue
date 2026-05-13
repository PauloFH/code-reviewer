<template>
  <div>
    <div class="header">
      <h2>Clients</h2>
      <button @click="openCreate">+ New Client</button>
    </div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Platform</th>
            <th>Base URL</th>
            <th>Username</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in clients" :key="c.name">
            <td>{{ c.name }}</td>
            <td><span class="badge">{{ c.platform }}</span></td>
            <td>{{ c.base_url }}</td>
            <td>{{ c.username }}</td>
            <td class="actions">
              <button @click="openEdit(c)">Edit</button>
              <button class="danger" @click="confirmDelete(c.name)">Delete</button>
            </td>
          </tr>
          <tr v-if="clients.length === 0">
            <td colspan="5" class="empty">No clients configured</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create / Edit modal -->
    <div v-if="modal" class="overlay" @click.self="modal = null">
      <div class="modal">
        <h3>{{ editingName ? 'Edit Client' : 'New Client' }}</h3>
        <p v-if="formError" class="error">{{ formError }}</p>
        <form @submit.prevent="saveClient">
          <label>Name <input v-model="form.name" required /></label>
          <label>
            Platform
            <select v-model="form.platform" required>
              <option value="gitlab">GitLab</option>
              <option value="gitlab-selfhosted">GitLab Self-hosted</option>
              <option value="github">GitHub</option>
              <option value="gitea">Gitea</option>
              <option value="gitea-selfhosted">Gitea Self-hosted</option>
            </select>
          </label>
          <label>Base URL <input v-model="form.base_url" placeholder="https://gitlab.com" required /></label>
          <label>API Token <input v-model="form.token" type="password" /></label>
          <label>Webhook Secret <input v-model="form.webhook_secret" type="password" /></label>
          <label>Watch Username <input v-model="form.username" required /></label>
          <div class="modal-actions">
            <button type="button" @click="modal = null">Cancel</button>
            <button type="submit" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete confirm modal -->
    <div v-if="deleteTarget" class="overlay" @click.self="deleteTarget = null">
      <div class="modal confirm">
        <p>Delete client <strong>{{ deleteTarget }}</strong>?</p>
        <div class="modal-actions">
          <button @click="deleteTarget = null">Cancel</button>
          <button class="danger" @click="doDelete">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listClients, createClient, updateClient, deleteClient, type Client, type ClientInput } from '@/api/clients'

const clients = ref<Client[]>([])
const modal = ref<'create' | 'edit' | null>(null)
const editingName = ref('')
const deleteTarget = ref('')
const saving = ref(false)
const formError = ref('')

const emptyForm = (): ClientInput => ({
  name: '', platform: 'gitlab', base_url: '', token: '', webhook_secret: '', username: '',
})
const form = ref<ClientInput>(emptyForm())

onMounted(load)

async function load() {
  clients.value = await listClients()
}

function openCreate() {
  form.value = emptyForm()
  editingName.value = ''
  formError.value = ''
  modal.value = 'create'
}

function openEdit(c: Client) {
  form.value = { name: c.name, platform: c.platform, base_url: c.base_url, token: '', webhook_secret: '', username: c.username }
  editingName.value = c.name
  formError.value = ''
  modal.value = 'edit'
}

async function saveClient() {
  formError.value = ''
  saving.value = true
  try {
    if (editingName.value) {
      await updateClient(editingName.value, form.value)
    } else {
      await createClient(form.value)
    }
    modal.value = null
    await load()
  } catch (e: unknown) {
    formError.value = (e as { response?: { data?: string } })?.response?.data || 'An error occurred'
  } finally {
    saving.value = false
  }
}

function confirmDelete(name: string) {
  deleteTarget.value = name
}

async function doDelete() {
  if (!deleteTarget.value) return
  await deleteClient(deleteTarget.value)
  deleteTarget.value = ''
  await load()
}
</script>

<style scoped>
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
.header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
.header button { padding: .5rem 1rem; background: #6366f1; color: white; border: none; border-radius: 6px; cursor: pointer; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .6rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; }
td { padding: .6rem .75rem; border-bottom: 1px solid #1e293b; color: #cbd5e1; }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; background: #1e3a5f; color: #93c5fd; }
.actions { display: flex; gap: .5rem; }
.actions button {
  padding: .3rem .6rem;
  border: 1px solid #334155;
  background: #1e293b;
  color: #e2e8f0;
  border-radius: 4px;
  cursor: pointer;
  font-size: .8rem;
}
.actions button.danger { color: #f87171; border-color: #7f1d1d; }
.empty { text-align: center; color: #475569; padding: 2rem; }
.overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; width: 100%; max-width: 480px;
}
.modal h3 { font-size: 1.1rem; font-weight: 600; color: #f8fafc; margin-bottom: 1rem; }
.modal form { display: flex; flex-direction: column; gap: .75rem; }
.modal label { display: flex; flex-direction: column; gap: .3rem; font-size: .8rem; color: #94a3b8; }
.modal input, .modal select {
  padding: .5rem .75rem; background: #0f172a; border: 1px solid #475569;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
.modal-actions { display: flex; justify-content: flex-end; gap: .75rem; margin-top: .5rem; }
.modal-actions button {
  padding: .5rem 1.1rem; border-radius: 6px; cursor: pointer; font-size: .875rem;
  border: 1px solid #334155; background: #334155; color: #e2e8f0;
}
.modal-actions button[type="submit"] { background: #6366f1; border-color: #6366f1; color: white; }
.modal-actions button.danger { background: #7f1d1d; border-color: #7f1d1d; color: #fca5a5; }
.confirm { text-align: center; }
.confirm p { color: #e2e8f0; margin-bottom: 1.5rem; }
.error { color: #f87171; font-size: .8rem; margin-bottom: .5rem; }
</style>
