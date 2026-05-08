<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listSpaTokens, revokeSpaToken, type SpaToken } from '@/api/admin'

const tokens = ref<SpaToken[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    tokens.value = await listSpaTokens()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function revoke(t: SpaToken) {
  if (!confirm(`Revoke session ${t.id.slice(0, 8)}…? The user will be logged out.`)) return
  try {
    await revokeSpaToken(t.id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Revoke failed'
  }
}

onMounted(load)
</script>

<template>
  <p v-if="loading" class="text-text-muted">Loading…</p>
  <p v-else-if="error" class="text-danger">{{ error }}</p>

  <p v-else-if="!tokens.length" class="text-text-muted text-sm">No active SPA sessions.</p>

  <table v-else class="w-full text-sm">
    <thead class="text-text-muted text-xs uppercase tracking-wide">
      <tr>
        <th class="text-left py-2 px-3">User</th>
        <th class="text-left py-2 px-3">User-Agent</th>
        <th class="text-left py-2 px-3">Issued</th>
        <th class="text-left py-2 px-3">Last seen</th>
        <th class="text-left py-2 px-3">Expires</th>
        <th class="text-right py-2 px-3">Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="t in tokens"
        :key="t.id"
        class="border-t border-border"
      >
        <td class="py-2 px-3 font-mono text-xs">{{ t.user_id }}</td>
        <td class="py-2 px-3 text-xs truncate max-w-[16rem]" :title="t.user_agent">
          {{ t.user_agent || '—' }}
        </td>
        <td class="py-2 px-3 font-mono text-xs">{{ t.issued_at }}</td>
        <td class="py-2 px-3 font-mono text-xs">{{ t.last_seen_at }}</td>
        <td class="py-2 px-3 font-mono text-xs">{{ t.expires_at }}</td>
        <td class="py-2 px-3 text-right">
          <button
            type="button"
            class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
            @click="revoke(t)"
          >Revoke</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
