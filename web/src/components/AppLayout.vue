<template>
  <div class="layout">
    <!-- Mobile overlay -->
    <div v-if="menuOpen" class="overlay-bg" @click="menuOpen = false" />

    <aside :class="['sidebar', { open: menuOpen }]">
      <div class="brand">
        <span>Code Reviewer</span>
        <button class="close-btn" @click="menuOpen = false">✕</button>
      </div>
      <nav @click="menuOpen = false">
        <router-link to="/dashboard">
          <span class="icon">▦</span> Dashboard
        </router-link>
        <router-link to="/events">
          <span class="icon">⚡</span> Events
        </router-link>
        <router-link v-if="auth.canManageResources" to="/clients">
          <span class="icon">⚙</span> Clients
        </router-link>
        <router-link v-if="auth.canManageResources" to="/prompts">
          <span class="icon">✎</span> Prompts
        </router-link>
        <router-link v-if="auth.canManageUsers" to="/users">
          <span class="icon">👤</span> Users
        </router-link>
      </nav>
      <button class="logout" @click="handleLogout">Logout</button>
    </aside>

    <div class="body">
      <header class="topbar">
        <button class="menu-btn" @click="menuOpen = true">☰</button>
        <span class="topbar-title">Code Reviewer</span>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>

    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import ToastContainer from '@/components/ToastContainer.vue'

const auth = useAuthStore()
const router = useRouter()
const menuOpen = ref(false)

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout { display: flex; min-height: 100vh; }

/* ── Sidebar ─────────────────────────────────────────────────────────────── */
.sidebar {
  width: 220px;
  background: #1e293b;
  border-right: 1px solid #334155;
  display: flex;
  flex-direction: column;
  padding: 1.25rem 1rem;
  gap: 1rem;
  position: fixed;
  top: 0; bottom: 0; left: 0;
  z-index: 200;
  transition: transform .25s ease;
}

.brand {
  display: flex; align-items: center; justify-content: space-between;
  font-size: .95rem; font-weight: 700; color: #6366f1;
  padding: 0 .5rem .75rem;
  border-bottom: 1px solid #334155;
}
.close-btn {
  display: none;
  background: none; border: none; color: #64748b; cursor: pointer; font-size: 1rem;
}

nav { display: flex; flex-direction: column; gap: .25rem; flex: 1; }
nav a {
  display: flex; align-items: center; gap: .5rem;
  padding: .55rem .75rem;
  border-radius: 6px;
  color: #94a3b8;
  text-decoration: none;
  font-size: .9rem;
}
nav a:hover, nav a.router-link-active { background: #334155; color: #e2e8f0; }
.icon { font-size: .85rem; width: 16px; text-align: center; }

.logout {
  background: none; border: 1px solid #475569; color: #94a3b8;
  padding: .5rem; border-radius: 6px; cursor: pointer; font-size: .85rem;
}
.logout:hover { background: #334155; }

/* ── Main body ───────────────────────────────────────────────────────────── */
.body { margin-left: 220px; flex: 1; display: flex; flex-direction: column; min-width: 0; }

.topbar {
  display: none;
  align-items: center; gap: .75rem;
  padding: .75rem 1rem;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  position: sticky; top: 0; z-index: 50;
}
.menu-btn { background: none; border: none; color: #e2e8f0; font-size: 1.2rem; cursor: pointer; }
.topbar-title { font-size: .95rem; font-weight: 700; color: #6366f1; }

.content { padding: 2rem; flex: 1; }

.overlay-bg {
  display: none;
  position: fixed; inset: 0; background: rgba(0,0,0,.5); z-index: 190;
}

/* ── Responsive ──────────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
  }
  .sidebar.open {
    transform: translateX(0);
  }
  .close-btn { display: block; }
  .overlay-bg { display: block; }
  .body { margin-left: 0; }
  .topbar { display: flex; }
  .content { padding: 1rem; }
}
</style>
