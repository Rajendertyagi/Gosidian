# gosidian v2.1.1 — Go deps cleanup + Node 24 LTS

PATCH release. Closes outstanding Dependabot debt against v2.1.0:
four open Go-module PRs and the Docker base bump (overridden to
LTS). No runtime behaviour change — the SPA bundle is not affected.

## What changed

### Go modules

- `github.com/fsnotify/fsnotify` 1.10.0 → 1.10.1
- `golang.org/x/term` 0.42.0 → 0.43.0
- `golang.org/x/crypto` 0.50.0 → 0.51.0
- `github.com/mark3labs/mcp-go` 0.50.0 → 0.52.0 (includes upstream
  fix `setTools may resulted in an empty tools`, relevant to
  `internal/mcp/tools.go`)
- Transitive: `golang.org/x/sys` 0.44, `golang.org/x/text` 0.37

### Docker base

- Builder stage `Dockerfile`: `node:22-alpine` → `node:24-alpine`.
  Dependabot proposed `node:26-alpine` (Current release); we
  override to **Node 24 LTS** (support through October 2027) for
  long-term stability over latest. Same multi-stage pipeline,
  same final image size.

### Deferred

- **`vite` 5 → 8 + `esbuild` removal + `vitest` major** is *not*
  bundled here. The PR proposes a 3-major-version jump of Vite,
  which warrants a dedicated incremental upgrade plan
  (5 → 6 → 7 → 8) with runtime SPA validation per step (CSP
  semantics + plugin compatibility for vue / intlify / tailwind).
  Tracked for the v2.2.x cycle. The two related Dependabot alerts
  remain `dismissed: tolerable_risk` (dev-only, GitHub-hosted CI
  runner is the only consumer of `vite dev`).

## For end users

Nothing changes. The SPA `dist/` is byte-identical to v2.1.0 — the
embedded bundle is not rebuilt by this PATCH. If you're already on
v2.1.0 the v2.1.1 image is a drop-in swap purely to keep the Go
side current and the Docker base on LTS.

## For operators

Pull `ghcr.io/daniele-chiappa/gosidian:v2.1.1` (or `:latest`) and
recreate. No volume changes, no schema migration, no config delta.

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v2.1.1
docker rm -f gosidian
docker run -d --name gosidian -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v2.1.1
```

## Verification

- `go vet ./...`, `go test ./...` green across all 16 packages
- `npm audit --omit=dev` = 0 vulnerabilities (unchanged)
- Vitest 16/16 green
- SPA `dist/` byte-identical to v2.1.0 (no bundle drift)
