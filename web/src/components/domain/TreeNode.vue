<script setup lang="ts">
/**
 * TreeNode — recursive component that renders one apiTreeNode shape
 * (see internal/api/v1/tree.go). Folders use <details> for native
 * keyboard accessibility + open/close persistence in localStorage
 * across reloads (mirrors the v1.x sidebar-tree.js behaviour).
 *
 * Phase 3.1 ships the read-only renderer; Phase 3.2 adds the
 * download icon next to the note count, in-progress badge, and the
 * filter input from the v1.x sidebar.
 */
import type { TreeNode as TN } from '@/api/tree'
import { computed } from 'vue'
import { useRecentlyViewed } from '@/composables/useRecentlyViewed'
import { useWindowsStore } from '@/stores/windows'
import { planciaKey } from '@/composables/usePlanciaSync'

const props = defineProps<{ node: TN }>()
const recents = useRecentlyViewed()
const windows = useWindowsStore()

const expandedKey = computed(() => `gosidian.tree.open:${props.node.path}`)

function toggleExpanded(open: boolean) {
  try {
    localStorage.setItem(expandedKey.value, open ? '1' : '0')
  } catch {
    // ignore
  }
}
function persistedOpen(): boolean {
  try {
    return localStorage.getItem(expandedKey.value) === '1'
  } catch {
    return false
  }
}

function openNote() {
  if (props.node.is_dir) return
  const title = props.node.name.replace(/\.md$/, '')
  recents.record(props.node.path, title)
  windows.open({
    type: 'note',
    key: planciaKey('note', props.node.path),
    title,
    props: { path: props.node.path },
  })
}
</script>

<template>
  <li class="text-sm">
    <template v-if="node.is_dir">
      <details
        :open="persistedOpen()"
        @toggle="toggleExpanded(($event.target as HTMLDetailsElement).open)"
        class="group"
      >
        <summary
          class="flex items-center gap-1.5 py-0.5 px-1 rounded cursor-pointer hover:bg-surface-hover select-none"
        >
          <span class="opacity-60 group-open:rotate-90 transition-transform">▸</span>
          <span class="flex-1 truncate">{{ node.name }}</span>
          <span
            v-if="node.note_count"
            class="text-xs text-text-muted px-1"
          >{{ node.note_count }}</span>
          <span
            v-if="node.hidden_from_mcp"
            class="text-[10px] uppercase text-warning"
            title="Hidden from MCP"
          >hidden</span>
        </summary>
        <ul class="pl-4 border-l border-border ml-1.5">
          <TreeNode
            v-for="child in node.children ?? []"
            :key="child.path"
            :node="child"
          />
        </ul>
      </details>
    </template>

    <template v-else>
      <button
        type="button"
        @click="openNote"
        class="w-full flex items-center gap-1.5 py-0.5 px-2 rounded text-left text-text-muted hover:text-text hover:bg-surface-hover truncate"
      >
        <span class="opacity-50 text-xs">·</span>
        <span class="truncate">{{ node.name.replace(/\.md$/, '') }}</span>
        <span
          v-if="node.in_progress"
          class="text-[10px] text-info ml-auto"
          title="status:in-progress"
        >●</span>
      </button>
    </template>
  </li>
</template>
