# gosidian web — Vue 3 SPA

The browser frontend that pairs with the gosidian Go binary. Source for
the v2.0 SPA scheduled for tag `v2.0.0` post-soak. See the authoritative
plan in `gosidian/plans/20260430-v2-spa-rewrite` (vault) for scope and
phasing.

## Getting started

```bash
nvm use            # picks node from .nvmrc (22)
npm ci
npm run dev        # http://localhost:5173, proxies API to 127.0.0.1:8080
```

Build artifacts are emitted into `../internal/server/web/dist/` so the
Go embed picks them up at compile time:

```bash
npm run build
```

## Layout

- `src/api/`         typed wrappers around `/api/v1/*`
- `src/stores/`      Pinia stores (auth, ui, theme, events)
- `src/composables/` reusable hooks (useNote, useTheme, useSSE, …)
- `src/components/`  primitives + domain + layout + editor + graph
- `src/views/`       one per route; lazy-imported by the router
- `src/locales/`     vue-i18n loader; catalogs come from `internal/i18n/catalogs/*.json`
- `src/styles/`      semantic CSS tokens + Tailwind layer

## Auth model

The SPA holds a Bearer token in `localStorage` (`gosidian.auth`). Axios
attaches it to every `/api/v1/*` request. CSP is strict (no
`unsafe-inline`/`unsafe-eval` in `script-src`), markdown rendering
passes through DOMPurify, and `npm audit` runs in CI.

## Testing

- `npm run test:unit`   Vitest + happy-dom + msw
- `npm run test:e2e`    Playwright against a real binary

## Phase 0 status

Scaffolding only. The shell renders a placeholder `PlaceholderView`,
imports the IT/EN i18n catalogs, applies Catppuccin Mocha tokens, and
proves the toolchain. Phases 1–8 (REST API, auth, routes, editor,
graph, theme, tests, build/cutover) follow.
