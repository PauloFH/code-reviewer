<script setup lang="ts">
export type IconName =
  | 'plus' | 'trash' | 'edit' | 'close' | 'refresh' | 'activity' | 'document' | 'layers'
  | 'check' | 'play' | 'settings' | 'merge' | 'external' | 'search' | 'calendar'
  | 'chevron-left' | 'chevron-right' | 'sun' | 'moon' | 'logout' | 'menu' | 'keyboard'

withDefaults(defineProps<{ name: IconName; size?: number }>(), { size: 14 })

// Traço reto de 2px em grade 24x24; um único mapa para todos os ícones do painel.
const PATHS: Record<IconName, string> = {
  plus: 'M12 4v16M4 12h16',
  trash: 'M4 7h16M9 7V4h6v3M6 7v13h12V7M10 11v6M14 11v6',
  edit: 'M4 20v-4L16 4l4 4L8 20zM13 7l4 4',
  close: 'M5 5l14 14M19 5L5 19',
  refresh: 'M20 4v6h-6M20 10a8 8 0 1 0 1 6',
  activity: 'M2 12h5l3-8 4 16 3-8h5',
  document: 'M6 3h8l4 4v14H6zM14 3v4h4M9 12h6M9 16h6',
  layers: 'M12 3l9 5-9 5-9-5zM3 12l9 5 9-5M3 16l9 5 9-5',
  check: 'M4 12l5 5L20 6',
  play: 'M7 4l13 8-13 8z',
  settings: 'M4 6h8M16 6h4M4 12h2M10 12h10M4 18h10M18 18h2M12 4h4v4h-4zM6 10h4v4H6zM14 16h4v4h-4z',
  merge: 'M4 3h4v4H4zM16 3h4v4h-4zM4 17h4v4H4zM6 7v10M18 7v3L6 14',
  external: 'M14 4h6v6M20 4L10 14M18 14v6H4V6h6',
  search: 'M16 10a6 6 0 1 1-12 0 6 6 0 0 1 12 0zM15 15l6 6',
  calendar: 'M4 5h16v16H4zM4 10h16M8 3v4M16 3v4',
  'chevron-left': 'M15 5l-7 7 7 7',
  'chevron-right': 'M9 5l7 7-7 7',
  sun: 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0zM12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M5 19l2-2M17 7l2-2',
  moon: 'M20 14A8 8 0 0 1 10 4a8 8 0 1 0 10 10z',
  logout: 'M10 4H4v16h6M9 12h11M16 8l4 4-4 4',
  menu: 'M3 6h18M3 12h18M3 18h18',
  keyboard: 'M2 6h20v12H2zM6 10h2M11 10h2M16 10h2M7 14h10',
}
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="square"
    stroke-linejoin="miter"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="PATHS[name]" />
  </svg>
</template>
