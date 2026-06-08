<script setup lang="ts">
/**
 * MarkdownPreview — renders the HTML produced by /api/v1/preview inside a
 * sanitized v-html container. Server-side goldmark already escapes raw HTML;
 * DOMPurify is defense-in-depth (strips on*= attrs, javascript: URLs, unknown
 * tags). DOMPurify keeps data-* attributes by default, so the renderer's
 * `data-preview-path` on resolved wikilinks survives.
 *
 * Link interception (plancia): internal links open as windows instead of
 * navigating away from the SPA —
 *   - resolved wikilink (`data-preview-path`) → note window
 *   - tag link (`/tags/<tag>`)               → tags window
 *   - in-note anchor (`#heading`)            → scroll within this preview
 *   - external link                          → open in a new tab
 * Modified clicks (ctrl/cmd/middle) fall through to the browser (new tab on
 * the canonical deep-link URL).
 */
import { computed, inject, ref } from 'vue'
import DOMPurify from 'dompurify'
import { useWindowsStore, type OpenSpec } from '@/stores/windows'
import { planciaKey } from '@/composables/usePlanciaSync'

const props = defineProps<{ html: string }>()

const store = useWindowsStore()
const openWindow = inject<(spec: OpenSpec) => string>('openWindow', (s) => store.open(s))
const root = ref<HTMLElement | null>(null)

const sanitized = computed(() =>
  DOMPurify.sanitize(props.html, {
    ADD_TAGS: ['math', 'mfrac', 'mrow', 'msup', 'mn', 'mi'],
    ADD_ATTR: ['class', 'data-preview-path'],
  }),
)

function onClick(e: MouseEvent) {
  if (e.defaultPrevented || e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return
  const a = (e.target as HTMLElement | null)?.closest('a')
  if (!a) return

  const previewPath = a.getAttribute('data-preview-path')
  const href = a.getAttribute('href') ?? ''

  if (previewPath) {
    e.preventDefault()
    openWindow({
      type: 'note',
      key: planciaKey('note', previewPath),
      title: (previewPath.split('/').pop() ?? previewPath).replace(/\.md$/, ''),
      props: { path: previewPath },
    })
    return
  }
  if (href.startsWith('/tags/')) {
    e.preventDefault()
    const tag = decodeURIComponent(href.slice('/tags/'.length).split('#')[0] ?? '')
    if (tag) openWindow({ type: 'tags', key: planciaKey('tags', tag), title: `#${tag}`, props: { tag } })
    return
  }
  if (href.startsWith('#')) {
    e.preventDefault()
    const id = href.slice(1)
    root.value?.querySelector(`#${CSS.escape(id)}`)?.scrollIntoView({ behavior: 'smooth' })
    return
  }
  if (href.startsWith('/notes/new')) {
    // Unresolved wikilink → nothing to open yet; swallow to stay in the SPA.
    e.preventDefault()
    return
  }
  if (/^https?:\/\//i.test(href)) {
    e.preventDefault()
    window.open(href, '_blank', 'noopener,noreferrer')
  }
}
</script>

<template>
  <div
    ref="root"
    class="prose prose-invert max-w-none prose-pre:bg-bg-elevated prose-pre:border prose-pre:border-border prose-code:before:hidden prose-code:after:hidden"
    v-html="sanitized"
    @click="onClick"
  />
</template>
