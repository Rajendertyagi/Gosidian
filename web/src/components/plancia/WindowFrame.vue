<script setup lang="ts">
/**
 * WindowFrame — chrome around one plancia window. Content-agnostic: the body
 * is a <slot/>. Ported from products-dc; adapted to gosidian's semantic tokens
 * and lucide icons, with a 4th "direct links" button that opens an ego-graph
 * of the window's note (req #7).
 */
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Maximize2, Minus, X, Network } from 'lucide-vue-next'
import {
  useWindowsStore,
  type OpenSpec,
  type WindowInstance,
  type WindowTag,
} from '@/stores/windows'
import { windowTone } from './windowModules'

const props = defineProps<{
  win: WindowInstance
  /** Module label when the window is cross-module (from WindowManager): the
   *  header shows "[module]: title". Null/absent for native windows. */
  foreignLabel?: string | null
}>()
const store = useWindowsStore()
const { t } = useI18n()

// Provided by WindowManager: cross-window open + tag-linkability predicate.
const openWindow = inject<(spec: OpenSpec) => string>('openWindow', () => '')
const canOpenWindowType = inject<(type: string) => boolean>('canOpenWindowType', () => false)

const focused = computed(() => store.focusedId === props.win.id)
const titleAttr = computed(() =>
  props.foreignLabel ? `${props.foreignLabel}: ${props.win.title}` : props.win.title,
)

/** Note path carried by note-shaped windows; gates the "direct links" button. */
const notePath = computed(() =>
  typeof props.win.props?.path === 'string' ? (props.win.props.path as string) : null,
)

function tagClickable(tag: WindowTag): boolean {
  return !!tag.open && canOpenWindowType(tag.open.type)
}
function onTagClick(tag: WindowTag): void {
  if (tagClickable(tag)) openWindow(tag.open!)
}

const widthClass = computed(() => {
  switch (props.win.width) {
    case 's':
      return 'w-[32rem]'
    case 'full':
      return 'w-full'
    default:
      return 'w-[44rem]'
  }
})

/** Open the ego-graph (direct links, depth 1) of this window's note. */
function openLinks(): void {
  const path = notePath.value
  if (!path) return
  openWindow({
    type: 'graph',
    key: `graph:${path}`,
    title: `↳ ${props.win.title || path}`,
    props: { focus: path, depth: 1 },
  })
}

function onClose() {
  if (props.win.dirty && !window.confirm(t('plancia.unsavedClose'))) return
  store.close(props.win.id)
}
</script>

<template>
  <section
    :data-win-id="win.id"
    class="flex h-full shrink-0 snap-start flex-col overflow-hidden rounded-lg border bg-surface shadow-sm transition-shadow max-md:!w-[calc(100vw-3rem)]"
    :class="[widthClass, focused ? 'border-accent ring-2 ring-accent/30' : 'border-border']"
    @mousedown="store.focus(win.id)"
  >
    <header
      class="flex shrink-0 flex-col gap-1 border-b border-border px-3 py-2"
      :class="focused ? 'bg-accent/10' : 'bg-bg-elevated/60'"
    >
      <div class="flex items-center gap-2">
        <span class="flex-1 truncate text-sm font-semibold text-text" :title="titleAttr">
          <span
            v-if="foreignLabel"
            class="mr-1 rounded px-1 align-middle text-[0.62rem] uppercase tracking-wide bg-surface-hover text-text-muted"
            >{{ foreignLabel }}</span
          >
          {{ win.title || '—' }}
          <span v-if="win.dirty" class="text-warning" :title="t('plancia.dirty')">•</span>
        </span>
        <button
          v-if="notePath"
          type="button"
          class="rounded p-1 text-text-muted hover:bg-surface-hover hover:text-text"
          :title="t('plancia.links')"
          :aria-label="t('plancia.links')"
          @click="openLinks"
        >
          <Network class="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          class="rounded p-1 text-text-muted hover:bg-surface-hover hover:text-text"
          :title="`${t('plancia.resize')} (${win.width.toUpperCase()})`"
          :aria-label="t('plancia.resize')"
          @click="store.cycleWidth(win.id)"
        >
          <Maximize2 class="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          class="rounded p-1 text-text-muted hover:bg-surface-hover hover:text-text"
          :title="t('plancia.minimize')"
          :aria-label="t('plancia.minimize')"
          @click="store.minimize(win.id)"
        >
          <Minus class="h-3.5 w-3.5" />
        </button>
        <button
          type="button"
          class="rounded p-1 text-text-muted hover:bg-danger/15 hover:text-danger"
          :title="t('plancia.close')"
          :aria-label="t('plancia.close')"
          @click="onClose"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>

      <!-- Header tags: window associations, some clickable. -->
      <div v-if="win.tags.length" class="flex flex-wrap items-center gap-1">
        <component
          :is="tagClickable(tag) ? 'button' : 'span'"
          v-for="(tag, i) in win.tags"
          :key="i"
          :type="tagClickable(tag) ? 'button' : undefined"
          class="rounded px-1 align-middle text-[0.66rem]"
          :class="[
            windowTone(tag.tone),
            tagClickable(tag) ? 'cursor-pointer underline-offset-2 hover:underline' : '',
          ]"
          :title="tagClickable(tag) ? `${t('plancia.openTag')}: ${tag.label}` : tag.label"
          @click="onTagClick(tag)"
          >{{ tag.label }}</component
        >
      </div>
    </header>
    <div class="min-h-0 flex-1 overflow-y-auto">
      <slot />
    </div>
  </section>
</template>
