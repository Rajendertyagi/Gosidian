import { expect, test } from '@playwright/test'

/**
 * BUG-009 regression guard. The Phase 6 incident (blank page on
 * load because vue-i18n's runtime compiler hit `new Function()` and
 * tripped the strict `script-src 'self'` CSP) passed every Vitest
 * spec — Vitest runs in Node where `eval` is allowed, the browser
 * with CSP enforcement is the only place the failure shows up.
 *
 * This canary loads /login (the unauthenticated entry point) in a
 * real Chromium with our production CSP active and asserts:
 *
 *   1. The form actually rendered (no blank #app).
 *   2. No JavaScript page errors fired during boot.
 *   3. No CSP violations were reported by the browser.
 *
 * Anything that recreates the BUG-009 symptom (wrong CSP, eval-
 * dependent dep, init-order race) trips at least one of those.
 */
test('BUG-009: SPA shell boots under strict CSP without runtime eval', async ({ page }) => {
  const pageErrors: string[] = []
  const cspViolations: string[] = []
  const consoleErrors: string[] = []

  page.on('pageerror', (e) => pageErrors.push(`${e.message}\n${e.stack ?? ''}`))
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text())
  })
  // Browsers report violations either via the report-uri (we don't
  // configure one) or as console errors — the listener above
  // catches the latter. We separately match a substring for clarity.

  await page.goto('/login', { waitUntil: 'domcontentloaded' })

  // The login form must paint — anything blanker means the bundle
  // crashed mid-init.
  await expect(page.locator('input[name="username"], input#username, input[autofocus]')).toBeVisible()
  await expect(page.locator('input[type="password"]')).toBeVisible()

  // BUG-009 surface: any "blocked by CSP" or eval-related console
  // error is a hard fail.
  for (const text of consoleErrors) {
    expect(text, `unexpected console error: ${text}`).not.toMatch(/blocked by csp/i)
    expect(text, `unexpected eval error: ${text}`).not.toMatch(/eval(error)?|new Function/i)
  }
  cspViolations.push(...consoleErrors.filter((t) => /blocked by csp/i.test(t)))
  expect(cspViolations).toEqual([])
  expect(pageErrors).toEqual([])
})

test('AppShell mounts after login (no blank #app, TopBar nav present)', async ({ page }) => {
  // Defensive: this depends on a known seeded user existing in the
  // target vault. Defaults to the v2spa container's `owner` /
  // `testpass123` seed; override via PLAYWRIGHT_USER /
  // PLAYWRIGHT_PASS env when running against another instance
  // (prod 8080 ships with a different account).
  const user = process.env.PLAYWRIGHT_USER ?? 'owner'
  const pass = process.env.PLAYWRIGHT_PASS ?? 'testpass123'
  await page.goto('/login', { waitUntil: 'domcontentloaded' })
  await page.fill('input[type="text"], input#username, input[name="username"]', user)
  await page.fill('input[type="password"]', pass)
  await page.click('button[type="submit"]')

  // After login we expect to land on / with the AppShell + TopBar
  // visible. Use a generic landmark that's stable across phases:
  // the brand label.
  await expect(page.locator('text=gosidian').first()).toBeVisible({ timeout: 8000 })

  // BUG-009 symptom check: the app root must not be empty.
  const appHTML = await page.locator('#app').innerHTML()
  expect(appHTML.length).toBeGreaterThan(50)
})
