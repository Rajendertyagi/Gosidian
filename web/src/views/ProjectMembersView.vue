<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  listProjectMembers,
  setProjectMember,
  removeProjectMember,
  type ProjectMember,
} from '@/api/projects'
import { listUsers, type AdminUser } from '@/api/admin'

const props = defineProps<{ project: string }>()

const members = ref<ProjectMember[]>([])
const users = ref<AdminUser[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const addUser = ref('')
const addLevel = ref<'read' | 'write'>('read')
const busy = ref(false)

// Users eligible to add: not the owner (who sees everything), not disabled, and
// not already a member.
const candidates = computed(() => {
  const have = new Set(members.value.map((m) => m.user_id))
  return users.value.filter((u) => u.role !== 'owner' && !u.disabled_at && !have.has(u.id))
})

async function load() {
  loading.value = true
  error.value = null
  try {
    const [m, u] = await Promise.all([listProjectMembers(props.project), listUsers()])
    members.value = m
    users.value = u
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function add() {
  if (!addUser.value) return
  busy.value = true
  error.value = null
  try {
    await setProjectMember(props.project, addUser.value, addLevel.value)
    addUser.value = ''
    addLevel.value = 'read'
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Add failed'
  } finally {
    busy.value = false
  }
}

async function changeLevel(m: ProjectMember, level: string) {
  if (level !== 'read' && level !== 'write') return
  try {
    await setProjectMember(props.project, m.user_id, level)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Update failed'
  }
}

async function remove(m: ProjectMember) {
  try {
    await removeProjectMember(props.project, m.user_id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Remove failed'
  }
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-xl mx-auto">
    <h2 class="text-lg font-semibold mb-1">Members · {{ project }}</h2>
    <p class="text-sm text-text-muted mb-4">
      Grant specific users access to this project. Effective only when
      <em>Project access</em> is set to per-project (Settings). Owners always have full access;
      public projects stay readable by everyone.
    </p>

    <p v-if="loading" class="text-text-muted">Loading…</p>
    <p v-if="error" class="text-sm text-danger mb-2">{{ error }}</p>

    <ul class="space-y-2 mb-4">
      <li
        v-for="m in members"
        :key="m.user_id"
        class="flex items-center gap-3 rounded border border-border bg-surface px-3 py-2"
      >
        <span class="flex-1 text-sm font-medium">{{ m.username }}</span>
        <select
          class="text-xs rounded bg-bg-elevated border border-border px-2 py-1"
          :value="m.level"
          @change="changeLevel(m, ($event.target as HTMLSelectElement).value)"
        >
          <option value="read">read</option>
          <option value="write">write</option>
        </select>
        <button
          type="button"
          class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
          @click="remove(m)"
        >Remove</button>
      </li>
      <li v-if="!loading && members.length === 0" class="text-sm text-text-muted">
        No members yet.
      </li>
    </ul>

    <form class="flex items-end gap-2" @submit.prevent="add">
      <label class="flex-1 text-sm">
        <span class="text-text-muted text-xs">Add user</span>
        <select
          v-model="addUser"
          class="mt-1 w-full rounded bg-bg-elevated border border-border px-2 py-2"
        >
          <option value="">Select a user…</option>
          <option v-for="u in candidates" :key="u.id" :value="u.id">
            {{ u.username }} ({{ u.role }})
          </option>
        </select>
      </label>
      <label class="text-sm">
        <span class="text-text-muted text-xs">Level</span>
        <select
          v-model="addLevel"
          class="mt-1 rounded bg-bg-elevated border border-border px-2 py-2"
        >
          <option value="read">read</option>
          <option value="write">write</option>
        </select>
      </label>
      <button
        type="submit"
        :disabled="busy || !addUser"
        class="rounded bg-accent text-accent-fg px-3 py-2 text-sm hover:bg-accent-hover disabled:opacity-60"
      >+ Add</button>
    </form>
  </div>
</template>
