<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
import { search, type SearchHit } from '@/api/search'

const route = useRoute()
const router = useRouter()

const query = ref<string>(typeof route.query.q === 'string' ? route.query.q : '')
const project = ref<string>(typeof route.query.project === 'string' ? route.query.project : '')
const hits = ref<SearchHit[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const lastSubmitted = ref('')

async function run(q: string, p: string) {
  if (!q.trim()) {
    hits.value = []
    error.value = null
    return
  }
  loading.value = true
  error.value = null
  try {
    hits.value = await search({ q, project: p || undefined, limit: 50 })
    lastSubmitted.value = q
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Search failed'
    hits.value = []
  } finally {
    loading.value = false
  }
}

const debounced = useDebounceFn(() => {
  router.replace({
    query: {
      ...(query.value ? { q: query.value } : {}),
      ...(project.value ? { project: project.value } : {}),
    },
  })
  void run(query.value, project.value)
}, 300)

watch([query, project], debounced)
onMounted(() => {
  if (query.value) void run(query.value, project.value)
})
</script>

<template>
  <div class="p-8 max-w-3xl mx-auto">
    <h1 class="text-xl font-semibold mb-4">Search</h1>

    <div class="flex gap-2 mb-6">
      <input
        v-model="query"
        type="search"
        autofocus
        placeholder="Full-text search…"
        class="flex-1 rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
      />
      <input
        v-model="project"
        type="text"
        placeholder="project filter (optional)"
        class="w-48 rounded bg-bg-elevated border border-border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-accent"
      />
    </div>

    <p v-if="loading" class="text-text-muted text-sm">Searching…</p>
    <p v-else-if="error" class="text-danger text-sm">{{ error }}</p>
    <p
      v-else-if="lastSubmitted && hits.length === 0"
      class="text-text-muted text-sm"
    >
      No matches for <strong class="font-mono">{{ lastSubmitted }}</strong>.
    </p>

    <ul v-if="hits.length" class="space-y-3">
      <li
        v-for="hit in hits"
        :key="hit.path"
        class="rounded border border-border bg-surface px-4 py-3"
      >
        <RouterLink
          :to="'/notes/' + encodeURIComponent(hit.path)"
          class="font-medium hover:text-accent"
        >{{ hit.title || hit.path }}</RouterLink>
        <p class="text-xs text-text-muted font-mono mt-0.5">{{ hit.path }}</p>
        <p
          v-if="hit.snippet"
          class="text-sm text-text-muted mt-2 line-clamp-2"
        >{{ hit.snippet }}</p>
      </li>
    </ul>
  </div>
</template>
