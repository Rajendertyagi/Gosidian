<script setup lang="ts">
import TopBar from './TopBar.vue'
import Sidebar from './Sidebar.vue'
import CommandPalette from './CommandPalette.vue'
import ConflictDialog from '@/components/domain/ConflictDialog.vue'
import { useSidebarResize } from '@/composables/useSidebarResize'

const { width, startDrag, reset } = useSidebarResize()
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

      <main class="flex-1 overflow-auto">
        <RouterView />
      </main>
    </div>

    <ConflictDialog />
    <CommandPalette />
  </div>
</template>
