import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false, layout: 'bare' },
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'home',
        component: () => import('@/views/HomeView.vue'),
      },
      {
        path: 'notes/:path(.*)/edit',
        name: 'note-edit',
        component: () => import('@/views/NoteEditView.vue'),
        meta: { requiresWrite: true },
      },
      {
        path: 'notes/:path(.*)',
        name: 'note',
        component: () => import('@/views/NoteView.vue'),
      },
      {
        path: 'search',
        name: 'search',
        component: () => import('@/views/SearchView.vue'),
      },
      {
        path: 'projects',
        name: 'projects',
        component: () => import('@/views/ProjectsView.vue'),
      },
      {
        path: 'tags',
        name: 'tags',
        component: () => import('@/views/TagsView.vue'),
      },
      {
        path: 'tags/:tag',
        name: 'tag-detail',
        component: () => import('@/views/TagsView.vue'),
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
      },
      {
        path: 'trash',
        name: 'trash',
        component: () => import('@/views/TrashView.vue'),
      },
      {
        path: 'graph',
        name: 'graph',
        component: () => import('@/views/GraphView.vue'),
      },
      {
        path: 'notes/:path(.*)/history',
        name: 'note-history',
        component: () => import('@/views/NoteHistoryView.vue'),
      },
      {
        path: 'admin',
        component: () => import('@/views/admin/AdminLayout.vue'),
        meta: { requiresOwner: true },
        children: [
          {
            path: '',
            redirect: '/admin/users',
          },
          {
            path: 'users',
            name: 'admin-users',
            component: () => import('@/views/admin/AdminUsersView.vue'),
          },
          {
            path: 'tokens',
            name: 'admin-tokens',
            component: () => import('@/views/admin/AdminTokensView.vue'),
          },
          {
            path: 'spa-tokens',
            name: 'admin-spa-tokens',
            component: () => import('@/views/admin/AdminSpaTokensView.vue'),
          },
          {
            path: 'invites',
            name: 'admin-invites',
            component: () => import('@/views/admin/AdminInvitesView.vue'),
          },
          {
            path: 'audit',
            name: 'admin-audit',
            component: () => import('@/views/admin/AdminAuditView.vue'),
          },
        ],
      },
      // Phase 3.4+ add /graph here.
      {
        path: ':pathMatch(.*)*',
        name: 'not-found',
        component: () => import('@/views/PlaceholderView.vue'),
      },
    ],
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from) => {
  const auth = useAuthStore()
  const requires = to.matched.some((r) => r.meta.requiresAuth !== false)
  const ownerOnly = to.matched.some((r) => r.meta.requiresOwner)
  const writeOnly = to.matched.some((r) => r.meta.requiresWrite)

  if (requires && !auth.isAuthenticated) {
    return {
      name: 'login',
      query: to.fullPath !== '/' ? { next: to.fullPath } : {},
    }
  }
  if (ownerOnly && !auth.isOwner) {
    return { name: 'home' }
  }
  if (writeOnly && !auth.canWrite) {
    // Guests are read-only — keep them out of the editor.
    return { name: 'home' }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'home' }
  }
  return true
})
