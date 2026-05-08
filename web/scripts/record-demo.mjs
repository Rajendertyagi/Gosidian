import { chromium } from 'playwright'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const BASE = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8083'
const USER = process.env.PLAYWRIGHT_USER ?? 'demo'
const PASS = process.env.PLAYWRIGHT_PASS ?? 'demopass1'
const OUT = process.env.DEMO_OUT_DIR ?? '/work/out'

await fs.mkdir(OUT, { recursive: true })

const browser = await chromium.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
})
const ctx = await browser.newContext({
  viewport: { width: 1100, height: 700 },
  recordVideo: { dir: OUT, size: { width: 1100, height: 700 } },
  baseURL: BASE,
})
const page = await ctx.newPage()

const beat = (ms) => page.waitForTimeout(ms)
const safe = async (label, fn) => {
  try {
    console.log(label)
    await fn()
  } catch (e) {
    console.warn(`  ${label} skipped: ${e.message?.split('\n')[0] || e}`)
  }
}

try {
  await safe('1/9 login', async () => {
    await page.goto('/login', { waitUntil: 'domcontentloaded' })
    await beat(800)
    await page.fill('input[type="text"], input[name="username"]', USER)
    await beat(200)
    await page.fill('input[type="password"]', PASS)
    await beat(400)
    await page.click('button[type="submit"]')
    await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 8000 })
  })

  await safe('2/9 home + sidebar tree', async () => {
    await beat(1800)
  })

  await safe('3/9 open welcome note', async () => {
    await page.goto('/notes/welcome.md', { waitUntil: 'domcontentloaded' })
    await beat(2200)
  })

  await safe('4/9 navigate to graph', async () => {
    await page.goto('/graph', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('canvas', { timeout: 10_000 })
    await beat(2200)
  })

  await safe('5/9 hover graph node', async () => {
    const canvas = page.locator('canvas').first()
    const box = await canvas.boundingBox()
    if (!box) return
    await page.mouse.move(box.x + box.width * 0.3, box.y + box.height * 0.4)
    await beat(700)
    await page.mouse.move(box.x + box.width * 0.5, box.y + box.height * 0.5)
    await beat(900)
    await page.mouse.move(box.x + box.width * 0.6, box.y + box.height * 0.45)
    await beat(1500)
  })

  await safe('6/9 settings -> theme switch', async () => {
    await page.goto('/settings', { waitUntil: 'domcontentloaded' })
    await beat(1200)
    const select = page.locator('select').first()
    await select.selectOption('tokyo-night').catch(() => {})
    await beat(1500)
    await select.selectOption('catppuccin-latte').catch(() => {})
    await beat(1500)
    await select.selectOption('catppuccin-mocha').catch(() => {})
    await beat(800)
  })

  await safe('7/9 open architecture note', async () => {
    await page.goto('/notes/blog%2Farchitecture.md', { waitUntil: 'domcontentloaded' })
    await beat(2000)
  })

  await safe('8/9 enter editor (CodeMirror)', async () => {
    await page.goto('/notes/blog%2Farchitecture.md/edit', { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('.cm-editor, .cm-content', { timeout: 10_000 }).catch(() => {})
    await beat(2500)
  })

  await safe('9/9 wrap', async () => {
    await beat(700)
  })
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
