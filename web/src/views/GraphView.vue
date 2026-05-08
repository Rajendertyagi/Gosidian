<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
import { fetchGraph, type GraphResponse } from '@/api/graph'
import { listProjects, type Project } from '@/api/projects'
import { listTags, type TagCount } from '@/api/tags'
import { suggestNoteTitles, type NoteTitleHit } from '@/api/noteTitles'
import SearchSelect from '@/components/primitives/SearchSelect.vue'

// Lazy-load Cytoscape via the canvas component — keeps the graph
// chunk separate from the AppShell shipping bundle.
const GraphCanvas = defineAsyncComponent(
  () => import('@/components/graph/GraphCanvas.vue'),
)

const route = useRoute()
const router = useRouter()

const data = ref<GraphResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const project = ref<string>(strFromQuery('project'))
const tag = ref<string>(strFromQuery('tag'))
const focus = ref<string>(strFromQuery('focus'))
const depth = ref<number>(numFromQuery('depth', 2))
const minDegree = ref<number>(numFromQuery('min_degree', 0))
const limit = ref<number>(numFromQuery('limit', 0))

function strFromQuery(key: string): string {
  const v = route.query[key]
  return typeof v === 'string' ? v : ''
}
function numFromQuery(key: string, fallback: number): number {
  const v = route.query[key]
  if (typeof v !== 'string') return fallback
  const parsed = parseInt(v, 10)
  return Number.isFinite(parsed) ? parsed : fallback
}

const params = computed(() => ({
  project: project.value || undefined,
  tag: tag.value || undefined,
  focus: focus.value || undefined,
  depth: focus.value ? depth.value : undefined,
  min_degree: minDegree.value > 0 ? minDegree.value : undefined,
  limit: limit.value > 0 ? limit.value : undefined,
}))

