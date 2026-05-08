<script setup lang="ts">
/**
 * NoteView — read-only render of a single note. Markdown is shipped
 * to /api/v1/preview so wikilinks resolve against the live index;
 * the response is sanitized via DOMPurify in MarkdownPreview before
 * v-html. Phase 3.3 adds the edit pane next to this read view.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getNote, type Note } from '@/api/notes'
import { renderPreview } from '@/api/preview'
import { useRecentlyViewed } from '@/composables/useRecentlyViewed'
import MarkdownPreview from '@/components/domain/MarkdownPreview.vue'

const route = useRoute()
const recents = useRecentlyViewed()
const note = ref<Note | null>(null)
const html = ref<string>('')
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
  html.value = ''
  try {
    const fetched = await getNote(path.value)
    note.value = fetched
    if (fetched) recents.record(fetched.path, fetched.title || fetched.path)
    html.value = await renderPreview(fetched.content)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load note'
    note.value = null
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(path, load)
</script>

<template>
  <div class="p-8 max-w-3xl mx-auto">
    <p v-if="loading" class="mt-4 text-text-muted">Loading…</p>
    <p v-else-if="error" class="mt-4 text-danger">{{ error }}</p>

    <article v-else-if="note">
      <header class="mb-6 flex items-baseline gap-3">
        <h1 class="text-2xl font-semibold flex-1">{{ note.title || note.path }}</h1>
        <RouterLink
          :to="'/notes/' + encodeURIComponent(note.path) + '/history'"
          class="text-sm text-text-muted hover:text-text"
        >History</RouterLink>
        <RouterLink
          :to="'/notes/' + encodeURIComponent(note.path) + '/edit'"
          class="text-sm px-3 py-1 rounded bg-accent text-accent-fg hover:bg-accent-hover"
        >Edit</RouterLink>
      </header>
      <p class="text-xs text-text-muted font-mono mb-6">
        {{ note.path }} · etag {{ note.etag.slice(0, 12) }} · {{ note.size }} bytes
      </p>
      <MarkdownPreview :html="html" />
    </article>
  </div>
</template>
