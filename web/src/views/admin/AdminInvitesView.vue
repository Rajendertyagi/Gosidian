<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listInvites, createInvite, deleteInvite, type Invite } from '@/api/admin'

const invites = ref<Invite[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const fresh = ref<Invite | null>(null)
const ttlHours = ref(24)

async function load() {
  loading.value = true
  error.value = null
  try {
    invites.value = await listInvites()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function create() {
  try {
    fresh.value = await createInvite(ttlHours.value > 0 ? ttlHours.value * 3600 * 1000 : undefined)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Create failed'
  }
}

async function destroy(iv: Invite) {
  if (!confirm(`Revoke invite ${iv.token.slice(0, 8)}…?`)) return
  try {
    await deleteInvite(iv.token)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  }
}

function dismissFresh() {
  fresh.value = null
}

const signupURL = (token: string) => `${window.location.origin}/login?invite=${encodeURIComponent(token)}`

onMounted(load)
</script>

<template>
  <div v-if="fresh" class="rounded border border-success bg-success/10 p-4 mb-6 space-y-2">
    <p class="text-sm font-semibold text-success">Invite token created.</p>
    <code class="block bg-bg-elevated rounded px-3 py-2 font-mono text-sm break-all select-all">{{ fresh.token }}</code>
    <p class="text-xs text-text-muted">Share the signup link with the new member:</p>
    <code class="block bg-bg-elevated rounded px-3 py-2 font-mono text-xs break-all select-all">{{ signupURL(fresh.token) }}</code>
    <p class="text-xs text-text-muted">Expires {{ fresh.expires_at }}.</p>
    <button
      type="button"
      class="text-xs px-2 py-1 rounded border border-border hover:bg-surface-hover"
      @click="dismissFresh"
    >Dismiss</button>
  </div>

  <form
    class="flex items-end gap-3 mb-6"
    @submit.prevent="create"
  >
    <label class="text-sm">
      <span class="block text-text-muted text-xs mb-1">TTL (hours)</span>
      <input
        v-model.number="ttlHours"
        type="number"
        min="1"
        class="rounded bg-bg-elevated border border-border px-3 py-2 w-32"
      />
    </label>
    <button
      type="submit"
      class="px-3 py-2 rounded bg-accent text-accent-fg hover:bg-accent-hover"
    >Create invite</button>
  </form>

  <p v-if="loading" class="text-text-muted">Loading…</p>
  <p v-else-if="error" class="text-danger">{{ error }}</p>

  <p v-else-if="!invites.length" class="text-text-muted text-sm">No invites in flight.</p>

  <table v-else class="w-full text-sm">
    <thead class="text-text-muted text-xs uppercase tracking-wide">
      <tr>
        <th class="text-left py-2 px-3">Token</th>
        <th class="text-left py-2 px-3">Created by</th>
        <th class="text-left py-2 px-3">Created</th>
        <th class="text-left py-2 px-3">Expires</th>
        <th class="text-left py-2 px-3">Status</th>
        <th class="text-right py-2 px-3">Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="iv in invites"
        :key="iv.token"
        class="border-t border-border"
      >
        <td class="py-2 px-3 font-mono text-xs">{{ iv.token.slice(0, 16) }}…</td>
        <td class="py-2 px-3 font-mono text-xs">{{ iv.created_by }}</td>
        <td class="py-2 px-3 font-mono text-xs">{{ iv.created_at }}</td>
        <td class="py-2 px-3 font-mono text-xs">{{ iv.expires_at }}</td>
        <td class="py-2 px-3">
          <span
            v-if="iv.consumed_at"
            class="text-xs text-text-muted"
          >consumed by {{ iv.consumed_by }}</span>
          <span
            v-else-if="iv.pending"
            class="text-xs text-success"
          >pending</span>
          <span
            v-else
            class="text-xs text-warning"
          >expired</span>
        </td>
        <td class="py-2 px-3 text-right">
          <button
            v-if="iv.pending"
            type="button"
            class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
            @click="destroy(iv)"
          >Revoke</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
