<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { getSettings, updateSettings, type Settings } from '@/api/settings'
import { useAuthStore } from '@/stores/auth'
import { useUIStore, type LocaleCode, type ThemePreset } from '@/stores/ui'

const auth = useAuthStore()
const ui = useUIStore()

interface PresetOption { value: ThemePreset; label: string; tone: 'dark' | 'light' }
const presetOptions: PresetOption[] = [
  { value: 'catppuccin-mocha', label: 'Catppuccin Mocha', tone: 'dark' },
  { value: 'tokyo-night', label: 'Tokyo Night', tone: 'dark' },
  { value: 'catppuccin-latte', label: 'Catppuccin Latte', tone: 'light' },
  { value: 'solarized-light', label: 'Solarized Light', tone: 'light' },
  { value: 'custom', label: 'Custom (default = Mocha)', tone: 'dark' },
]
interface LocaleOption { value: LocaleCode; label: string }
const localeOptions: LocaleOption[] = [
  { value: 'it', label: 'Italiano' },
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Español' },
  { value: 'fr', label: 'Français' },
  { value: 'de', label: 'Deutsch' },
]
const data = ref<Settings | null>(null)
const draft = reactive<{
  git: {
    enabled: boolean
    remote: string
    branch: string
    debounce_ms: number
    push: boolean
    token_env: string
  }
  trash: { enabled: boolean; retention_ms: number }
  i18n: { default_lang: string; enabled_langs: string }
}>({
  git: { enabled: false, remote: '', branch: '', debounce_ms: 30000, push: false, token_env: '' },
  trash: { enabled: false, retention_ms: 0 },
  i18n: { default_lang: 'en', enabled_langs: 'it,en' },
})
const loading = ref(false)
const saving = ref(false)
const message = ref<string | null>(null)
const error = ref<string | null>(null)

