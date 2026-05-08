<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const auth = useAuthStore()
const route = useRoute()

interface Tab { name: string; to: string; label: string }
const tabs: Tab[] = [
  { name: 'admin-users', to: '/admin/users', label: 'Users' },
  { name: 'admin-tokens', to: '/admin/tokens', label: 'MCP tokens' },
  { name: 'admin-spa-tokens', to: '/admin/spa-tokens', label: 'SPA tokens' },
  { name: 'admin-invites', to: '/admin/invites', label: 'Invites' },
  { name: 'admin-audit', to: '/admin/audit', label: 'Audit' },
]

const isActive = computed(() => (name: string) => route.name === name)
</script>

<template>
  <div class="p-6 max-w-6xl mx-auto">
    <h1 class="text-2xl font-semibold mb-1">Admin</h1>
    <p
      v-if="!auth.isOwner"
      class="text-danger text-sm mb-6"
    >
      You need owner role to access these pages.
    </p>

    <template v-else>
      <nav class="flex gap-1 mb-6 border-b border-border text-sm">
        <RouterLink
          v-for="t in tabs"
          :key="t.name"
          :to="t.to"
          class="px-3 py-2 -mb-px border-b-2"
          :class="isActive(t.name) ? 'border-accent text-accent' : 'border-transparent hover:text-text'"
        >{{ t.label }}</RouterLink>
      </nav>

      <RouterView />
    </template>
  </div>
</template>
