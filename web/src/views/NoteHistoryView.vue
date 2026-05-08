<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getHistory, type HistoryEntry } from '@/api/history'

const route = useRoute()
const entries = ref<HistoryEntry[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const path = computed(() => {
  const raw = route.params.path
  return Array.isArray(raw) ? raw.join('/') : (raw ?? '')
})

async function load() {
  if (!path.value) return
  loading.value = true
  error.value = null
  try {
    entries.value = await getHistory(path.value, 100)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load history'
    entries.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(path, load)
</script>

<template>
  <div class="p-8 max-w-4xl mx-auto">
    <header class="flex items-center gap-3 mb-4">
      <RouterLink
        :to="'/notes/' + encodeURIComponent(path)"
        class="text-sm text-text-muted hover:text-text"
      >← Note</RouterLink>
      <h1 class="text-xl font-semibold">History</h1>
      <span class="font-mono text-sm text-text-muted truncate">{{ path }}</span>
    </header>

    <p v-if="loading" class="text-text-muted">Loading…</p>
    <p v-else-if="error" class="text-danger">{{ error }}</p>
    <p v-else-if="!entries.length" class="text-text-muted text-sm">
      No git history available — either git-sync is disabled or this note was never committed.
    </p>

    <ol v-else class="space-y-2">
      <li
        v-for="e in entries"
        :key="e.sha"
        class="rounded border border-border bg-surface px-4 py-3"
      >
        <div class="flex items-baseline gap-3 text-xs">
          <code class="font-mono text-accent">{{ e.short_sha }}</code>
          <span class="text-text-muted">{{ e.date }}</span>
          <span class="text-text-muted">{{ e.author }}</span>
        </div>
        <p class="text-sm mt-1">{{ e.subject }}</p>
      </li>
    </ol>
  </div>
</template>
