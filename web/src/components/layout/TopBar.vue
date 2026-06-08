<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { Search, LogOut } from 'lucide-vue-next'

const { t } = useI18n()

const router = useRouter()
const auth = useAuthStore()

async function handleLogout() {
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <header
    class="flex items-center justify-between px-4 py-2 border-b border-border bg-bg-elevated"
  >
    <div class="flex items-center gap-3">
      <span class="font-semibold tracking-tight">gosidian</span>
      <span class="text-xs text-text-muted hidden sm:inline">v2.0</span>
      <RouterLink
        to="/search"
        class="text-xs text-text-muted hover:text-text px-2 py-0.5 rounded border border-border ml-3 hidden md:inline-flex items-center gap-1"
      >
        <Search class="w-3 h-3" />
        <span>{{ t('nav.search', 'Search') }}</span>
        <kbd class="opacity-60">⌘K</kbd>
      </RouterLink>
      <nav class="hidden md:flex items-center gap-3 ml-3 text-xs text-text-muted">
        <RouterLink to="/projects" class="hover:text-text">{{ t('nav.projects') }}</RouterLink>
        <RouterLink to="/tags" class="hover:text-text">{{ t('nav.tags') }}</RouterLink>
        <RouterLink to="/graph" class="hover:text-text">{{ t('nav.graph') }}</RouterLink>
        <RouterLink v-if="auth.canWrite" to="/trash" class="hover:text-text">{{ t('nav.trash') }}</RouterLink>
        <RouterLink v-if="auth.canWrite" to="/settings" class="hover:text-text">{{ t('nav.settings') }}</RouterLink>
        <RouterLink
          v-if="auth.isOwner"
          to="/admin"
          class="hover:text-text"
        >{{ t('nav.admin', 'Admin') }}</RouterLink>
      </nav>
    </div>
    <div class="flex items-center gap-3 text-sm">
      <span class="text-text-muted hidden md:inline">{{ auth.username }}</span>
      <span
        v-if="auth.isOwner"
        class="px-2 py-0.5 rounded text-xs bg-accent/20 text-accent"
      >owner</span>
      <span
        v-else-if="auth.isGuest"
        class="px-2 py-0.5 rounded text-xs border border-border text-text-muted"
        title="Read-only guest — public projects only"
      >guest</span>
      <button
        type="button"
        class="px-2 py-1 rounded text-xs hover:bg-surface-hover inline-flex items-center gap-1"
        @click="handleLogout"
      >
        <LogOut class="w-3 h-3" />
        {{ t('common.logout') }}
      </button>
    </div>
  </header>
</template>
