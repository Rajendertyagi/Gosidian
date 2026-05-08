# gosidian v2.0.0 — SPA cutover + REST API v1

This is the v2 rewrite. The HTMX-rendered web UI is gone; gosidian
now ships a Vue 3 single-page application backed by a versioned
REST API at `/api/v1/*`. End-user URLs are unchanged thanks to
Vue Router history mode — your bookmarks survive the upgrade.

## What changed

### For end users

- **Modern UI**: 4-mode editor (preview / split / stacked / edit-only)
  on CodeMirror 6, optimistic locking with conflict dialog, sidebar
  tree, searchable graph with three filter comboboxes
  (Project / Tag / Focus), theme picker
  (Catppuccin Mocha / Latte, Tokyo Night, Solarized Light), 5-language
  i18n (IT, EN, ES, FR, DE).
- **Lighthouse 100/100/100** on Login, Note, Graph (perf,
  accessibility, best practices).
- **Strict CSP**: no `unsafe-eval`, no `unsafe-inline`. Stronger
  defence against XSS and supply-chain script injection.

### For integrators

- **REST API v1** at `/api/v1/*` is now the public contract.
  Versioned, stable, fully documented. See `docs/migration-v2.md`
  for the complete v1.x → v2.0 route mapping.
- **Bearer-token authentication** replaces session cookies. POST
  to `/api/v1/login`, use the returned `token` as
  `Authorization: Bearer …`. Tokens are `gsp_<base64url>` and are
  rotated on every login.
- **MCP transport at `/mcp/sse`**: unchanged. MCP clients keep
  working without reconfiguration.
- **Optimistic locking** with `ETag` + `If-Match` → 412 on
  conflict. The SPA's conflict dialog handles the resolution UX.

### For operators

- **Single binary, single port** (8080). The SPA bundle is
  `go:embed`ed. No separate static asset deployment.
- **Strict CSP + 4 defence headers** out of the box.
- **No schema migration** required. SQLite vault format unchanged.

## Migration from v1.x

Walk through `docs/migration-v2.md`. Summary:

1. **Bookmarks** keep working — end-user URLs are unchanged.
2. **Legacy HTMX endpoints** (`/api/preview`, `/api/render`,
   `/api/tree`, `/api/backlinks`, `/api/note-excerpt`,
   `/api/command-palette`, `/api/attach`, `/api/upload`,
   `/api/i18n`) return 404. Migrate clients to the equivalents
   under `/api/v1/*`.
3. **Login flow** is JWT REST POST instead of session cookie.
4. **`GOSIDIAN_SPA_MODE`** env var was the dev-branch feature
   gate. It's gone — drop it from your env / compose.
5. **Database / vault format** unchanged. No migration needed.
6. **Pull the new image**:
   `ghcr.io/daniele-chiappa/gosidian:v2.0.0`. Recreate the
   container, no volume changes.

## Quick start

```bash
docker run -d \
  --name gosidian \
  -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v2.0.0
```

Open `http://localhost:8080/`, sign in with your existing
credentials, and you're done. MCP clients connect to
`http://localhost:8080/mcp/sse` with the bearer token from the
admin token store.

## Acknowledgements

- Vue 3.5 + Pinia 2 + Vue Router 4 stack
- CodeMirror 6 multi-mode editor
- Cytoscape 3.30 + fcose for the graph
- vue-i18n 9 with `@intlify/unplugin-vue-i18n` AOT precompile
  (the trick that lets us run under strict CSP)
- Playwright (chromium + firefox) for canary regression guards
