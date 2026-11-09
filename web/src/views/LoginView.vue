<template>
  <div class="login-wrap">
    <BackgroundMotion :intensity="1.6" />
    <div class="login-card">
      <div class="brand">
        <BrandLogo variant="horizontal" class="brand-logo" />
      </div>

      <form @submit.prevent="handleLogin">
        <h1>Entrar</h1>
        <p v-if="error" class="error" role="alert">{{ error }}</p>

        <div class="field">
          <label for="login-username">Usuário</label>
          <input
            id="login-username"
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="admin"
            required
          />
        </div>

        <div class="field">
          <label for="login-password">Senha</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            required
          />
        </div>

        <button type="submit" :disabled="loading" class="btn primary">
          <span v-if="loading" class="spinner" />
          {{ loading ? 'Entrando…' : 'Entrar' }}
        </button>
      </form>

      <p class="aviso">
        Usamos seu usuário para controlar o acesso ao painel.
        <router-link to="/privacidade">Política de privacidade</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import BrandLogo from '@/components/BrandLogo.vue'
import BackgroundMotion from '@/components/BackgroundMotion.vue'

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
    error.value = 'Credenciais inválidas'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 20px;
}

.login-card {
  position: relative;
  z-index: 1;
  background: var(--panel-solid);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  padding: 2rem;
  width: 100%;
  max-width: 380px;
}

.brand {
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--hairline);
  margin-bottom: 1.5rem;
}

.brand-logo {
  width: 180px;
  margin: 0 auto;
}

h1 {
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text);
  margin-bottom: 0.5rem;
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
  font-size: 13px;
  color: var(--muted);
}

.field input {
  padding: 10px 12px;
  background: var(--bg-2);
  border: 1px solid var(--field-border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: var(--font-sans);
  transition: border-color 0.12s;
}

.field input:focus {
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

.aviso {
  margin-top: 1.25rem;
  font-size: 12.5px;
  color: var(--muted);
}

.aviso a {
  text-decoration: underline;
}

.error {
  color: var(--danger);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--bg-2);
  border: 1px solid var(--danger);
  border-radius: var(--radius);
  overflow-wrap: anywhere;
}
</style>
