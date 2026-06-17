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
 * IMAGE REFERENCES (IMP-059): a note may reference vault images by URL
 * (`<img src="/vault-files/.../attachments/x.webp">`) instead of inlining them.
 * The iframe CSP only allows `img-src data:`, so we fetch those vault images in
 * the PARENT context (where /vault-files is reachable) and inline them as data:
 * URIs at render time only. The stored note is never modified — the reference
 * form is preserved for MCP reads, downloads and storage (token savings). This
 * stays within ADR-011: the CSP is unchanged, the iframe never reaches the
 * network, and data: images are inert.
 *
 * The HTML is the author's content rendered as-is (NOT sanitized): isolation,
 * not sanitization, is the boundary here.
 */
import { ref, watch, onMounted } from 'vue'

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

// Match <img> src attributes pointing at the vault's public /vault-files/ path.
const vaultImgRe = /(<img\b[^>]*?\bsrc\s*=\s*)(["'])(\/vault-files\/[^"']+)\2/gi

const dataUrlCache = new Map<string, string>()

async function toDataUrl(url: string): Promise<string | null> {
  const cached = dataUrlCache.get(url)
  if (cached) return cached
  try {
    // The server returns the data: URI directly (?inline), with an immutable
    // Cache-Control — content-addressed images never change, so the browser
    // keeps the base64 forever and we don't regenerate it on every render.
    const sep = url.includes('?') ? '&' : '?'
    const res = await fetch(`${url}${sep}inline=1`)
    if (!res.ok) return null
    const dataUrl = (await res.text()).trim()
    if (!dataUrl.startsWith('data:')) return null
    dataUrlCache.set(url, dataUrl)
    return dataUrl
  } catch {
    return null
  }
}

async function inlineVaultImages(html: string): Promise<string> {
  const urls = new Set<string>()
  for (const m of html.matchAll(vaultImgRe)) {
    if (m[3]) urls.add(m[3])
  }
  if (urls.size === 0) return html
  const resolved = new Map<string, string>()
  await Promise.all(
    [...urls].map(async (u) => {
      const d = await toDataUrl(u)
      if (d) resolved.set(u, d)
    }),
  )
  return html.replace(vaultImgRe, (full, pre, quote, url) => {
    const d = resolved.get(url)
    return d ? `${pre}${quote}${d}${quote}` : full
  })
}

// Strip a leading frontmatter block so it never renders as visible text. The
// HTML-comment form (<!-- --- ... --- -->, ADR-011) is already invisible, but
// the bare markdown form (--- ... ---) shows — drop either.
function stripFrontmatter(html: string): string {
  return html
    .replace(/^\s*<!--\s*\r?\n---[\s\S]*?---\s*\r?\n?\s*-->\s*/, '')
    .replace(/^\s*---\r?\n[\s\S]*?\r?\n---\r?\n?/, '')
}

// The shell is served with a per-request CSP nonce (script-src 'self'
// 'nonce-X'). An about:srcdoc iframe INHERITS that policy, which intersects
// away the injected 'unsafe-inline' above — so a note's inline <script> only
// runs if it carries the nonce. Read it from the shell <meta> and stamp it
// onto every <script> tag (BUG-019). Absent (e.g. `npm run dev`), leave the
// markup untouched and the dev shell's looser CSP applies.
function cspNonce(): string {
  return document.querySelector('meta[name="csp-nonce"]')?.getAttribute('content') ?? ''
}

function stampScriptNonce(html: string, nonce: string): string {
  if (!nonce) return html
  return html.replace(/<script(?=[\s>])/gi, `<script nonce="${nonce}"`)
}

function buildSrcdoc(rawHtml: string): string {
  const html = stampScriptNonce(stripFrontmatter(rawHtml), cspNonce())
  // Inject the CSP meta as early as possible so it governs everything that
  // follows. Place it inside an existing <head>, else after <html>, else wrap
  // the fragment in a minimal document.
  if (/<head[^>]*>/i.test(html)) {
    return html.replace(/<head[^>]*>/i, (m) => `${m}${META}`)
  }
  if (/<html[^>]*>/i.test(html)) {
    return html.replace(/<html[^>]*>/i, (m) => `${m}<head>${META}</head>`)
  }
  return `<!DOCTYPE html><html><head>${META}</head><body>${html}</body></html>`
}

// Render immediately (text/layout show at once), then swap in the inlined
// version once the vault images have been fetched + converted.
const srcdoc = ref(buildSrcdoc(props.html ?? ''))

async function rebuild() {
  const inlined = await inlineVaultImages(props.html ?? '')
  srcdoc.value = buildSrcdoc(inlined)
}

onMounted(rebuild)
watch(
  () => props.html,
  () => {
    srcdoc.value = buildSrcdoc(props.html ?? '')
    void rebuild()
  },
)
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
