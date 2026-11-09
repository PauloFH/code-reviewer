<template>
  <div aria-busy="true">
    <span class="sr-only" role="status">Carregando…</span>
    <table class="table" aria-hidden="true">
      <tbody>
        <tr v-for="r in rows" :key="r">
          <td v-for="c in cols" :key="c">
            <span class="skeleton" :style="{ width: width(r, c) }" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ cols: number; rows?: number }>(), { rows: 4 })

// Larguras fixas por posição: variam entre linhas sem mudar a cada render.
const WIDTHS = ['70%', '45%', '60%', '35%', '55%']
const width = (r: number, c: number) => WIDTHS[(r + c * 2) % WIDTHS.length]
</script>
