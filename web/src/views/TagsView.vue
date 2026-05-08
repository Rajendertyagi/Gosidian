<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listTags, notesByTag, type TagCount, type NoteSummary } from '@/api/tags'

const route = useRoute()

const tags = ref<TagCount[]>([])
const notes = ref<NoteSummary[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const selectedTag = computed(() => {
  const raw = route.params.tag
  return Array.isArray(raw) ? raw[0] : (raw ?? '')
})

async function loadTags() {
  loading.value = true
  error.value = null
  try {
    tags.value = await listTags()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load tags'
  } finally {
    loading.value = false
  }
}

async function loadNotes(tag: string) {
  if (!tag) {
    notes.value = []
    return
  }
  try {
    notes.value = await notesByTag(tag)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load notes for tag'
  }
}

onMounted(async () => {
  await loadTags()
  if (selectedTag.value) await loadNotes(selectedTag.value)
})

watch(selectedTag, (t) => {
  if (t) void loadNotes(t)
})
</script>

<template>
  <div class="p-8 max-w-5xl mx-auto grid gap-8 grid-cols-1 md:grid-cols-3">
    <aside class="md:col-span-1">
      <h1 class="text-xl font-semibold mb-3">Tags</h1>
      <p v-if="loading" class="text-text-muted text-sm">Loading…</p>
      <p v-else-if="error" class="text-danger text-sm">{{ error }}</p>
      <ul v-else class="space-y-1">
        <li v-for="t in tags" :key="t.tag">
          <RouterLink
            :to="'/tags/' + encodeURIComponent(t.tag)"
            class="flex justify-between items-center px-2 py-1 rounded hover:bg-surface-hover"
            :class="selectedTag === t.tag ? 'bg-surface-hover' : ''"
          >
            <span class="truncate text-sm">#{{ t.tag }}</span>
            <span class="text-xs text-text-muted">{{ t.count }}</span>
          </RouterLink>
        </li>
      </ul>
    </aside>

    <section class="md:col-span-2">
      <template v-if="selectedTag">
        <h2 class="text-lg font-semibold mb-3">
          Notes tagged <span class="text-accent">#{{ selectedTag }}</span>
        </h2>
        <ul v-if="notes.length" class="space-y-2">
          <li
            v-for="n in notes"
            :key="n.path"
            class="rounded border border-border bg-surface px-3 py-2"
          >
            <RouterLink
              :to="'/notes/' + encodeURIComponent(n.path)"
              class="font-medium hover:text-accent"
            >{{ n.title || n.path }}</RouterLink>
            <p class="text-xs text-text-muted font-mono">{{ n.path }}</p>
          </li>
        </ul>
        <p v-else class="text-text-muted text-sm">No notes tagged with this label.</p>
      </template>
      <p v-else class="text-text-muted text-sm">
        Pick a tag on the left to see the notes that carry it.
      </p>
    </section>
  </div>
</template>
