<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const totp = ref('')
const error = ref<string | null>(null)
const submitting = ref(false)

const nextTarget = computed(() => {
  const raw = route.query.next
  if (typeof raw === 'string' && raw.startsWith('/')) return raw
  return '/'
})

async function handleSubmit() {
  if (submitting.value) return
  submitting.value = true
  error.value = null
  try {
    await auth.login(username.value, password.value, totp.value || undefined)
    await router.push(nextTarget.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-bg text-text px-4">
    <div class="w-full max-w-sm">
      <div class="mb-6 text-center">
        <h1 class="text-2xl font-semibold">gosidian</h1>
        <p class="text-sm text-text-muted">Sign in to continue</p>
      </div>

      <form
        class="space-y-3 rounded-lg bg-surface p-5 shadow ring-1 ring-border"
        @submit.prevent="handleSubmit"
      >
        <label class="block text-sm">
          <span class="text-text-muted">Username</span>
          <input
            v-model.trim="username"
            type="text"
            autocomplete="username"
            required
            autofocus
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
          />
        </label>

        <label class="block text-sm">
          <span class="text-text-muted">Password</span>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
          />
        </label>

        <label class="block text-sm">
          <span class="text-text-muted">TOTP <span class="opacity-60">(optional)</span></span>
          <input
            v-model.trim="totp"
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="123 456"
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
          />
        </label>

        <p v-if="error" class="text-sm text-danger">{{ error }}</p>

        <button
          type="submit"
          :disabled="submitting"
          class="w-full rounded bg-accent text-accent-fg py-2 font-medium hover:bg-accent-hover disabled:opacity-60"
        >
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>
