import { createRouter, createWebHistory, type RouteComponent } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { NAV, type Capability, type NavPath } from '@/domain'

// Uma view por item de NAV: o tipo não compila se faltar ou sobrar uma.
const VIEWS: Record<NavPath, () => Promise<RouteComponent>> = {
  '/dashboard': () => import('@/views/DashboardView.vue'),
  '/events': () => import('@/views/EventsView.vue'),
  '/clients': () => import('@/views/ClientsView.vue'),
  '/prompts': () => import('@/views/PromptsView.vue'),
  '/ai-configs': () => import('@/views/AiConfigView.vue'),
  '/review-profiles': () => import('@/views/ReviewProfilesView.vue'),
  '/users': () => import('@/views/UsersView.vue'),
}

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
      children: NAV.map((n) => ({
        path: n.path.slice(1),
        name: n.path.slice(1),
        component: VIEWS[n.path],
        meta: { title: n.label, need: n.need },
      })),
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.path === '/' && auth.isAuthenticated) return '/dashboard'
  if (to.meta.public) return
  if (!auth.isAuthenticated) return '/login'
  const need = to.meta.need as Capability | undefined
  if (need) {
    await auth.initPromise
    if (!auth.isAuthenticated) return '/login'
    if (!auth.can(need)) return '/dashboard'
  }
})

// Título por página (WCAG 2.4.2): leitor de tela e aba dizem onde se está.
router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · Mergeye` : 'Mergeye'
})

export default router
