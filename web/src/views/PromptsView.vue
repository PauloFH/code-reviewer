<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · prompts</div>
        <h1 class="page-title">Prompts e atribuições</h1>
        <div class="page-sub">Defina os prompts usados pelo agente e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Prompts</h2>
            <div class="card-sub">{{ prompts.length }} {{ prompts.length === 1 ? 'definido' : 'definidos' }}</div>
          </div>
          <button class="btn primary" type="button" @click="openForm(null)">
            <AppIcon name="plus" :size="13" />
            Novo prompt
          </button>
        </div>

        <SkeletonRows v-if="loading && !prompts.length" :cols="COLUMNS.length" />
        <div v-else-if="prompts.length === 0" class="empty-state">
          <div class="empty-title">Nenhum prompt cadastrado</div>
          <div class="empty-sub">Crie um prompt para customizar como o agente analisa o código.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th v-for="c in COLUMNS" :key="c">{{ c }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in prompts" :key="p.id">
              <td>
                <div class="name-cell">
                  <div class="row-icon">
                    <AppIcon name="document" :size="14" />
                  </div>
                  <div class="name">{{ p.name }}</div>
                </div>
              </td>
              <td class="num">{{ formatDate(p.created_at) }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn sm ghost" type="button" title="Editar" :aria-label="`Editar ${p.name}`" @click="openForm(p)">
                    <AppIcon name="edit" :size="12" />
                  </button>
                  <button class="btn sm ghost danger" type="button" title="Excluir" :aria-label="`Excluir ${p.name}`" @click="deleteTarget = p">
                    <AppIcon name="trash" :size="12" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AssignmentsCard
        :api="promptsApi.assignments"
        :assignments="assignments"
        :loading="loading"
        :targets="prompts"
        target-label="Prompt"
        empty-sub="O sistema está usando a configuração padrão do ambiente. Crie uma atribuição para sobrepor o fallback em clientes ou projetos específicos."
        @changed="load"
      />
    </div>

    <PromptForm v-if="showForm" :initial="editing" :save="save" @close="showForm = false" />

    <ConfirmModal
      v-if="deleteTarget"
      :name="deleteTarget.name"
      warning="Atribuições vinculadas serão removidas."
      :loading="deleting"
      @close="deleteTarget = null"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import AppIcon from '@/components/AppIcon.vue'
import AssignmentsCard from '@/components/AssignmentsCard.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { promptsApi } from '@/api/prompts'
import { useLibrary } from '@/composables/useLibrary'
import { formatDate } from '@/brand/format'

// O editor (Tiptap) só é baixado quando o modal abre, não com a página.
// Após um deploy o chunk antigo some: recarregar busca o index.html novo, que aponta para o chunk atual.
const PromptForm = defineAsyncComponent({
  loader: () => import('@/components/PromptForm.vue'),
  onError: () => location.reload(),
})

const COLUMNS = ['Nome', 'Criado em', '']

const { items: prompts, assignments, loading, deleting, showForm, editing, deleteTarget, load, openForm, save, doDelete } =
  useLibrary(promptsApi, {
    loadError: 'Erro ao carregar prompts',
    created: 'Prompt criado',
    updated: 'Prompt atualizado',
    deleted: (n) => `Prompt "${n}" excluído`,
  })
</script>
