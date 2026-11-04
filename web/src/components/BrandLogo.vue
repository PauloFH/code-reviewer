<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from '@/composables/useTheme'

const props = withDefaults(defineProps<{ variant?: 'horizontal' | 'simbolo' }>(), { variant: 'horizontal' })

const { theme } = useTheme()

// URLs literais para o Vite resolver em build.
const sources = {
  'horizontal-dark': new URL('../assets/brand/mergeye-logo-horizontal-escuro.png', import.meta.url).href,
  'horizontal-light': new URL('../assets/brand/mergeye-logo-horizontal-claro.png', import.meta.url).href,
  'simbolo-dark': new URL('../assets/brand/mergeye-simbolo-escuro.png', import.meta.url).href,
  'simbolo-light': new URL('../assets/brand/mergeye-simbolo-claro.png', import.meta.url).href,
}

const src = computed(() => sources[`${props.variant}-${theme.value}` as keyof typeof sources])
</script>

<template>
  <img :src="src" alt="Mergeye" class="brand-logo" />
</template>

<style scoped>
.brand-logo {
  display: block;
  max-width: 100%;
  height: auto;
}
</style>