function hydrate(s: Settings) {
  data.value = s
  draft.git = {
    enabled: s.git.enabled,
    remote: s.git.remote,
    branch: s.git.branch,
    debounce_ms: s.git.debounce_ms,
    push: s.git.push,
    token_env: s.git.token_env,
  }
  draft.trash = { enabled: s.trash.enabled, retention_ms: s.trash.retention_ms }
  draft.i18n = {
    default_lang: s.i18n.default_lang,
    enabled_langs: (s.i18n.enabled_langs ?? []).join(','),
  }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    hydrate(await getSettings())
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load settings'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  error.value = null
  message.value = null
  try {
    const result = await updateSettings({
      git: { ...draft.git },
      trash: { ...draft.trash },
      i18n: {
        default_lang: draft.i18n.default_lang,
        enabled_langs: draft.i18n.enabled_langs
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      },
    })
    hydrate(result)
    message.value = 'Saved.'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Save failed'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="p-8 max-w-3xl mx-auto">
    <h1 class="text-2xl font-semibold mb-1">Settings</h1>
    <p
      v-if="!auth.isOwner"
      class="text-sm text-text-muted mb-6"
    >
      Read-only — only owners can change server settings.
    </p>

    <fieldset class="rounded border border-border bg-surface p-4 space-y-3 mb-6">
      <legend class="px-2 text-sm uppercase tracking-wide text-text-muted">Appearance</legend>
      <label class="block text-sm">
        <span class="text-text-muted">Theme preset</span>
        <select
          :value="ui.preset"
          class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
          @change="ui.setPreset(($event.target as HTMLSelectElement).value as ThemePreset)"
        >
          <option v-for="p in presetOptions" :key="p.value" :value="p.value">
            {{ p.label }} ({{ p.tone }})
          </option>
        </select>
      </label>
      <label class="block text-sm">
        <span class="text-text-muted">Language</span>
        <select
          :value="ui.locale"
          class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
          @change="ui.setLocale(($event.target as HTMLSelectElement).value as LocaleCode)"
        >
          <option v-for="l in localeOptions" :key="l.value" :value="l.value">
            {{ l.label }}
          </option>
        </select>
      </label>
      <p class="text-xs text-text-muted">
        Theme + language are stored in your browser. Server-side `git`/`trash`/`i18n.default_lang`
        below configures the gosidian instance for everyone.
      </p>
    </fieldset>

    <p v-if="loading" class="text-text-muted">Loading…</p>
    <p v-else-if="error" class="text-danger">{{ error }}</p>

    <form
      v-else-if="data"
      class="space-y-8"
      @submit.prevent="save"
    >
      <fieldset class="rounded border border-border bg-surface p-4 space-y-3">
        <legend class="px-2 text-sm uppercase tracking-wide text-text-muted">Git sync</legend>
        <label class="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            :disabled="!auth.isOwner"
            v-model="draft.git.enabled"
          />
          <span>Enabled</span>
        </label>
        <label class="block text-sm">
          <span class="text-text-muted">Remote</span>
          <input
            v-model.trim="draft.git.remote"
            :disabled="!auth.isOwner"
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
          />
        </label>
        <div class="grid grid-cols-2 gap-3">
          <label class="block text-sm">
            <span class="text-text-muted">Branch</span>
            <input
              v-model.trim="draft.git.branch"
              :disabled="!auth.isOwner"
              class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
            />
          </label>
          <label class="block text-sm">
            <span class="text-text-muted">Debounce (ms)</span>
            <input
              v-model.number="draft.git.debounce_ms"
              :disabled="!auth.isOwner"
              type="number"
              min="1000"
              class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
            />
          </label>
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            :disabled="!auth.isOwner"
            v-model="draft.git.push"
          />
          <span>Push to remote on commit</span>
        </label>
        <label class="block text-sm">
          <span class="text-text-muted">Token env var name</span>
          <input
            v-model.trim="draft.git.token_env"
            :disabled="!auth.isOwner"
            placeholder="GITEA_TOKEN"
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2 font-mono"
          />
        </label>
      </fieldset>

      <fieldset class="rounded border border-border bg-surface p-4 space-y-3">
        <legend class="px-2 text-sm uppercase tracking-wide text-text-muted">Trash</legend>
        <label class="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            :disabled="!auth.isOwner"
            v-model="draft.trash.enabled"
          />
          <span>Enabled (soft-delete instead of hard-delete)</span>
        </label>
        <label class="block text-sm">
          <span class="text-text-muted">Retention (ms, 0 = forever)</span>
          <input
            v-model.number="draft.trash.retention_ms"
            :disabled="!auth.isOwner"
            type="number"
            min="0"
            class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
          />
        </label>
      </fieldset>

      <fieldset class="rounded border border-border bg-surface p-4 space-y-3">
        <legend class="px-2 text-sm uppercase tracking-wide text-text-muted">i18n</legend>
        <div class="grid grid-cols-2 gap-3">
          <label class="block text-sm">
            <span class="text-text-muted">Default language</span>
            <input
              v-model.trim="draft.i18n.default_lang"
              :disabled="!auth.isOwner"
              class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
            />
          </label>
          <label class="block text-sm">
            <span class="text-text-muted">Enabled (comma-separated)</span>
            <input
              v-model.trim="draft.i18n.enabled_langs"
              :disabled="!auth.isOwner"
              class="mt-1 w-full rounded bg-bg-elevated border border-border px-3 py-2"
            />
          </label>
        </div>
      </fieldset>

      <div class="flex items-center gap-3">
        <button
          type="submit"
          :disabled="!auth.isOwner || saving"
          class="px-4 py-2 rounded bg-accent text-accent-fg hover:bg-accent-hover disabled:opacity-50"
        >{{ saving ? 'Saving…' : 'Save' }}</button>
        <p v-if="message" class="text-sm text-success">{{ message }}</p>
      </div>
    </form>
  </div>
</template>
