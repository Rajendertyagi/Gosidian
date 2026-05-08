<script setup lang="ts">
/**
 * NoteEditView — Phase 3.3 textarea+preview editor. Phase 4 swaps
 * the textarea for CodeMirror 6, the rest of the wiring (load,
 * save with If-Match, debounced preview, conflict UX) is built so
 * the swap is a one-component substitution.
 *
 * Key invariants:
 *   - getNote() yields the etag we ship back as If-Match on PUT.
 *     A concurrent write (MCP, other tab) lands a 412 the api
 *     client emits as `note.concurrency-conflict` — picked up by
 *     the AppShell's ConflictDialog automatically.
 *   - "Force overwrite" PUT skips If-Match (handled inside the
 *     ConflictDialog reload/force flow Phase 3.4 wires up; this
 *     view emits the events but doesn't drive the dialog).
 *   - Editor mode (split / preview / editor only) persists in
 *     localStorage so the user's preference survives reloads.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch, defineAsyncComponent } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
import { getNote, updateNote, deleteNote, type Note } from '@/api/notes'
import { renderPreview } from '@/api/preview'
import MarkdownPreview from '@/components/domain/MarkdownPreview.vue'
import { useTreeStore } from '@/stores/tree'

// CodeMirror is ~150KB tree-shaken — lazy-load so the read-only
// NoteView, search, projects, etc. don't pay for it.
const CodeMirrorEditor = defineAsyncComponent(
  () => import('@/components/editor/CodeMirrorEditor.vue'),
)

type EditorMode = 'editor' | 'split' | 'stacked' | 'preview'

const route = useRoute()
const router = useRouter()
const treeStore = useTreeStore()

const note = ref<Note | null>(null)
const draft = ref<string>('')
const previewHTML = ref<string>('')
const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const dirty = ref(false)
const lastSavedAt = ref<string | null>(null)

const path = computed(() => {
  const raw = route.params.path
  return Array.isArray(raw) ? raw.join('/') : (raw ?? '')
})

// Derive project (top-level vault folder) from the note path; the
// editor's paste/drop upload uses it to pick the project's
// attachments/ folder. Notes at the vault root → undefined → upload
// lands in vault-root attachments/.
const project = computed(() => {
  const parts = path.value.split('/')
  return parts.length > 1 ? parts[0] : undefined
})

const STORAGE_MODE = 'gosidian.editorMode'
const mode = ref<EditorMode>(loadMode())

function loadMode(): EditorMode {
  try {
    const v = localStorage.getItem(STORAGE_MODE)
    if (v === 'editor' || v === 'split' || v === 'stacked' || v === 'preview') return v
  } catch {
    /* ignore */
  }
  return 'split'
}
watch(mode, (m) => {
  try {
    localStorage.setItem(STORAGE_MODE, m)
  } catch {
    /* ignore */
  }
})

async function load() {
  if (!path.value) return
  loading.value = true
  error.value = null
  try {
    const fetched = await getNote(path.value)
    note.value = fetched
    draft.value = fetched.content
    previewHTML.value = await renderPreview(fetched.content)
    dirty.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load note'
    note.value = null
  } finally {
    loading.value = false
  }
}

const refreshPreview = useDebounceFn(async () => {
  try {
    previewHTML.value = await renderPreview(draft.value)
  } catch {
    /* preview failure shouldn't block editing — leave the previous render */
  }
}, 300)

watch(draft, () => {
  if (!note.value) return
  dirty.value = draft.value !== note.value.content
  if (mode.value !== 'editor') void refreshPreview()
  // 'stacked' also needs the preview refreshed; the condition above
  // already covers it (any mode that isn't pure 'editor').
})

async function save() {
  if (!note.value || !dirty.value || saving.value) return
  saving.value = true
  error.value = null
  try {
    const updated = await updateNote(note.value.path, {
      content: draft.value,
      ifMatch: note.value.etag,
    })
    note.value = updated
    draft.value = updated.content
    dirty.value = false
    lastSavedAt.value = new Date().toLocaleTimeString()
  } catch (e) {
    // 412 fires the conflict event the AppShell ConflictDialog
    // listens for; we surface a thin error band so the user
    // knows the local save failed even before the dialog
    // closes.
    error.value = e instanceof Error ? e.message : 'Save failed'
  } finally {
    saving.value = false
  }
}

async function destroy() {
  if (!note.value) return
  if (!confirm(`Delete ${note.value.path}? This moves it to the trash.`)) return
  try {
    await deleteNote(note.value.path)
    treeStore.invalidateAll()
    await router.push('/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    void save()
  }
}

onMounted(() => {
  void load()
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
watch(path, load)

onBeforeRouteLeave((to, _from, next) => {
  if (dirty.value) {
    if (!confirm('You have unsaved changes. Leave anyway?')) {
      next(false)
      return
    }
  }
  next()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <header class="flex items-center gap-3 px-6 py-3 border-b border-border bg-bg-elevated">
      <RouterLink
        v-if="note"
        :to="'/notes/' + encodeURIComponent(note.path)"
        class="text-sm text-text-muted hover:text-text"
      >← Read view</RouterLink>
      <span v-if="note" class="font-mono text-sm text-text-muted truncate">
        {{ note.path }}
      </span>
      <span v-if="dirty" class="text-xs text-warning">●</span>
      <span v-else-if="lastSavedAt" class="text-xs text-success">saved {{ lastSavedAt }}</span>

      <div class="flex-1" />

      <div class="inline-flex rounded border border-border overflow-hidden text-xs">
        <button
          v-for="m in (['editor', 'split', 'stacked', 'preview'] as EditorMode[])"
          :key="m"
          type="button"
          class="px-2 py-1"
          :class="mode === m ? 'bg-accent text-accent-fg' : 'hover:bg-surface-hover'"
          @click="mode = m"
        >{{ m }}</button>
      </div>
      <button
        type="button"
        class="text-xs px-2 py-1 rounded bg-accent text-accent-fg hover:bg-accent-hover disabled:opacity-50"
        :disabled="!dirty || saving"
        @click="save"
      >{{ saving ? 'Saving…' : 'Save' }}</button>
      <button
        v-if="note"
        type="button"
        class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
        @click="destroy"
      >Delete</button>
    </header>

    <p v-if="loading" class="p-6 text-text-muted">Loading…</p>
    <p v-else-if="error" class="p-6 text-danger">{{ error }}</p>

    <div v-else-if="note" class="flex-1 grid min-h-0" :class="{
      'grid-cols-1': mode !== 'split',
      'grid-cols-2': mode === 'split',
      'grid-rows-2': mode === 'stacked',
    }">
      <div
        v-if="mode !== 'preview'"
        class="h-full overflow-hidden"
        :class="{
          'border-r border-border': mode === 'split',
          'border-b border-border': mode === 'stacked',
        }"
      >
        <CodeMirrorEditor
          v-model="draft"
          :project="project"
          placeholder="Markdown…"
        />
      </div>
      <div
        v-if="mode !== 'editor'"
        class="overflow-auto p-6 max-w-none"
      >
        <MarkdownPreview :html="previewHTML" />
      </div>
    </div>
  </div>
</template>
