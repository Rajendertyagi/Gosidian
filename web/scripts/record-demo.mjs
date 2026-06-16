import { chromium } from 'playwright'
import { promises as fs } from 'node:fs'
import path from 'node:path'

// record-demo.mjs — drives a headless Chromium through the README demo
// storyboard and records a .webm. Convert it to docs/demo.gif with ffmpeg
// (palettegen/paletteuse); see the project docs for the exact filtergraph.
//
// The storyboard leans on the plancia deep-link (`/?w=tok1,tok2,…`) so each
// scene is a single navigation that opens the right tiled windows — no clicking
// through the UI to set up state. Tokens: note:<path>, history:<path>, graph.
//
// Everything is env-parametrized so it runs against any instance (the demo
// vault on :8081 by default). Locale + preset are forced from the first paint
// to avoid the persisted-store theme/locale flicker on reload.
const BASE = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8081'
const USER = process.env.PLAYWRIGHT_USER ?? 'demo'
const PASS = process.env.PLAYWRIGHT_PASS ?? 'demopass1'
const OUT = process.env.DEMO_OUT_DIR ?? '/work/out'
const LOCALE = process.env.DEMO_LOCALE ?? 'en'
const PRESET = process.env.DEMO_PRESET ?? 'catppuccin-mocha'

await fs.mkdir(OUT, { recursive: true })

const enc = encodeURIComponent
const tok = {
  guide: `note:${enc('atlas/docs/guide/getting-started.md')}`,
  api: `note:${enc('atlas/docs/api-reference.html')}`,
  arch: `note:${enc('atlas/memory/architecture.md')}`,
  hist: `history:${enc('atlas/memory/architecture.md')}`,
  graph: 'graph',
}

const browser = await chromium.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
})

// Pre-auth in a throwaway context so the recording starts already inside the
// app (no login screen in the gif).
const auth = await browser.newContext({ viewport: { width: 1280, height: 720 } })
const ap = await auth.newPage()
await ap.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' })
await ap.fill('input[name="username"], input[type="text"]', USER)
await ap.fill('input[type="password"]', PASS)
await ap.click('button[type="submit"]')
await ap.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 12_000 })
const storageState = await auth.storageState()
await auth.close()

const ctx = await browser.newContext({
  viewport: { width: 1280, height: 720 },
  storageState,
  recordVideo: { dir: OUT, size: { width: 1280, height: 720 } },
  baseURL: BASE,
})
// Force a consistent theme + locale from the very first paint.
await ctx.addInitScript(
  ([preset, locale]) => {
    try {
      localStorage.setItem('gosidian.ui', JSON.stringify({ preset, locale }))
      document.documentElement.dataset.preset = preset
    } catch (e) {
      void e
    }
  },
  [PRESET, LOCALE],
)

const page = await ctx.newPage()
const beat = (ms) => page.waitForTimeout(ms)
const go = async (w) => {
  const first = w.split(',')[0]
  await page.goto(`/?w=${w}&f=${first}`, { waitUntil: 'domcontentloaded' })
  await beat(1100)
}

try {
  // 1 — plancia tiling: illustrated guide + html note + graph, side by side.
  await go(`${tok.guide},${tok.api},${tok.graph}`)
  await beat(2600)

  // 2 — illustrated guide, scrolled to reveal the inline images.
  await go(tok.guide)
  await page.mouse.move(640, 400) // over the note content so the wheel scrolls it
  await beat(500)
  for (let i = 0; i < 5; i++) {
    await page.mouse.wheel(0, 320)
    await beat(720)
  }
  await beat(300)

  // 3 — the two note kinds side by side (markdown + single-file html).
  await go(`${tok.guide},${tok.api}`)
  await beat(2600)

  // 4 — versioning: a note next to its git history.
  await go(`${tok.arch},${tok.hist}`)
  await beat(2800)

  // 5 — manual creation: the + on a tree folder opens the create form.
  await go(tok.guide)
  const plus = page.locator('button[aria-label="New note in this folder"]').first()
  await plus.click({ force: true }).catch(() => {})
  await beat(2600)

  // 6 — graph view of the project, with a little hover motion.
  await go(tok.graph)
  await page.waitForSelector('canvas', { timeout: 8000 }).catch(() => {})
  const box = await page.locator('canvas').first().boundingBox().catch(() => null)
  if (box) {
    await page.mouse.move(box.x + box.width * 0.42, box.y + box.height * 0.45)
    await beat(800)
    await page.mouse.move(box.x + box.width * 0.6, box.y + box.height * 0.52)
    await beat(1300)
  } else {
    await beat(1800)
  }
} finally {
  await ctx.close()
  await browser.close()
}

const files = await fs.readdir(OUT)
const webm = files.find((f) => f.endsWith('.webm'))
if (!webm) {
  console.error('no .webm produced!')
  process.exit(1)
}
console.log('video:', path.join(OUT, webm))
