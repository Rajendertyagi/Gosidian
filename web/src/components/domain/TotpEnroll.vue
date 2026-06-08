<script setup lang="ts">
import { ref } from 'vue'
import { enrollTOTP, confirmTOTP } from '@/api/totp'

const emit = defineEmits<{ done: [] }>()

const secret = ref('')
const uri = ref('')
const code = ref('')
const error = ref<string | null>(null)
const busy = ref(false)

async function start() {
  busy.value = true
  error.value = null
  try {
    const d = await enrollTOTP()
    secret.value = d.secret
    uri.value = d.otpauth_uri
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to start enrolment'
  } finally {
    busy.value = false
  }
}

async function confirm() {
  if (!code.value.trim()) return
  busy.value = true
  error.value = null
  try {
    await confirmTOTP(secret.value, code.value.trim())
    emit('done')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Invalid code'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="space-y-3">
    <button
      v-if="!secret"
      type="button"
      :disabled="busy"
      class="rounded bg-accent text-accent-fg px-3 py-2 text-sm hover:bg-accent-hover disabled:opacity-60"
      @click="start"
    >
      Set up two-factor (TOTP)
    </button>

    <div v-else class="space-y-3">
      <p class="text-sm text-text-muted">
        Scan the link below in your authenticator app, or enter the secret manually, then
        confirm a generated code to enable two-factor authentication.
      </p>
      <a :href="uri" class="block break-all text-xs text-accent font-mono">{{ uri }}</a>
      <p class="text-xs text-text-muted">
        Secret: <code class="font-mono text-text">{{ secret }}</code>
      </p>
      <input
        v-model.trim="code"
        inputmode="numeric"
        autocomplete="one-time-code"
        placeholder="123 456"
        class="w-full rounded bg-bg-elevated border border-border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-accent"
      />
      <button
        type="button"
        :disabled="busy"
        class="rounded bg-accent text-accent-fg px-3 py-2 text-sm hover:bg-accent-hover disabled:opacity-60"
        @click="confirm"
      >
        Confirm &amp; enable
      </button>
    </div>

    <p v-if="error" class="text-sm text-danger">{{ error }}</p>
  </div>
</template>
