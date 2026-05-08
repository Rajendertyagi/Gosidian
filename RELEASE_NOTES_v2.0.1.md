# gosidian v2.0.1 — deps cleanup

PATCH release. Closes outstanding Dependabot debt against v2.0.0:
two open Go-module PRs and three high+critical npm dev-dependency
advisories. No runtime behaviour change — the SPA bundle output is
byte-identical to v2.0.0.

## What changed

### Go modules

- `github.com/mark3labs/mcp-go` 0.47.1 → 0.50.0
- `github.com/fsnotify/fsnotify` 1.7.0 → 1.10.0
- Go toolchain directive 1.25.0 → 1.25.5 (auto-bumped by `go get`,
  build target still pinned to `golang:1.25-alpine` in the Dockerfile)

### npm devDependencies

- `happy-dom` 15.0.0 → 20.9.0 — addresses GHSA-37j7-fg3j-429f
  (critical, VM Context Escape RCE), GHSA-w4gp-fjgq-3q4g (high, fetch
  credentials), GHSA-6q6h-j7hj-3r64 (high, ECMAScript module compiler
  injection).
- `playwright` + `@playwright/test` 1.47.0 → 1.59.1 — addresses
  GHSA-7mvr-c777-76hp (high, browser download integrity).

### Deliberately not bumped

- **Vite 5 → 6** (GHSA-4w7w-66w2-5vf9 medium) and **esbuild 0.21 →
  0.25** (GHSA-67mh-4wv8-2f99 medium): dev-only attack paths (dev
  server only — never on production builds), GitHub-hosted CI runner
  mitigates, and Vite 6 is a major upgrade with cascading plugin
  refresh cost. Tracked for the v2.1 cycle.

## For end users

Nothing changes. The SPA bundle is identical, the binary behaviour is
identical. If you're already on v2.0.0 the v2.0.1 image is a drop-in
swap purely to keep the dev / CI side free of advisory debt.

## For operators

Pull `ghcr.io/daniele-chiappa/gosidian:v2.0.1` (or `:latest`) and
recreate. No volume changes, no schema migration, no config delta.

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v2.0.1
docker rm -f gosidian
docker run -d --name gosidian -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v2.0.1
```

## Verification

- `go vet ./...`, `go test ./...` green across all 16 packages
- `npm audit --omit=dev` = 0 vulnerabilities
- Vitest 16/16 green with happy-dom 20
- `vite build` produces byte-identical `dist/` (no bundle drift)
