<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · perfis de revisão</div>
        <h1 class="page-title">Perfis de revisão e atribuições</h1>
        <div class="page-sub">Defina os perfis de revisão e a quais escopos eles se aplicam — global, por cliente ou por projeto.</div>
      </div>
    </div>

    <div class="stack">
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Perfis</h2>
            <div class="card-sub">{{ profiles.length }} {{ profiles.length === 1 ? 'definido' : 'definidos' }}</div>
          </div>
          <button class="btn primary" type="button" @click="openForm(null)">
            <AppIcon name="plus" :size="13" />
            Novo perfil
          </button>
        </div>

        <SkeletonRows v-if="loading && !profiles.length" :cols="COLUMNS.length" />
        <div v-else-if="profiles.length === 0" class="empty-state">
          <div class="empty-title">Nenhum perfil cadastrado</div>
          <div class="empty-sub">Crie um perfil para configurar as camadas de revisão do agente.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th v-for="c in COLUMNS" :key="c">{{ c }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in profiles" :key="p.id">
              <td>
                <div class="name-cell">
                  <div class="row-icon">
                    <AppIcon name="layers" :size="14" />
                  </div>
                  <div>
                    <div class="name">{{ p.name }}</div>
                    <div class="card-sub">criado em {{ formatDate(p.created_at) }}</div>
                  </div>
                </div>
              </td>
              <td>
                <div class="badges">
                  <span v-if="p.include_related" class="badge related">relacionado</span>
                  <span v-if="p.agentic" class="badge agentic">agêntico</span>
                  <span v-if="p.execution" class="badge execution">sandbox</span>
                  <span v-if="!p.include_related && !p.agentic && !p.execution" class="badge none">básico</span>
                </div>
              </td>
              <td class="num">{{ p.max_iterations || '—' }}</td>
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
        :api="reviewProfilesApi.assignments"
        :assignments="assignments"
        :loading="loading"
        :targets="profiles"
        target-label="Perfil"
        empty-sub="O sistema está usando a configuração padrão do ambiente. Crie uma atribuição para sobrepor o fallback em clientes ou projetos específicos."
        @changed="load"
      />
    </div>

    <ReviewProfileForm v-if="showForm" :initial="editing" :save="save" @close="showForm = false" />

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
import SkeletonRows from '@/components/SkeletonRows.vue'
import AppIcon from '@/components/AppIcon.vue'
import AssignmentsCard from '@/components/AssignmentsCard.vue'
import ReviewProfileForm from '@/components/ReviewProfileForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { reviewProfilesApi } from '@/api/reviewProfiles'
import { useLibrary } from '@/composables/useLibrary'
import { formatDate } from '@/brand/format'

const COLUMNS = ['Nome', 'Camadas', 'Máx. iterações', '']

const { items: profiles, assignments, loading, deleting, showForm, editing, deleteTarget, load, openForm, save, doDelete } =
  useLibrary(reviewProfilesApi, {
    loadError: 'Erro ao carregar perfis',
    created: 'Perfil criado',
    updated: 'Perfil atualizado',
    deleted: (n) => `Perfil "${n}" excluído`,
  })
</script>

<style scoped>
.badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.badge {
  font-size: 11.5px;
  font-family: var(--font-mono);
  padding: 2px 7px;
  border-radius: var(--radius);
  border: 1px solid var(--hairline-strong);
  font-weight: 500;
  color: var(--text-2);
  background: var(--bg-2);
}

.badge.agentic {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}

/* Camada mais sensível (executa comandos): marca-texto pontual, sempre com --ink. */
.badge.execution {
  color: var(--ink);
  border-color: var(--hl-2);
  background: var(--highlight);
}

.badge.none {
  color: var(--muted);
  border-color: var(--hairline);
}
</style>
