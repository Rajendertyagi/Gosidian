<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTreeStore } from '@/stores/tree'
import { useRecentlyViewed } from '@/composables/useRecentlyViewed'
import { useSSE } from '@/composables/useSSE'
import TreeNode from '@/components/domain/TreeNode.vue'
import { RefreshCw } from 'lucide-vue-next'

const { t } = useI18n()

const treeStore = useTreeStore()
const recents = useRecentlyViewed()
const sse = useSSE(['tree'])

const root = computed(() => treeStore.byProject[''])
const loading = computed(() => Boolean(treeStore.loading['']))
const error = computed(() => treeStore.error[''])

onMounted(() => {
  void treeStore.load()

  // Real-time invalidation: any tree-topic event from the SSE hub
  // means the sidebar might be stale. The simplest correct response
  // is "drop the cache and refetch" — debounce isn't required at
  // current vault sizes.
  sse.on('tree', () => {
    treeStore.invalidate()
    void treeStore.load()
  })
})
</script>

<template>
  <aside class="h-full flex flex-col bg-bg overflow-hidden">
    <div class="px-3 py-2 border-b border-border flex items-center justify-between">
      <span class="text-xs font-semibold uppercase tracking-wide text-text-muted">
        {{ t('common.vault') }}
      </span>
      <button
        type="button"
        class="text-text-muted hover:text-text disabled:opacity-50"
        :disabled="loading"
        @click="treeStore.load()"
        :aria-label="'Refresh tree'"
        :title="'Refresh tree'"
      >
        <RefreshCw class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <div class="flex-1 overflow-auto px-2 py-2">
      <p v-if="loading && !root" class="text-xs text-text-muted px-1">{{ t('common.loading') }}</p>
      <p v-else-if="error" class="text-xs text-danger px-1">{{ error }}</p>
      <ul v-else-if="root && root.children?.length">
        <TreeNode
          v-for="child in root.children"
          :key="child.path"
          :node="child"
        />
      </ul>
      <p v-else class="text-xs text-text-muted px-1">No notes yet.</p>
    </div>

    <div
      v-if="recents.entries.value.length"
      class="border-t border-border px-3 py-2"
    >
      <p class="text-xs font-semibold uppercase tracking-wide text-text-muted mb-1">
        Recent
      </p>
      <ul class="space-y-0.5 text-sm">
        <li
          v-for="entry in recents.entries.value.slice(0, 5)"
          :key="entry.path"
        >
          <RouterLink
            :to="'/notes/' + encodeURIComponent(entry.path)"
            class="block truncate text-text-muted hover:text-text"
          >{{ entry.title }}</RouterLink>
        </li>
      </ul>
    </div>
  </aside>
</template>
