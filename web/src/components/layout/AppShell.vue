<script setup lang="ts">
import { onMounted } from 'vue'
import TopBar from './TopBar.vue'
import Sidebar from './Sidebar.vue'
import CommandPalette from './CommandPalette.vue'
import TotpEnroll from '@/components/domain/TotpEnroll.vue'
import WindowManager from '@/components/plancia/WindowManager.vue'
import { windowRegistry } from '@/components/plancia/windowRegistry'
import { usePlanciaSync } from '@/composables/usePlanciaSync'
import { useSidebarResize } from '@/composables/useSidebarResize'
import { useAuthStore } from '@/stores/auth'

const { width, startDrag, reset } = useSidebarResize()
const auth = useAuthStore()
const plancia = usePlanciaSync()

onMounted(() => plancia.hydrate())

function onEnrolled() {
  auth.setEnrolled(true)
  auth.clearEnrollment()
}
</script>

<template>
  <div class="h-screen flex flex-col bg-bg text-text">
    <TopBar />
    <div class="flex-1 flex overflow-hidden min-h-0">
      <div
        class="border-r border-border flex-shrink-0"
        :style="{ width: width + 'px' }"
      >
        <Sidebar />
      </div>

      <div
        class="w-1 cursor-col-resize bg-border hover:bg-accent/40 select-none flex-shrink-0"
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize sidebar"
        @pointerdown="startDrag"
        @dblclick="reset"
      />

      <main class="flex-1 min-w-0 overflow-hidden">
        <WindowManager :registry="windowRegistry" />
      </main>
    </div>

    <!-- Forced TOTP enrolment interstitial: blocks the app when the user's
         effective policy requires two-factor but no secret is enrolled. -->
    <div
      v-if="auth.enrollmentRequired"
      class="fixed inset-0 z-50 flex items-center justify-center bg-bg/95 p-4"
    >
      <div class="w-full max-w-md rounded-lg bg-surface p-6 ring-1 ring-border shadow">
        <h2 class="text-lg font-semibold mb-1">Two-factor required</h2>
        <p class="text-sm text-text-muted mb-4">
          Your account requires two-factor authentication. Set it up to continue.
        </p>
        <TotpEnroll @done="onEnrolled" />
      </div>
    </div>

    <CommandPalette />
  </div>
</template>
