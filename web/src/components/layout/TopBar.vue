<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useWindowsStore } from '@/stores/windows'
import { planciaKey } from '@/composables/usePlanciaSync'
import { Search, LogOut } from 'lucide-vue-next'
import InsightsBadge from '@/components/layout/InsightsBadge.vue'

const { t } = useI18n()

const router = useRouter()
const auth = useAuthStore()
const windows = useWindowsStore()

function openSearch() {
  windows.open({ type: 'search', key: planciaKey('search') })
}

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
      <button
        type="button"
        class="text-xs text-text-muted hover:text-text px-2 py-0.5 rounded border border-border ml-3 inline-flex items-center gap-1"
        @click="openSearch"
      >
        <Search class="w-3 h-3" />
        <span>{{ t('nav.search', 'Search') }}</span>
        <kbd class="opacity-60">⌘K</kbd>
      </button>
    </div>
    <div class="flex items-center gap-3 text-sm">
      <InsightsBadge v-if="auth.isOwner" />
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
