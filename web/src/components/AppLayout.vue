<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">Code Reviewer</div>
      <nav>
        <router-link to="/dashboard">Dashboard</router-link>
        <router-link to="/events">Events</router-link>
        <router-link to="/clients">Clients</router-link>
      </nav>
      <button class="logout" @click="handleLogout">Logout</button>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout { display: flex; min-height: 100vh; }
.sidebar {
  width: 220px;
  background: #1e293b;
  border-right: 1px solid #334155;
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1rem;
  gap: 1rem;
  position: fixed;
  top: 0; bottom: 0; left: 0;
}
.brand { font-size: 1rem; font-weight: 700; color: #6366f1; padding: 0 .5rem .5rem; border-bottom: 1px solid #334155; }
nav { display: flex; flex-direction: column; gap: .25rem; flex: 1; }
nav a {
  display: block;
  padding: .5rem .75rem;
  border-radius: 6px;
  color: #94a3b8;
  text-decoration: none;
  font-size: .9rem;
}
nav a:hover, nav a.router-link-active { background: #334155; color: #e2e8f0; }
.logout {
  background: none;
  border: 1px solid #475569;
  color: #94a3b8;
  padding: .5rem;
  border-radius: 6px;
  cursor: pointer;
  font-size: .85rem;
}
.content { margin-left: 220px; flex: 1; padding: 2rem; }
</style>
