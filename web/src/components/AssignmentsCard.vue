<template>
  <div class="card">
    <div class="card-head">
      <div>
        <h2 class="card-title">Atribuições</h2>
        <div class="card-sub">resolução por escopo · ordem: projeto → cliente → global</div>
      </div>
      <button class="btn primary" type="button" @click="adding = true">
        <AppIcon name="plus" :size="13" />
        Nova atribuição
      </button>
    </div>

    <SkeletonRows v-if="loading && !assignments.length" :cols="columns.length" />
    <div v-else-if="assignments.length === 0" class="empty-state">
      <div class="empty-title">Nenhuma atribuição configurada</div>
      <div class="empty-sub">{{ emptySub }}</div>
    </div>
    <table v-else class="table">
      <thead>
        <tr>
          <th v-for="c in columns" :key="c" :class="{ 'col-scope': c === 'Escopo' }">{{ c }}</th>
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
          <td>{{ a.target_name }}</td>
          <td>
            <div class="row-actions">
              <button class="btn sm ghost danger" type="button" title="Remover atribuição" aria-label="Remover atribuição" @click="remove(a)">
                <AppIcon name="trash" :size="12" />
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <ScopeAssignmentForm v-if="adding" :targets="targets" :target-label="targetLabel" :save="save" @close="adding = false" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import AppIcon from '@/components/AppIcon.vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import ScopeAssignmentForm from '@/components/ScopeAssignmentForm.vue'
import type { Assignment, AssignmentApi, AssignmentInput } from '@/api/assignments'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'
import { scopeLabel } from '@/domain'

const props = defineProps<{
  api: AssignmentApi
  assignments: Assignment[]
  loading: boolean
  targets: { id: number; name: string }[]
  targetLabel: string
  emptySub: string
}>()
// O pai recarrega a lista (e o que mais depender dela) depois de salvar ou remover.
const emit = defineEmits<{ changed: [] }>()

const toast = useToastStore()
const adding = ref(false)
const columns = computed(() => ['Escopo', 'Cliente', 'Projeto', props.targetLabel, ''])

async function save(data: AssignmentInput) {
  await props.api.set(data)
  toast.push('Atribuição salva', 'success')
  adding.value = false
  emit('changed')
}

async function remove(a: Assignment) {
  try {
    await props.api.remove(a)
    toast.push('Atribuição removida', 'success')
    emit('changed')
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao remover'), 'error')
  }
}
</script>

<style scoped>
.col-scope {
  width: 110px;
}
</style>
