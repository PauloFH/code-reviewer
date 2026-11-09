<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from '@/composables/useTheme'

const props = withDefaults(defineProps<{ variant?: 'horizontal' | 'simbolo' }>(), { variant: 'horizontal' })

const { theme } = useTheme()

// URLs literais para o Vite resolver em build. Largura e altura reservam o espaço antes de a imagem chegar.
const sources = {
  'horizontal-dark': { src: new URL('../assets/brand/mergeye-logo-horizontal-escuro.webp', import.meta.url).href, w: 600, h: 109 },
  'horizontal-light': { src: new URL('../assets/brand/mergeye-logo-horizontal-claro.webp', import.meta.url).href, w: 600, h: 108 },
  'simbolo-dark': { src: new URL('../assets/brand/mergeye-simbolo-escuro.webp', import.meta.url).href, w: 600, h: 312 },
  'simbolo-light': { src: new URL('../assets/brand/mergeye-simbolo-claro.webp', import.meta.url).href, w: 600, h: 312 },
}

const logo = computed(() => sources[`${props.variant}-${theme.value}` as keyof typeof sources])
</script>

<template>
  <img :src="logo.src" :width="logo.w" :height="logo.h" alt="Mergeye" class="brand-logo" fetchpriority="high" />
</template>

<style scoped>
.brand-logo {
  display: block;
  max-width: 100%;
  height: auto;
}
</style>
