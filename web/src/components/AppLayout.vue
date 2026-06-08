<template>
  <div class="layout">
    <!-- Mobile overlay -->
    <div v-if="menuOpen" class="overlay-bg" @click="menuOpen = false" />

    <aside :class="['sidebar', { open: menuOpen }]">
      <!-- Brand block -->
      <div class="brand">
        <div class="brand-mark">CR</div>
        <div class="brand-info">
          <div class="brand-name">Code Reviewer</div>
          <div class="brand-sub">v 2.4 · prod</div>
        </div>
        <button class="close-btn" @click="menuOpen = false">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- Navigation sections -->
      <div class="nav-section">
        <div class="nav-label">Workspace</div>
        <router-link to="/dashboard" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="7" height="9" rx="1.5"/>
            <rect x="14" y="3" width="7" height="5" rx="1.5"/>
            <rect x="14" y="12" width="7" height="9" rx="1.5"/>
            <rect x="3" y="16" width="7" height="5" rx="1.5"/>
          </svg>
          <span>Dashboard</span>
          <span class="nav-kbd">D</span>
        </router-link>
        <router-link to="/events" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 12h-3.5l-2.5 7-5-16-2.5 9H2"/>
          </svg>
          <span>Events</span>
          <span class="nav-kbd">E</span>
        </router-link>
        <router-link v-if="auth.canManageResources" to="/clients" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="3" width="16" height="18" rx="1"/>
            <path d="M9 8h.01M15 8h.01M9 12h.01M15 12h.01M9 16h6"/>
          </svg>
          <span>Clients</span>
          <span class="nav-kbd">C</span>
        </router-link>
      </div>

      <div class="nav-section">
        <div class="nav-label">Configuração</div>
        <router-link v-if="auth.canManageResources" to="/prompts" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
            <path d="M14 2v6h6"/>
            <path d="M8 13h8M8 17h6"/>
          </svg>
          <span>Prompts</span>
          <span class="nav-kbd">P</span>
        </router-link>
        <router-link v-if="auth.canManageResources" to="/ai-configs" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="4" width="16" height="16" rx="2"/>
            <rect x="9" y="9" width="6" height="6"/>
            <path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>
          </svg>
          <span>Modelos</span>
          <span class="nav-kbd">M</span>
        </router-link>
      </div>

      <div class="nav-section">
        <div class="nav-label">Acesso</div>
        <router-link v-if="auth.canManageUsers" to="/users" class="nav-item" @click="menuOpen = false">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M17 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2"/>
            <circle cx="10" cy="7" r="3.5"/>
            <path d="M21 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
          <span>Users</span>
          <span class="nav-kbd">U</span>
        </router-link>
      </div>

      <!-- Sidebar footer -->
      <div class="sidebar-foot">
        <div class="avatar">{{ userInitials }}</div>
        <div class="user-info">
          <div class="user-name">{{ auth.user?.username || 'admin' }}</div>
          <div class="user-role">{{ auth.user?.role || 'super' }} · code-reviewer</div>
        </div>
        <button class="icon-btn" title="Trocar tema" @click="toggleTheme">
          <svg v-if="theme === 'dark'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="4"/>
            <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>
          </svg>
        </button>
        <button class="icon-btn" title="Sair" @click="handleLogout">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <path d="M16 17l5-5-5-5M21 12H9"/>
          </svg>
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <div class="body">
      <header class="topbar">
        <button class="menu-btn" @click="menuOpen = true">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <line x1="3" y1="6" x2="21" y2="6"/>
            <line x1="3" y1="12" x2="21" y2="12"/>
            <line x1="3" y1="18" x2="21" y2="18"/>
          </svg>
        </button>
        <span class="topbar-brand">Code Reviewer</span>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>

    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import ToastContainer from '@/components/ToastContainer.vue'

const auth = useAuthStore()
const router = useRouter()
const menuOpen = ref(false)
const theme = ref<'dark' | 'light'>('dark')

const userInitials = computed(() => {
  const name = auth.user?.username || 'admin'
  return name.slice(0, 2).toUpperCase()
})

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', theme.value)
  localStorage.setItem('theme', theme.value)
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

// Keyboard navigation
const NAV_KEYS: Record<string, string> = {
  d: '/dashboard',
  e: '/events',
  c: '/clients',
  p: '/prompts',
  m: '/ai-configs',
  u: '/users',
}

function handleKeydown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLSelectElement || e.target instanceof HTMLTextAreaElement) return
  if (e.metaKey || e.ctrlKey || e.altKey) return

  const path = NAV_KEYS[e.key.toLowerCase()]
  if (path) {
    router.push(path)
    e.preventDefault()
  }
}

onMounted(() => {
  const saved = localStorage.getItem('theme') as 'dark' | 'light' | null
  if (saved) {
    theme.value = saved
    document.documentElement.setAttribute('data-theme', saved)
  }
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
}

/* ── Sidebar ──────────────────────────────── */
.sidebar {
  width: 232px;
  border-right: 1px solid var(--hairline);
  background: var(--bg);
  display: flex;
  flex-direction: column;
  padding: 20px 14px;
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 200;
  transition: transform 0.25s ease;
}

/* Brand block */
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px 18px 8px;
  border-bottom: 1px solid var(--hairline);
  margin-bottom: 14px;
  min-width: 0;
}

.brand-mark {
  width: 24px;
  height: 24px;
  border: 1px solid var(--hairline-strong);
  border-radius: 5px;
  display: grid;
  place-items: center;
  font-family: var(--font-mono);
  font-weight: 600;
  font-size: 11px;
  color: var(--text);
  background: linear-gradient(180deg, var(--panel) 0%, var(--bg-2) 100%);
  flex-shrink: 0;
}

.brand-info {
  flex: 1;
  min-width: 0;
}

.brand-name {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.brand-sub {
  font-size: 10.5px;
  color: var(--muted);
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.close-btn {
  display: none;
  background: none;
  border: none;
  color: var(--muted);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
}

.close-btn:hover {
  color: var(--text);
  background: var(--panel);
}

/* Navigation sections */
.nav-section {
  margin-top: 4px;
}

.nav-label {
  font-size: 10.5px;
  color: var(--muted-2);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  padding: 10px 8px 6px 8px;
  font-family: var(--font-mono);
  font-weight: 500;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 8px;
  border-radius: 5px;
  color: var(--text-2);
  text-decoration: none;
  font-size: 13px;
  line-height: 1;
  transition: background 0.12s, color 0.12s;
}

.nav-item:hover {
  background: var(--panel);
  color: var(--text);
}

.nav-item.router-link-active {
  background: var(--panel);
  color: var(--text);
  box-shadow: var(--ring);
}

.nav-item svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--muted);
}

.nav-item.router-link-active svg {
  color: var(--text);
}

/* Sidebar footer */
.sidebar-foot {
  margin-top: auto;
  padding-top: 14px;
  border-top: 1px solid var(--hairline);
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  font-size: 11px;
  color: var(--muted);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Main body ─────────────────────────────── */
.body {
  margin-left: 232px;
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
  background: rgba(8, 9, 11, 0.85);
  backdrop-filter: blur(8px);
}

[data-theme="light"] .topbar {
  background: rgba(247, 246, 243, 0.85);
}

.menu-btn {
  background: none;
  border: none;
  color: var(--text);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
}

.menu-btn:hover {
  background: var(--panel);
}

.topbar-brand {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
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

/* ── Responsive ────────────────────────────── */
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