<template>
  <FormModal title="Nova atribuição" max-width="480px" :saving="saving" :error="error" @close="$emit('close')" @submit="onSubmit">
    <div class="field">
      <label :for="`${id}-target`">{{ targetLabel }}</label>
      <select :id="`${id}-target`" v-model="targetId" required>
        <option value="" disabled>Selecione…</option>
        <option v-for="t in targets" :key="t.id" :value="t.id">{{ t.name }}</option>
      </select>
    </div>
    <div class="field">
      <label :for="`${id}-scope`">Escopo</label>
      <select :id="`${id}-scope`" v-model="scope" required>
        <option v-for="s in SCOPES" :key="s.value" :value="s.value">{{ s.label }} ({{ s.hint }})</option>
      </select>
    </div>
    <div v-if="scope !== 'global'" class="field">
      <label :for="`${id}-client`">Cliente</label>
      <select :id="`${id}-client`" v-model="clientName" required>
        <option value="" disabled>Selecione um cliente</option>
        <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }} ({{ platformLabel(c.platform) }})</option>
      </select>
    </div>
    <div v-if="scope === 'project'" class="field">
      <label :for="`${id}-project`">Projeto (referência)</label>
      <input :id="`${id}-project`" v-model="projectRef" required placeholder="Ex: org/repo-name" />
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref, useId } from 'vue'
import FormModal from '@/components/FormModal.vue'
import type { AssignmentInput } from '@/api/assignments'
import { useSubmit } from '@/composables/useSubmit'
import { useClients } from '@/composables/useClients'
import { SCOPES, platformLabel, type Scope } from '@/domain'

const props = defineProps<{
  targets: { id: number; name: string }[]
  targetLabel: string
  save: (data: AssignmentInput) => Promise<void>
}>()
defineEmits<{ close: [] }>()

const id = useId()
const { saving, error, submit } = useSubmit()
const clients = useClients()
// '' deixa a opção desabilitada selecionada, e `required` barra o envio sem escolha.
const targetId = ref<number | ''>('')
const scope = ref<Scope>('global')
const clientName = ref('')
const projectRef = ref('')

function onSubmit() {
  if (targetId.value === '') return
  const data: AssignmentInput = {
    target_id: targetId.value,
    scope: scope.value,
    client_name: clientName.value,
    project_ref: projectRef.value,
  }
  submit(() => props.save(data))
}
</script>
