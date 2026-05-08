<script setup lang="ts">
/**
 * ConflictDialog — surfaces the 412 etag mismatch the API client
 * emits on `note.concurrency-conflict`. Three action paths the user
 * can pick from:
 *
 *   - "Reload": discard local working state and load the remote.
 *   - "Force overwrite": PUT without If-Match (Phase 3 editor wires
 *     this up; today the dialog only emits an event the parent
 *     handles).
 *   - "View diff": placeholder — the proper diff view is Phase 2.1+.
 *     For now we show the remote excerpt next to a brief "your draft
 *     is preserved in the editor" reminder so the user knows their
 *     work isn't lost.
 *
 * Phase 2bis ships the dialog skeleton; the editor that opens it on
 * the conflict event lives in Phase 4 (CodeMirror integration).
 */
import { ref, onMounted, onUnmounted } from 'vue'
import { onApiEvent, type ConcurrencyConflictDetail } from '@/api/client'

const visible = ref(false)
const conflict = ref<ConcurrencyConflictDetail | null>(null)

const emit = defineEmits<{
  reload: [path: string]
  forceOverwrite: [path: string]
  dismiss: []
}>()

let unsubscribe: (() => void) | null = null

onMounted(() => {
  unsubscribe = onApiEvent('note.concurrency-conflict', (detail) => {
    conflict.value = detail
    visible.value = true
  })
})

onUnmounted(() => {
  unsubscribe?.()
})

function handleReload() {
  if (conflict.value) emit('reload', conflict.value.path)
  visible.value = false
}
function handleForce() {
  if (conflict.value) emit('forceOverwrite', conflict.value.path)
  visible.value = false
}
function handleDismiss() {
  emit('dismiss')
  visible.value = false
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible && conflict"
      class="fixed inset-0 z-50 flex items-center justify-center bg-overlay/70"
      role="alertdialog"
      aria-modal="true"
      :aria-label="'Conflict on ' + conflict.path"
    >
      <div
        class="w-[min(520px,90vw)] rounded-lg bg-surface p-6 shadow-lg ring-1 ring-border"
      >
        <h2 class="text-lg font-semibold">
          Note modified externally
        </h2>
        <p class="mt-2 text-sm text-text-muted">
          <code class="font-mono">{{ conflict.path }}</code> has changed since
          you opened it. Choose how to resolve the conflict; your draft is
          preserved in the editor either way.
        </p>

        <pre
          class="mt-4 max-h-40 overflow-auto rounded border border-border bg-bg-elevated p-2 text-xs text-text-muted whitespace-pre-wrap"
        >{{ conflict.current_content_excerpt }}</pre>

        <div class="mt-5 flex flex-wrap gap-2 justify-end">
          <button
            type="button"
            class="px-3 py-1.5 rounded text-sm hover:bg-surface-hover"
            @click="handleDismiss"
          >
            Keep editing
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded text-sm bg-surface-hover hover:bg-border"
            @click="handleReload"
          >
            Reload remote
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded text-sm bg-accent text-accent-fg hover:bg-accent-hover"
            @click="handleForce"
          >
            Overwrite
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
