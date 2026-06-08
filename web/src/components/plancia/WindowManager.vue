<script setup lang="ts">
/**
 * WindowManager — the "plancia": a niri-style horizontal strip of tiled
 * windows with scroll-snap, plus a horizontally-scrollable footer of minimized
 * windows. Generic: the type→component `registry` decides what each window
 * renders. Ported from products-dc and adapted to gosidian tokens + i18n.
 */
import { type Component, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import WindowFrame from './WindowFrame.vue'
import { useWindowsStore, type OpenSpec, type WindowInstance } from '@/stores/windows'

const props = defineProps<{
  /** Window-type → content component map. Makes the manager generic. */
  registry: Record<string, Component>
  /** "Owner" type of this plancia. Windows of a different type are cross-module
   *  and get a "[module]: title" header prefix. Omitted ⇒ no foreign windows. */
  nativeType?: string
}>()

const store = useWindowsStore()
const { t } = useI18n()

function foreignLabel(w: WindowInstance): string | null {
  if (!props.nativeType || w.type === props.nativeType) return null
  return w.type
}
const stripEl = ref<HTMLElement | null>(null)

// Nested content (e.g. a note body) opens sibling windows without prop-drilling.
provide<(spec: OpenSpec) => string>('openWindow', (spec) => store.open(spec))
// Linkable-tag predicate: a tag is clickable only if its window-type is
// registered in this plancia (otherwise it stays informational).
provide<(type: string) => boolean>('canOpenWindowType', (type) => !!props.registry[type])

// Bring the focused window fully into view (horizontal scroll).
watch(
  () => store.focusedId,
  (id) => {
    if (!id) return
    nextTick(() => {
      stripEl.value
        ?.querySelector<HTMLElement>(`[data-win-id="${id}"]`)
        ?.scrollIntoView({ behavior: 'smooth', inline: 'nearest', block: 'nearest' })
    })
  },
)

function onKey(e: KeyboardEvent) {
  if (!e.altKey) return
  if (e.key === 'ArrowLeft') {
    e.preventDefault()
    store.focusAdjacent(-1)
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    store.focusAdjacent(1)
  }
}
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="flex h-full flex-col bg-bg">
    <div
      ref="stripEl"
      class="flex min-h-0 flex-1 snap-x snap-proximity flex-row gap-3 overflow-x-auto p-3"
    >
      <WindowFrame
        v-for="w in store.visible"
        :key="w.id"
        :win="w"
        :foreign-label="foreignLabel(w)"
      >
        <component
          :is="registry[w.type]"
          v-if="registry[w.type]"
          v-bind="w.props"
          @title="store.setTitle(w.id, $event)"
          @dirty="store.setDirty(w.id, $event)"
          @identify="store.identify(w.id, $event.key, $event.props)"
          @changed="store.touch()"
          @tags="store.setTags(w.id, $event)"
          @close="store.close(w.id)"
        />
        <div v-else class="p-4 text-sm text-danger">
          {{ t('plancia.unknownType', { type: w.type }) }}
        </div>
      </WindowFrame>

      <div v-if="!store.visible.length" class="grid w-full place-items-center text-text-muted">
        <div class="text-center">
          <p class="text-sm">{{ t('plancia.empty') }}</p>
          <p v-if="store.minimizedList.length" class="mt-1 text-xs">
            {{ t('plancia.minimizedHint', { n: store.minimizedList.length }) }}
          </p>
          <p v-else class="mt-1 text-xs">{{ t('plancia.openFromSidebar') }}</p>
        </div>
      </div>
    </div>

    <footer
      v-if="store.minimizedList.length"
      class="flex shrink-0 flex-nowrap items-center gap-2 overflow-x-auto border-t border-border bg-bg-elevated/60 px-3 py-2"
    >
      <span class="shrink-0 text-xs font-medium text-text-muted">{{ t('plancia.minimizedLabel') }}</span>
      <span
        v-for="w in store.minimizedList"
        :key="w.id"
        class="inline-flex shrink-0 items-center gap-1.5 rounded border border-border bg-surface px-2 py-1 text-xs"
      >
        <button
          type="button"
          class="max-w-[14rem] cursor-pointer truncate hover:text-accent"
          :title="t('plancia.restore')"
          @click="store.restore(w.id)"
        >
          {{ w.title || '—' }}
        </button>
        <button
          type="button"
          class="text-text-muted hover:text-danger"
          :title="t('plancia.close')"
          :aria-label="t('plancia.close')"
          @click="store.close(w.id)"
        >
          ✕
        </button>
      </span>
    </footer>
  </div>
</template>
