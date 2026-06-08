<script setup lang="ts">
/** NoteHistoryView — git history of a note as a plancia window. `path` comes
 *  from window props; the back link opens the note read window. */
import { onMounted, ref, watch, computed, inject } from 'vue'
import { getHistory, type HistoryEntry } from '@/api/history'
import { useWindowsStore, type OpenSpec } from '@/stores/windows'
import { planciaKey } from '@/composables/usePlanciaSync'

const props = defineProps<{ path: string }>()

const store = useWindowsStore()
const openWindow = inject<(spec: OpenSpec) => string>('openWindow', (s) => store.open(s))

const path = computed(() => props.path)
const entries = ref<HistoryEntry[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

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

function openNote() {
  openWindow({
    type: 'note',
    key: planciaKey('note', path.value),
    title: (path.value.split('/').pop() ?? path.value).replace(/\.md$/, ''),
    props: { path: path.value },
  })
}

onMounted(load)
watch(path, load)
</script>

<template>
  <div class="p-6 max-w-4xl mx-auto">
    <header class="flex items-center gap-3 mb-4">
      <button
        type="button"
        class="text-sm text-text-muted hover:text-text"
        @click="openNote"
      >Note</button>
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
