<template>
  <div class="page">
    <div class="header">
      <h2>Clients</h2>
      <button @click="openCreate">+ New Client</button>
    </div>

    <div v-if="loading" class="loading-row">
      <span class="spinner" /> Loading…
    </div>
    <div v-else class="table-wrap">
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
            <td class="url">{{ c.base_url }}</td>
            <td>{{ c.username }}</td>
            <td class="actions">
              <button @click="openEdit(c)">Edit</button>
              <button class="danger" @click="deleteTarget = c.name">Delete</button>
            </td>
          </tr>
          <tr v-if="clients.length === 0">
            <td colspan="5" class="empty">No clients configured</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create / Edit modal -->
    <ClientForm
      v-if="showForm"
      :initial="editingClient"
      ref="formRef"
      @close="showForm = false"
      @saved="handleSaved"
    />

    <!-- Delete confirm -->
    <div v-if="deleteTarget" class="overlay" @click.self="deleteTarget = ''">
      <div class="confirm-modal">
        <p>Delete client <strong>{{ deleteTarget }}</strong>?</p>
        <div class="confirm-actions">
          <button @click="deleteTarget = ''">Cancel</button>
          <button class="danger" :disabled="deleting" @click="doDelete">
            <span v-if="deleting" class="spinner" />
            {{ deleting ? 'Deleting…' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ClientForm from '@/components/ClientForm.vue'
import { useToastStore } from '@/stores/toast'
import { listClients, createClient, updateClient, deleteClient, type Client, type ClientInput } from '@/api/clients'

const toast = useToastStore()

const clients = ref<Client[]>([])
const loading = ref(false)
const deleting = ref(false)
const showForm = ref(false)
const editingClient = ref<Client | null>(null)
const deleteTarget = ref('')
const formRef = ref<InstanceType<typeof ClientForm> | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    clients.value = await listClients()
  } catch {
    toast.push('Failed to load clients', 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingClient.value = null
  showForm.value = true
}

function openEdit(c: Client) {
  editingClient.value = c
  showForm.value = true
}

async function handleSaved(data: ClientInput & { _oldName?: string }) {
  formRef.value?.setSaving(true)
  try {
    if (data._oldName) {
      const { _oldName, ...rest } = data
      await updateClient(_oldName, rest)
      toast.push('Client updated', 'success')
    } else {
      await createClient(data)
      toast.push('Client created', 'success')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'An error occurred'
    formRef.value?.setError(msg)
    formRef.value?.setSaving(false)
  }
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deleteClient(deleteTarget.value)
    toast.push(`Client "${deleteTarget.value}" deleted`, 'success')
    deleteTarget.value = ''
    await load()
  } catch {
    toast.push('Failed to delete client', 'error')
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 1.25rem; }
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
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
.url { font-size: .8rem; color: #94a3b8; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; background: #1e3a5f; color: #93c5fd; }
.actions { display: flex; gap: .5rem; }
.actions button {
  padding: .3rem .6rem; border: 1px solid #334155; background: #1e293b;
  color: #e2e8f0; border-radius: 4px; cursor: pointer; font-size: .8rem;
  display: flex; align-items: center; gap: .3rem;
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
.confirm-modal p { color: #e2e8f0; margin-bottom: 1.5rem; }
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
