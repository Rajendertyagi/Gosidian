<script setup lang="ts">
/**
 * HTMLPreview — renders a single-file HTML note (HTML + inline CSS + inline JS)
 * inside a locked-down sandboxed iframe.
 *
 * SECURITY MODEL (ADR-011) — do not weaken without review:
 *  - sandbox="allow-scripts" WITHOUT allow-same-origin. The document runs in an
 *    opaque origin: it cannot read the parent's cookies, localStorage, or DOM,
 *    and cannot make credentialed (same-origin) requests to the gosidian API.
 *    NEVER add allow-same-origin alongside allow-scripts — together they
 *    dissolve the sandbox and reintroduce stored-XSS against the viewer.
 *  - an injected <meta http-equiv="Content-Security-Policy"> with
 *    default-src 'none' blocks ALL network: the note's JS runs but cannot
 *    exfiltrate data or pull remote resources. Notes must be self-contained
 *    (inline CSS/JS, data: images). External assets are blocked by design.
 *
 * The HTML is the author's content rendered as-is (NOT sanitized): isolation,
 * not sanitization, is the boundary here.
 *
 * PRINTING: HTML notes are not printable yet (IMP-053). When the parent prints,
 * the browser clips a sandboxed iframe to one page, and we cannot make the
 * iframe print itself: a srcdoc inherits the parent CSP (script-src 'self'),
 * which combines with the injected meta and blocks any inline print bridge. So
 * NoteView shows the Print button for markdown notes only.
 */
import { computed } from 'vue'

const props = defineProps<{ html: string }>()

// Restrictive policy injected INTO the iframe document. Allows inline script and
// style (the note's own), data: images/fonts/media, and nothing over the
// network. Mirrors the "single self-contained file" contract.
const INJECTED_CSP = [
  "default-src 'none'",
  "script-src 'unsafe-inline'",
  "style-src 'unsafe-inline'",
  "img-src data:",
  "font-src data:",
  "media-src data:",
].join('; ')

const META = `<meta http-equiv="Content-Security-Policy" content="${INJECTED_CSP}">`

const srcdoc = computed(() => {
  const raw = props.html ?? ''
  // Inject the CSP meta as early as possible so it governs everything that
  // follows. Place it inside an existing <head>, else after <html>, else wrap
  // the fragment in a minimal document.
  if (/<head[^>]*>/i.test(raw)) {
    return raw.replace(/<head[^>]*>/i, (m) => `${m}${META}`)
  }
  if (/<html[^>]*>/i.test(raw)) {
    return raw.replace(/<html[^>]*>/i, (m) => `${m}<head>${META}</head>`)
  }
  return `<!DOCTYPE html><html><head>${META}</head><body>${raw}</body></html>`
})
</script>

<template>
  <iframe
    :srcdoc="srcdoc"
    sandbox="allow-scripts"
    referrerpolicy="no-referrer"
    title="HTML note"
    class="w-full min-h-[70vh] h-full border-0 bg-white"
  />
</template>
