<template>
  <FormModal :title="initial ? 'Editar cliente' : 'Novo cliente'" max-width="480px" :saving="saving" :error="error" @close="$emit('close')" @submit="submit(() => save(form))">
    <div class="field">
      <label for="client-name">Nome</label>
      <input id="client-name" v-model="form.name" required :disabled="!!initial" />
    </div>
    <div class="field">
      <label for="client-platform">Plataforma</label>
      <select id="client-platform" v-model="form.platform" required>
        <option v-for="p in PLATFORMS" :key="p.value" :value="p.value">{{ p.label }}</option>
      </select>
    </div>
    <div class="field">
      <label for="client-base-url">URL base</label>
      <input id="client-base-url" v-model="form.base_url" placeholder="https://gitlab.com" required />
    </div>
    <div class="field">
      <label for="client-token">Token de API</label>
      <input id="client-token" v-model="form.token" type="password" :placeholder="initial ? 'deixe em branco para manter' : ''" />
    </div>
    <div class="field">
      <label for="client-secret">Segredo do webhook</label>
      <input id="client-secret" v-model="form.webhook_secret" type="password" :placeholder="initial ? 'deixe em branco para manter' : ''" />
    </div>
    <div class="field">
      <label for="client-username">Usuário monitorado</label>
      <input id="client-username" v-model="form.username" required />
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FormModal from '@/components/FormModal.vue'
import { useSubmit } from '@/composables/useSubmit'
import { PLATFORMS } from '@/domain'
import type { Client, ClientInput } from '@/api/clients'

const props = defineProps<{
  initial: Client | null
  save: (data: ClientInput) => Promise<void>
}>()
defineEmits<{ close: [] }>()

const { saving, error, submit } = useSubmit()
const form = ref<ClientInput>({
  name: props.initial?.name ?? '',
  platform: props.initial?.platform ?? 'gitlab',
  base_url: props.initial?.base_url ?? '',
  token: '',
  webhook_secret: '',
  username: props.initial?.username ?? '',
})
</script>
