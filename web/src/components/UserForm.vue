<template>
  <FormModal :title="initial ? 'Editar usuário' : 'Novo usuário'" max-width="420px" :saving="saving" :error="error" @close="$emit('close')" @submit="onSubmit">
    <div class="field">
      <label for="user-username">Usuário</label>
      <input id="user-username" v-model="form.username" type="text" required />
    </div>
    <div class="field">
      <label for="user-password">Senha {{ initial ? '(deixe vazio para manter)' : '' }}</label>
      <input id="user-password" v-model="form.password" type="password" :required="!initial" />
    </div>
    <div class="field">
      <label for="user-role">Papel</label>
      <select id="user-role" v-model="form.role" required>
        <option v-for="r in ROLES" :key="r.value" :value="r.value">{{ r.label }}</option>
      </select>
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FormModal from '@/components/FormModal.vue'
import { useSubmit } from '@/composables/useSubmit'
import { ROLES } from '@/domain'
import type { User, UserInput } from '@/api/users'

const props = defineProps<{
  initial: User | null
  save: (data: UserInput) => Promise<void>
}>()
defineEmits<{ close: [] }>()

const { saving, error, submit } = useSubmit()
const form = ref({ username: props.initial?.username ?? '', password: '', role: props.initial?.role ?? 'padrao' })

function onSubmit() {
  const { username, password, role } = form.value
  submit(() => props.save({ username, password: password || undefined, role }))
}
</script>
