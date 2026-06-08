<template>
  <div class="login-wrap">
    <div class="login-card">
      <!-- Brand -->
      <div class="brand">
        <div class="brand-mark">CR</div>
        <div class="brand-info">
          <div class="brand-name">Code Reviewer</div>
          <div class="brand-sub">v 2.4</div>
        </div>
      </div>

      <form @submit.prevent="handleLogin">
        <h1>Sign in</h1>
        <p v-if="error" class="error">{{ error }}</p>

        <div class="field">
          <label>Username</label>
          <input
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="admin"
            required
          />
        </div>

        <div class="field">
          <label>Password</label>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            required
          />
        </div>

        <button type="submit" :disabled="loading" class="btn primary">
          <span v-if="loading" class="spinner" />
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    router.push('/dashboard')
  } catch {
    error.value = 'Invalid credentials'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 20px;
}

.login-card {
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  padding: 2rem;
  width: 100%;
  max-width: 380px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--hairline);
  margin-bottom: 1.5rem;
}

.brand-mark {
  width: 32px;
  height: 32px;
  border: 1px solid var(--hairline-strong);
  border-radius: 6px;
  display: grid;
  place-items: center;
  font-family: var(--font-mono);
  font-weight: 600;
  font-size: 12px;
  color: var(--text);
  background: linear-gradient(180deg, var(--panel-2) 0%, var(--bg-2) 100%);
}

.brand-info {
  flex: 1;
}

.brand-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.brand-sub {
  font-size: 11px;
  color: var(--muted);
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

h1 {
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text);
  margin-bottom: 1.5rem;
}

form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 12px;
  color: var(--muted);
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.field input {
  padding: 10px 12px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: var(--font-sans);
  transition: border-color 0.12s;
}

.field input:focus {
  outline: none;
  border-color: var(--accent);
}

.field input::placeholder {
  color: var(--muted-2);
}

.btn.primary {
  width: 100%;
  justify-content: center;
  padding: 10px;
  margin-top: 0.5rem;
}

.error {
  color: var(--danger);
  font-size: 13px;
  padding: 8px 12px;
  background: rgba(248, 113, 113, 0.08);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: var(--radius);
}
</style>