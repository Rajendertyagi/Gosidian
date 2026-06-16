import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import { fileURLToPath, URL } from 'node:url'

// gosidian SPA Vite config.
//
// Build output is written into the Go server's embedded FS at
// internal/server/web/dist via --outDir at npm run build time. Asset
// URLs are rooted at /static/dist/ so the embed handler can serve
// them with aggressive cache headers; the generated index.html and
// HTML shell stays at /, served by handlers_spa.go.
export default defineConfig({
  plugins: [
    vue(),
    // Pre-compile i18n message catalogs into AOT functions at build
    // time. Without this, vue-i18n's runtime compiler uses
    // `new Function(...)` to compile messages on demand — blocked by
    // our strict CSP `script-src 'self'` (no `unsafe-eval`), which
    // would otherwise throw on every t() call.
    VueI18nPlugin({
      include: [fileURLToPath(new URL('../internal/i18n/catalogs/**', import.meta.url))],
      runtimeOnly: true,
      compositionOnly: true,
      // Catalogs are shared with the Go side and contain inline <code>
      // / <kbd> tags for some help blurbs — strictMessage off keeps
      // them as literal strings (we control the rendering surface,
      // and DOMPurify still sanitises any v-html paths).
      strictMessage: false,
      escapeHtml: false,
    }),
  ],
  base: '/static/dist/',
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@catalogs': fileURLToPath(new URL('../internal/i18n/catalogs', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // During dev, proxy API + SSE to the Go server so the SPA can
      // run with `npm run dev` while the binary serves data.
      '/api': 'http://127.0.0.1:8080',
      '/static/dist': 'http://127.0.0.1:8080',
    },
  },
  build: {
    target: 'es2022',
    sourcemap: true,
    manifest: true,
    rollupOptions: {
      output: {
        // Split heavy chunks (graph, editor) so the initial shell
        // stays small on first load. Vite 8 / Rolldown dropped the
        // object form of manualChunks — use the function form (module
        // id → chunk name), which both Rollup and Rolldown accept.
        manualChunks(id) {
          if (/node_modules\/cytoscape(-fcose)?\//.test(id)) return 'graph'
          if (/node_modules\/@codemirror\//.test(id)) return 'editor'
        },
      },
    },
  },
})
