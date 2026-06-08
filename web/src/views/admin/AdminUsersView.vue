<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listUsers, disableUser, updateUserRole, updateUserTOTPPolicy, type AdminUser } from '@/api/admin'

const users = ref<AdminUser[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    users.value = await listUsers()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function disable(u: AdminUser) {
  if (!confirm(`Disable user "${u.username}"? They will be logged out and MCP tokens revoked.`)) return
  try {
    await disableUser(u.id)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Disable failed'
  }
}

async function changeRole(u: AdminUser, role: string) {
  if ((role !== 'member' && role !== 'guest') || role === u.role) return
  try {
    await updateUserRole(u.id, role)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Role change failed'
  }
}

async function changeTotpPolicy(u: AdminUser, policy: string) {
  try {
    await updateUserTOTPPolicy(u.id, policy)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'TOTP policy change failed'
  }
}

onMounted(load)
</script>

<template>
  <p v-if="loading" class="text-text-muted">Loading…</p>
  <p v-else-if="error" class="text-danger">{{ error }}</p>

  <table v-else class="w-full text-sm">
    <thead class="text-text-muted text-xs uppercase tracking-wide">
      <tr>
        <th class="text-left py-2 px-3">Username</th>
        <th class="text-left py-2 px-3">Role</th>
        <th class="text-left py-2 px-3">TOTP</th>
        <th class="text-left py-2 px-3">Created</th>
        <th class="text-left py-2 px-3">Status</th>
        <th class="text-right py-2 px-3">Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="u in users"
        :key="u.id"
        class="border-t border-border"
      >
        <td class="py-2 px-3 font-medium">{{ u.username }}</td>
        <td class="py-2 px-3">
          <span
            v-if="u.role === 'owner' || u.disabled_at"
            class="text-xs px-2 py-0.5 rounded"
            :class="u.role === 'owner' ? 'bg-accent/20 text-accent' : 'border border-border'"
          >{{ u.role }}</span>
          <select
            v-else
            class="text-xs rounded bg-bg-elevated border border-border px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
            :value="u.role"
            @change="changeRole(u, ($event.target as HTMLSelectElement).value)"
          >
            <option value="member">member</option>
            <option value="guest">guest</option>
          </select>
        </td>
        <td class="py-2 px-3">
          <div class="flex items-center gap-2">
            <select
              v-if="u.role !== 'owner' && !u.disabled_at"
              class="text-xs rounded bg-bg-elevated border border-border px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
              :value="u.totp_policy || ''"
              @change="changeTotpPolicy(u, ($event.target as HTMLSelectElement).value)"
            >
              <option value="">inherit</option>
              <option value="enabled">required</option>
              <option value="disabled">exempt</option>
            </select>
            <span v-else class="text-xs text-text-muted">{{ u.totp_policy || 'inherit' }}</span>
            <span v-if="u.totp_enrolled" class="text-xs text-success" title="TOTP enrolled">●</span>
          </div>
        </td>
        <td class="py-2 px-3 font-mono text-xs">{{ u.created_at }}</td>
        <td class="py-2 px-3">
          <span
            v-if="u.disabled_at"
            class="text-xs text-warning"
          >disabled {{ u.disabled_at }}</span>
          <span v-else class="text-xs text-success">active</span>
        </td>
        <td class="py-2 px-3 text-right">
          <button
            v-if="!u.disabled_at && u.role !== 'owner'"
            type="button"
            class="text-xs px-2 py-1 rounded text-danger hover:bg-surface-hover"
            @click="disable(u)"
          >Disable</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
