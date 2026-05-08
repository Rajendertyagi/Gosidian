<script setup lang="ts">
/**
 * MarkdownPreview — renders the HTML produced by /api/v1/preview
 * inside a sanitized v-html container. Server-side goldmark already
 * escapes raw HTML by default, but DOMPurify is the defense-in-depth
 * the v2.0 plan calls for: it strips on*= attributes, javascript:
 * URLs, and any tag we didn't whitelist.
 *
 * Allowed tag/attribute extensions: math/mfrac/mrow/msup/mn/mi for
 * KaTeX-rendered formulas, the data-* attribute family for any
 * helper attributes future MarkdownPreview variants emit. The
 * `class` attribute is preserved so the renderer's own styling
 * (callouts, code highlighting) survives.
 */
import { computed } from 'vue'
import DOMPurify from 'dompurify'

const props = defineProps<{ html: string }>()

const sanitized = computed(() =>
  DOMPurify.sanitize(props.html, {
    ADD_TAGS: ['math', 'mfrac', 'mrow', 'msup', 'mn', 'mi'],
    ADD_ATTR: ['class', 'data-href', 'data-target'],
  }),
)
</script>

<template>
  <div
    class="prose prose-invert max-w-none prose-pre:bg-bg-elevated prose-pre:border prose-pre:border-border prose-code:before:hidden prose-code:after:hidden"
    v-html="sanitized"
  />
</template>
