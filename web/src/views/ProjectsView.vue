<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  listProjects,
  createProject,
  updateProject,
  deleteProject,
  type Project,
} from '@/api/projects'
import { useTreeStore } from '@/stores/tree'

const projects = ref<Project[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const newName = ref('')
const treeStore = useTreeStore()

async function load() {
  loading.value = true
  error.value = null
  try {
    projects.value = await listProjects()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!newName.value.trim()) return
  try {
    await createProject(newName.value.trim())
    newName.value = ''
    await load()
    treeStore.invalidateAll()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  }
}

async function toggleHidden(p: Project) {
  await updateProject(p.name, { hidden_from_mcp: !p.hidden_from_mcp })
  await load()
}

async function toggleSkip(p: Project) {
  await updateProject(p.name, { skip_git_sync: !p.skip_git_sync })
  await load()
}

async function rename(p: Project) {
  const newSlug = prompt(`Rename "${p.name}" to:`, p.name)
  if (!newSlug || newSlug === p.name) return
  try {
    await updateProject(p.name, { new_name: newSlug })
    await load()
    treeStore.invalidateAll()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Rename failed'
  }
}

async function destroy(p: Project) {
  if (!confirm(`Delete project "${p.name}" and ${p.note_count} note(s)?`)) return
  try {
    await deleteProject(p.name)
    await load()
    treeStore.invalidateAll()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

onMounted(load)
</script>

<template>
  <div class="p-8 max-w-3xl mx-auto">
    <h1 class="text-2xl font-semibold mb-1">Projects</h1>
    <p class="text-sm text-text-muted mb-6">
      Top-level vault folders. Toggle <em>skip-git</em> to exclude from auto-commit, or
      <em>hidden</em> to keep the project invisible to MCP agents.
    </p>

    <form
      class="flex gap-2 mb-6"
      @submit.prevent="handleCreate"
    >
      <input
        v-model.trim="newName"
        type="text"
        placeholder="new-project"
        class="flex-1 rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
      />
      <button
        type="submit"
        class="px-3 py-2 rounded bg-accent text-accent-fg hover:bg-accent-hover"
      >Create</button>
    </form>

    <p v-if="loading" class="text-text-muted">Loading…</p>
    <p v-else-if="error" class="text-danger">{{ error }}</p>

    <ul v-else class="space-y-2">
      <li
        v-for="p in projects"
        :key="p.name"
        class="rounded border border-border bg-surface px-4 py-3 flex items-center gap-3"
      >
        <RouterLink
          :to="'/?project=' + encodeURIComponent(p.name)"
          class="font-medium hover:text-accent flex-1"
        >{{ p.name }}</RouterLink>
        <span class="text-xs text-text-muted">{{ p.note_count }} notes</span>

        <button
          type="button"
          class="text-xs px-2 py-1 rounded"
          :class="p.skip_git_sync ? 'bg-warning/20 text-warning' : 'border border-border'"
          :title="p.skip_git_sync ? 'Click to re-enable git sync' : 'Click to skip from git sync'"
          @click="toggleSkip(p)"
        >skip-git</button>
        <button
          type="button"
          class="text-xs px-2 py-1 rounded"
          :class="p.hidden_from_mcp ? 'bg-warning/20 text-warning' : 'border border-border'"
          :title="p.hidden_from_mcp ? 'Click to expose to MCP again' : 'Click to hide from MCP'"
          @click="toggleHidden(p)"
        >hidden</button>

        <button
          type="button"
          class="text-xs px-2 py-1 rounded hover:bg-surface-hover"
          @click="rename(p)"
        >Rename</button>
        <button
          type="button"
          class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
          @click="destroy(p)"
        >Delete</button>
      </li>
    </ul>
  </div>
</template>
