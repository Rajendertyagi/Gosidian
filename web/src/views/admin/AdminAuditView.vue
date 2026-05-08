<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { tailAudit, type AuditEntry } from '@/api/admin'

const entries = ref<AuditEntry[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const filters = reactive<{ actor: string; action: string; source: string; path_prefix: string; limit: number }>({
  actor: '',
  action: '',
  source: '',
  path_prefix: '',
  limit: 100,
})

async function load() {
  loading.value = true
  error.value = null
  try {
    entries.value = await tailAudit({
      actor: filters.actor || undefined,
      action: filters.action || undefined,
      source: filters.source || undefined,
      path_prefix: filters.path_prefix || undefined,
      limit: filters.limit,
    })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

function reset() {
  filters.actor = ''
  filters.action = ''
  filters.source = ''
  filters.path_prefix = ''
  filters.limit = 100
  void load()
}

onMounted(load)
</script>

<template>
  <form
    class="grid grid-cols-2 md:grid-cols-5 gap-2 mb-4 text-sm"
    @submit.prevent="load"
  >
    <input
      v-model.trim="filters.actor"
      type="text"
      placeholder="actor"
      class="rounded bg-bg-elevated border border-border px-2 py-1.5"
    />
    <input
      v-model.trim="filters.action"
      type="text"
      placeholder="action (e.g. note_create)"
      class="rounded bg-bg-elevated border border-border px-2 py-1.5"
    />
    <input
      v-model.trim="filters.source"
      type="text"
      placeholder="source (http/mcp/...)"
      class="rounded bg-bg-elevated border border-border px-2 py-1.5"
    />
    <input
      v-model.trim="filters.path_prefix"
      type="text"
      placeholder="path prefix"
      class="rounded bg-bg-elevated border border-border px-2 py-1.5"
    />
    <div class="flex gap-1">
      <input
        v-model.number="filters.limit"
        type="number"
        min="10"
        max="500"
        class="flex-1 rounded bg-bg-elevated border border-border px-2 py-1.5"
      />
      <button
        type="submit"
        class="px-3 py-1.5 rounded bg-accent text-accent-fg hover:bg-accent-hover text-xs"
      >Apply</button>
      <button
        type="button"
        class="px-3 py-1.5 rounded border border-border hover:bg-surface-hover text-xs"
        @click="reset"
      >Reset</button>
    </div>
  </form>

  <p v-if="loading" class="text-text-muted">Loading…</p>
  <p v-else-if="error" class="text-danger">{{ error }}</p>

  <p v-else-if="!entries.length" class="text-text-muted text-sm">No audit entries match the filters.</p>

  <div v-else class="rounded border border-border overflow-x-auto">
    <table class="w-full text-xs font-mono">
      <thead class="text-text-muted uppercase tracking-wide bg-bg-elevated">
        <tr>
          <th class="text-left py-2 px-2">TS</th>
          <th class="text-left py-2 px-2">Source</th>
          <th class="text-left py-2 px-2">Actor</th>
          <th class="text-left py-2 px-2">Action</th>
          <th class="text-left py-2 px-2">Path</th>
          <th class="text-right py-2 px-2">Size</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(e, i) in entries"
          :key="i"
          class="border-t border-border"
        >
          <td class="py-1 px-2 whitespace-nowrap">{{ e.ts }}</td>
          <td class="py-1 px-2">{{ e.source }}</td>
          <td class="py-1 px-2">{{ e.actor || e.token || '—' }}</td>
          <td class="py-1 px-2">{{ e.action }}</td>
          <td class="py-1 px-2 truncate max-w-[20rem]" :title="e.path">{{ e.path || '—' }}</td>
          <td class="py-1 px-2 text-right">{{ e.size || '' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
