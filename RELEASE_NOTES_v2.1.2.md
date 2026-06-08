# gosidian v2.1.2 — security bundle

PATCH release. Closes six open Dependabot PRs (#23–#28) and resolves
six Dependabot security alerts. The one runtime-facing fix is **axios**
in the SPA bundle; the Go bumps are routine hygiene; the dev-only
**vitest** critical is deferred to the Vite major upgrade.

## What changed

### Security (runtime)

- **`axios`** 1.15.2 → 1.16.0 — bundled in the SPA. Patches three
  advisories at once:
  - ReDoS via cookie-name injection (high)
  - allocation of resources without limits / DoS (high)
  - proxy-authorization header injection via prototype pollution (low)

### Security (dev-only)

- **`js-cookie`** 3.0.5 → 3.0.8 — transitive via `js-beautify`,
  build-time only. Patches a per-instance prototype hijack in
  `assign()` (high).

### Go modules

- `github.com/mark3labs/mcp-go` 0.52.0 → 0.54.1
- `modernc.org/sqlite` 1.50.0 → 1.51.0
- `golang.org/x/crypto` 0.51.0 → 0.52.0
- Transitive via `go mod tidy`: `golang.org/x/sys` 0.45,
  `modernc.org/libc` 1.72.3

### Deferred

- **`vitest` 2.1.9 → 4.1.0** is *not* bundled here. The critical
  advisory (Vitest UI server arbitrary file read / exec) is patched
  only in 4.1.0, and vitest 4 requires `vite ^6 || ^7 || ^8`. That
  drags in the full Vite 5 → 8 major upgrade, which warrants a
  dedicated incremental plan (5 → 6 → 7 → 8) with runtime SPA
  validation per step — tracked for the v2.2.x cycle. vitest is a
  dev-only test runner: the vulnerable `vitest --ui` surface is never
  built into the image nor exposed in production, so the alert stays
  `dismissed: tolerable_risk` until the Vite upgrade lands.

## For end users

This is a security PATCH. Unlike v2.1.0 / v2.1.1, the SPA `dist/`
bundle **is** rebuilt to embed axios 1.16.0, so the front-end no
longer carries the vulnerable axios. There is no UI, schema, or config
change — it is a drop-in swap.

## For operators

Pull `ghcr.io/daniele-chiappa/gosidian:v2.1.2` (or `:latest`) and
recreate. No volume changes, no schema migration, no config delta.

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v2.1.2
docker rm -f gosidian
docker run -d --name gosidian -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v2.1.2
```

## Verification

- `npm audit --omit=dev` = **0 vulnerabilities** (runtime clean)
- Vitest 16/16 green (runner unchanged at 2.1.9)
- `go build ./...`, `go vet ./...`, `go test -race ./...` green across
  all packages
- SPA production build green; bundle sizes unchanged vs v2.1.0
  (index 89.6 KB gz, graph 176 KB gz, editor 198 KB gz)
