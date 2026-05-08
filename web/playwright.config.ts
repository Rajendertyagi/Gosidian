import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright config for the gosidian SPA. Single Chromium project,
 * targets the gosidian-v2spa container on 8082 by default. Override
 * via PLAYWRIGHT_BASE_URL env var (CI lane points it at the
 * pipeline-built container instead).
 *
 * The point of this lane in v2.0 is BUG-009 prevention: load the
 * SPA in a real browser with the production CSP attached, fail the
 * build the moment a runtime eval / new Function() / mis-resolved
 * resource breaks first paint. Wider end-to-end coverage (note
 * CRUD, conflict, SSE, search, graph) lands in Phase 7.x as the
 * suite grows.
 */
export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8082',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Firefox lane is the cross-engine guardrail. CSP enforcement is
    // the area where Gecko and Blink diverge most often (BUG-009 was
    // CSP-driven), so a Firefox-only regression on script-src /
    // dynamic-import would otherwise pass the Chromium-only canary
    // silently. The MS Playwright image ships firefox preinstalled,
    // so the cost is ~3s extra per test run.
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],
})
