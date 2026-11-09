import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (to, _de, salva) => salva || (to.hash ? { el: to.hash } : undefined),
  routes: [
    {
      path: '/',
      name: 'landing',
      component: () => import('@/views/LandingView.vue'),
      meta: { public: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: 'Entrar' },
    },
    {
      path: '/privacidade',
      name: 'privacidade',
      component: () => import('@/views/PrivacyView.vue'),
      meta: { public: true, title: 'Política de privacidade' },
    },
    // A landing precisa vir primeiro; o pai do layout não pode voltar a ter filho vazio em '/'.
    {
      path: '/',
      component: () => import('@/components/AppLayout.vue'),
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: 'Painel' },
        },
        {
          path: 'events',
          name: 'events',
          component: () => import('@/views/EventsView.vue'),
          meta: { title: 'Revisões' },
        },
        {
          path: 'clients',
          name: 'clients',
          component: () => import('@/views/ClientsView.vue'),
          meta: { roles: ['super', 'gestor'], title: 'Clientes' },
        },
        {
          path: 'prompts',
          name: 'prompts',
          component: () => import('@/views/PromptsView.vue'),
          meta: { roles: ['super', 'gestor'], title: 'Prompts' },
        },
        {
          path: 'ai-configs',
          name: 'ai-configs',
          component: () => import('@/views/AiConfigView.vue'),
          meta: { roles: ['super', 'gestor'], title: 'Modelos' },
        },
        {
          path: 'review-profiles',
          name: 'review-profiles',
          component: () => import('@/views/ReviewProfilesView.vue'),
          meta: { roles: ['super', 'gestor'], title: 'Perfis de revisão' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UsersView.vue'),
          meta: { roles: ['super'], title: 'Usuários' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.name === 'landing' && auth.isAuthenticated) return { name: 'dashboard' }
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  const roles = to.meta.roles as string[] | undefined
  if (roles) {
    await auth.initPromise
    if (auth.user && !roles.includes(auth.user.role)) {
      return { name: 'dashboard' }
    }
  }
})

// Título por página (WCAG 2.4.2): leitor de tela e aba dizem onde se está.
router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · Mergeye` : 'Mergeye'
})

export default router
