<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useRecentlyViewed } from '@/composables/useRecentlyViewed'

const auth = useAuthStore()
const recents = useRecentlyViewed()
</script>

<template>
  <div class="p-8 max-w-3xl mx-auto">
    <h1 class="text-2xl font-semibold mb-1">
      Welcome back, {{ auth.username || 'friend' }}
    </h1>
    <p class="text-text-muted mb-6">
      Your vault is open. Pick a note from the sidebar or jump back in
      below.
    </p>

    <section v-if="recents.entries.value.length">
      <h2 class="text-sm font-semibold uppercase tracking-wide text-text-muted mb-2">
        Recent
      </h2>
      <ul class="space-y-1.5">
        <li
          v-for="entry in recents.entries.value"
          :key="entry.path"
          class="rounded border border-border bg-surface px-3 py-2"
        >
          <RouterLink
            :to="'/notes/' + encodeURIComponent(entry.path)"
            class="font-medium hover:text-accent"
          >{{ entry.title }}</RouterLink>
          <p class="text-xs text-text-muted truncate">{{ entry.path }}</p>
        </li>
      </ul>
    </section>

    <section v-else>
      <p class="text-text-muted text-sm italic">
        No recent notes yet. Open one from the sidebar to populate this list.
      </p>
    </section>
  </div>
</template>