async function load() {
  loading.value = true
  error.value = null
  try {
    data.value = await fetchGraph(params.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load graph'
    data.value = null
  } finally {
    loading.value = false
  }
}

const debouncedLoad = useDebounceFn(load, 200)

watch(
  params,
  (next) => {
    const query: Record<string, string> = {}
    if (next.project) query.project = next.project
    if (next.tag) query.tag = next.tag
    if (next.focus) query.focus = next.focus
    if (next.depth) query.depth = String(next.depth)
    if (next.min_degree) query.min_degree = String(next.min_degree)
    if (next.limit) query.limit = String(next.limit)
    void router.replace({ query })
    void debouncedLoad()
  },
  { deep: true },
)

function reset() {
  project.value = ''
  tag.value = ''
  focus.value = ''
  depth.value = 2
  minDegree.value = 0
  limit.value = 0
}

function onSelect(path: string) {
  void router.push('/notes/' + encodeURIComponent(path))
}

// --- Picker datasets ---------------------------------------------
//
// Project: vault top-level dirs sorted by mtime desc (most recent
//          activity first) — see /api/v1/projects + the mod_time
//          field added in the same change.
// Tag:     /api/v1/tags returns items already sorted by count desc
//          server-side (most-used first). No client sort needed.
// Focus:   /api/v1/note-titles?q= empty returns the 50 most-recently
//          edited notes (mtime desc); typing in the textbox falls
//          through SearchSelect's local filter rather than re-firing
//          the API. For 50 entries that's plenty of resolution.
const projects = ref<Project[]>([])
const tags = ref<TagCount[]>([])
const notes = ref<NoteTitleHit[]>([])

const sortedProjects = computed<Project[]>(() => {
  return [...projects.value].sort((a, b) => {
    const am = a.mod_time ?? ''
    const bm = b.mod_time ?? ''
    if (am === bm) return a.name.localeCompare(b.name)
    if (!am) return 1
    if (!bm) return -1
    return bm.localeCompare(am)
  })
})

onMounted(async () => {
  void load()
  // Fire dataset fetches in parallel; failures degrade to "no
  // suggestions, free typing still works".
  const [pRes, tRes, nRes] = await Promise.allSettled([
    listProjects(),
    listTags(),
    suggestNoteTitles('', 50),
  ])
  if (pRes.status === 'fulfilled') projects.value = pRes.value
  if (tRes.status === 'fulfilled') tags.value = tRes.value
  if (nRes.status === 'fulfilled') notes.value = nRes.value
})
</script>

<template>
  <div class="flex h-full">
    <aside class="w-72 shrink-0 border-r border-border bg-bg-elevated p-4 space-y-4 overflow-auto">
      <h1 class="text-lg font-semibold">Graph</h1>

      <div class="block text-sm">
        <span class="text-text-muted text-xs">Project</span>
        <SearchSelect
          v-model="project"
          class="mt-1"
          :items="sortedProjects"
          :value-key="(p: Project) => p.name"
          :label="(p: Project) => p.name"
          :secondary="(p: Project) => String(p.note_count)"
          placeholder="(all) — type to search"
        />
      </div>

      <div class="block text-sm">
        <span class="text-text-muted text-xs">Tag</span>
        <SearchSelect
          v-model="tag"
          class="mt-1"
          :items="tags"
          :value-key="(t: TagCount) => t.tag"
          :label="(t: TagCount) => '#' + t.tag"
          :secondary="(t: TagCount) => String(t.count)"
          placeholder="(no tag) — type to search"
        />
      </div>

      <div class="block text-sm">
        <span class="text-text-muted text-xs">Focus (path)</span>
        <SearchSelect
          v-model="focus"
          class="mt-1"
          :items="notes"
          :value-key="(n: NoteTitleHit) => n.path"
          :label="(n: NoteTitleHit) => n.title || n.path"
          :secondary="(n: NoteTitleHit) => n.path"
          placeholder="(no focus) — type to search"
        />
      </div>

      <label class="block text-sm">
        <span class="text-text-muted text-xs">Depth (hops, when focus is set)</span>
        <input
          v-model.number="depth"
          type="number"
          min="1"
          max="6"
          class="mt-1 w-full rounded bg-bg border border-border px-2 py-1.5 text-sm"
        />
      </label>
      <label class="block text-sm">
        <span class="text-text-muted text-xs">Min degree (drop leaves below)</span>
        <input
          v-model.number="minDegree"
          type="number"
          min="0"
          max="20"
          class="mt-1 w-full rounded bg-bg border border-border px-2 py-1.5 text-sm"
        />
      </label>
      <label class="block text-sm">
        <span class="text-text-muted text-xs">Limit (cap nodes; top-degree wins)</span>
        <input
          v-model.number="limit"
          type="number"
          min="0"
          max="2000"
          class="mt-1 w-full rounded bg-bg border border-border px-2 py-1.5 text-sm"
        />
      </label>

      <button
        type="button"
        class="w-full text-xs px-2 py-1 rounded border border-border hover:bg-surface-hover"
        @click="reset"
      >Reset</button>

      <div v-if="data" class="text-xs text-text-muted space-y-1 pt-3 border-t border-border">
        <p>Nodes: <strong class="text-text">{{ data.stats.node_count }}</strong></p>
        <p>Edges: <strong class="text-text">{{ data.stats.edge_count }}</strong></p>
        <p v-if="data.stats.truncated" class="text-warning">Truncated by limit</p>
        <p v-if="data.stats.filter" class="font-mono break-all">{{ data.stats.filter }}</p>
      </div>
    </aside>

    <section class="flex-1 relative">
      <p
        v-if="loading"
        class="absolute top-3 left-3 z-10 text-xs text-text-muted bg-bg-elevated px-2 py-1 rounded border border-border"
      >Loading…</p>
      <p
        v-else-if="error"
        class="absolute top-3 left-3 z-10 text-xs text-danger bg-bg-elevated px-2 py-1 rounded border border-danger"
      >{{ error }}</p>
      <GraphCanvas
        v-if="data"
        :nodes="data.nodes"
        :edges="data.edges"
        @select="onSelect"
      />
      <p
        v-else-if="!loading"
        class="p-8 text-text-muted text-sm"
      >No data yet.</p>
    </section>
  </div>
</template>
