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

test('AppShell mounts under CSP with no runtime i18n compile (auth-shell BUG-009 variant)', async ({
  page,
  context,
}) => {
  // The /login canary above only covers the unauthenticated entry point —
  // and LoginView never calls t(). vue-i18n's message compiler only trips
  // once a t()-using component mounts (TopBar/Sidebar/WindowManager), which
  // happens in the AUTHENTICATED shell. A vite8/intlify11 build paired with
  // a mismatched vue-i18n runtime once shipped messages the runtime had to
  // compile on the fly, throwing SyntaxError and blanking the shell — green
  // on /login, broken after login.
  //
  // Inject a fake auth state so the shell mounts without a seeded user (the
  // API calls 401, but the i18n SyntaxError — if any — fires at component
  // setup, before/independent of the network). No credentials needed → this
  // runs in CI against any instance.
  await context.addInitScript(() => {
    localStorage.setItem(
      'gosidian.auth',
      JSON.stringify({ token: 'canary-fake-token', user: { username: 'canary', role: 'owner' } }),
    )
  })

  const pageErrors: string[] = []
  const consoleErrors: string[] = []
  page.on('pageerror', (e) => pageErrors.push(`${e.message}\n${e.stack ?? ''}`))
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text())
  })

  await page.goto('/', { waitUntil: 'domcontentloaded' })
  // Give the shell a beat to mount its t()-calling components.
  await page.waitForTimeout(1500)

  // A blank #app means a component setup() threw during mount.
  const appHTML = await page.locator('#app').innerHTML()
  expect(appHTML.length, 'authenticated #app is blank — a component setup() crashed').toBeGreaterThan(100)

  // No vue-i18n runtime compiler / eval / CSP error may fire on the shell.
  for (const text of [...consoleErrors, ...pageErrors]) {
    expect(text, `i18n/eval/CSP error on shell: ${text}`).not.toMatch(
      /message-compiler|new Function|eval(error)?|blocked by csp/i,
    )
  }
})
