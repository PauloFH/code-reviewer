<template>
  <div class="layout">
    <BackgroundMotion />

    <div v-if="menuOpen" class="overlay-bg" @click="menuOpen = false" />

    <aside :class="['sidebar', { open: menuOpen }]">
      <div class="brand">
        <BrandLogo variant="horizontal" class="brand-logo" />
        <button class="close-btn" aria-label="Fechar menu" @click="menuOpen = false">
          <AppIcon name="close" :size="14" />
        </button>
      </div>

      <nav class="nav" aria-label="Principal">
        <div v-for="sec in sections" :key="sec.name" class="nav-section">
          <div class="nav-label">{{ sec.name }}</div>
          <router-link v-for="n in sec.items" :key="n.path" :to="n.path" class="nav-item" @click="menuOpen = false">
            <span>{{ n.label }}</span><kbd v-if="atalhos" class="nav-kbd">{{ n.key.toUpperCase() }}</kbd>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-foot">
        <div class="user-row">
          <div class="user-info">
            <template v-if="auth.user">
              <div class="user-name">{{ auth.user.username }}</div>
              <div class="user-role">{{ roleLabel(auth.user.role) }}</div>
            </template>
          </div>
          <button class="icon-btn" title="Sair" aria-label="Sair" @click="handleLogout">
            <AppIcon name="logout" :size="14" />
          </button>
        </div>
        <button class="theme-btn" type="button" @click="toggleTheme">
          <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" :size="14" />
          {{ theme === 'dark' ? 'Tema claro' : 'Tema escuro' }}
        </button>
        <button class="theme-btn" type="button" :aria-pressed="atalhos" @click="alternarAtalhos">
          <AppIcon name="keyboard" :size="14" />
          {{ atalhos ? 'Atalhos ligados' : 'Atalhos desligados' }}
        </button>
        <router-link to="/privacidade" class="privacidade">Política de privacidade</router-link>
      </div>
    </aside>

    <div class="body">
      <header class="topbar">
        <button class="menu-btn" aria-label="Abrir menu" @click="menuOpen = true">
          <AppIcon name="menu" :size="20" />
        </button>
        <BrandLogo variant="horizontal" class="topbar-logo" />
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import { isEditableTarget } from '@/brand/shortcuts'
import { NAV, roleLabel } from '@/domain'
import { readStorage, writeStorage } from '@/brand/storage'
import AppIcon from '@/components/AppIcon.vue'
import BackgroundMotion from '@/components/BackgroundMotion.vue'
import BrandLogo from '@/components/BrandLogo.vue'

const auth = useAuthStore()
const router = useRouter()
const { theme, toggleTheme } = useTheme()
const menuOpen = ref(false)

const allowed = computed(() => NAV.filter((n) => auth.can(n.need)))
// Seções na ordem em que aparecem pela primeira vez em NAV.
const sections = computed(() => {
  const bySection = new Map<string, (typeof NAV)[number][]>()
  for (const n of allowed.value) bySection.set(n.section, [...(bySection.get(n.section) ?? []), n])
  return [...bySection].map(([name, items]) => ({ name, items }))
})
// Atalhos de uma tecla podem ser desligados (WCAG 2.1.4); a escolha persiste.
const atalhos = ref(readStorage('shortcuts') !== 'off')

function alternarAtalhos() {
  atalhos.value = !atalhos.value
  writeStorage('shortcuts', atalhos.value ? 'on' : 'off')
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && menuOpen.value) {
    menuOpen.value = false
    return
  }
  if (!atalhos.value || document.querySelector('[aria-modal="true"]')) return
  if (isEditableTarget(e.target as HTMLElement | null)) return
  if (e.metaKey || e.ctrlKey || e.altKey) return

  const item = allowed.value.find((n) => n.key === e.key.toLowerCase())
  if (item) {
    router.push(item.path)
    e.preventDefault()
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
}

.sidebar {
  width: 236px;
  border-right: 1px solid var(--hairline);
  background: var(--bg-2);
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px 14px;
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 200;
  transition: transform 0.25s ease;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 6px;
  min-width: 0;
}

.brand-logo {
  width: 190px;
  min-width: 0; /* no drawer, encolhe só o necessário para o botão de fechar caber na barra */
  max-width: 100%;
}

.close-btn {
  display: none;
  background: none;
  border: none;
  color: var(--muted);
  cursor: pointer;
  padding: 4px;
  flex-shrink: 0;
  margin-left: auto;
}

.close-btn:hover {
  color: var(--text);
  background: var(--panel);
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  min-height: 0;
  padding-bottom: 2px; /* o item ativo girado (-0,5deg) excede ~0,86px e geraria rolagem */
  scrollbar-width: thin;
  scrollbar-color: var(--hairline-strong) transparent;
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-label {
  font-size: 12px;
  line-height: 1.3;
  color: var(--muted);
  padding: 0 8px 2px;
}

.nav-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  padding: 4px 8px;
  line-height: 1.3;
  border-radius: 2px;
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
  font-size: 15px;
  transition: background 0.2s;
}

.nav-item:hover {
  background: color-mix(in srgb, var(--text) 8%, transparent);
}

.nav-item.router-link-active {
  background: var(--highlight);
  color: var(--ink);
  font-weight: 800;
  transform: rotate(-0.5deg);
}

.nav-kbd {
  font: 11px var(--font-mono);
  color: var(--muted);
}

.nav-item.router-link-active .nav-kbd {
  color: var(--ink);
}

.sidebar-foot {
  margin-top: auto;
  padding-top: 14px;
  border-top: 1px solid var(--hairline);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.user-row {
  display: flex;
  align-items: center;
  gap: 9px;
}

.user-info {
  flex: 1;
  min-width: 0;
  font-size: 13px;
}

.user-name {
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  font-size: 12px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.theme-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 6px 9px;
  border: 1px solid var(--hairline);
  border-radius: 2px;
  background: none;
  color: var(--text);
  font: inherit;
  font-size: 12.5px;
  cursor: pointer;
}

.theme-btn:hover {
  border-color: var(--text);
}

.privacidade {
  font-size: 12px;
  color: var(--muted);
  text-align: center;
}

.body {
  position: relative;
  z-index: 1;
  margin-left: 236px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  display: none;
  align-items: center;
  gap: 0.75rem;
  padding: 14px 20px;
  border-bottom: 1px solid var(--hairline);
  position: sticky;
  top: 0;
  z-index: 50;
  background: color-mix(in srgb, var(--bg) 88%, transparent);
  backdrop-filter: blur(8px);
}

.topbar-logo {
  width: 140px;
}

.menu-btn {
  background: none;
  border: none;
  color: var(--text);
  cursor: pointer;
  padding: 4px;
}

.menu-btn:hover {
  background: var(--panel);
}

.content {
  padding: 32px 32px 56px;
  flex: 1;
  max-width: 1320px;
}

.overlay-bg {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  z-index: 190;
}

@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
  }
  .sidebar.open {
    transform: translateX(0);
  }
  .close-btn {
    display: grid;
  }
  .overlay-bg {
    display: block;
  }
  .body {
    margin-left: 0;
  }
  .topbar {
    display: flex;
  }
  .content {
    padding: 20px 16px 40px;
  }
}
</style>
