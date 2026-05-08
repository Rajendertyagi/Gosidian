import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

// Vitest config inherits from vite.config (so the `@/*` and
// `@catalogs/*` aliases resolve identically to runtime), then layers
// the test runner settings: happy-dom for DOM-dependent specs,
// `tests/setup.ts` for global lifecycle hooks (Pinia reset etc.).
export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'happy-dom',
      globals: true,
      setupFiles: ['./tests/setup.ts'],
      include: ['tests/unit/**/*.{test,spec}.ts'],
      exclude: ['tests/e2e/**', 'node_modules/**'],
    },
  }),
)
