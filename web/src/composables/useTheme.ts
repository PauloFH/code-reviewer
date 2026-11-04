import { ref } from 'vue'

export type Theme = 'dark' | 'light'

// Um único ref compartilhado; o script inline de index.html já aplicou o atributo antes do Vue montar.
const theme = ref<Theme>(
  document.documentElement.getAttribute('data-theme') === 'light' ? 'light' : 'dark',
)

function toggleTheme(): void {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
  if (theme.value === 'light') document.documentElement.setAttribute('data-theme', 'light')
  else document.documentElement.removeAttribute('data-theme')
  try {
    localStorage.setItem('theme', theme.value)
  } catch {
    // armazenamento bloqueado: o tema vale só nesta sessão
  }
}

export function useTheme() {
  return { theme, toggleTheme }
}
