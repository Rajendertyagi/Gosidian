# Changelog

All notable changes to gosidian are documented here. The format loosely
follows [Keep a Changelog](https://keepachangelog.com/); dates are
`YYYY-MM-DD`; versions follow [SemVer](https://semver.org/).

This file is the single source for per-release notes — each GitHub Release
pulls its body from the matching section below. There are no separate
`RELEASE_NOTES_*` files.

## [2.7.2] — 2026-06-15 — "dependency maintenance"

PATCH release. Routine dependency maintenance and a transitive security fix.
No behaviour changes; existing deployments need no migration.

### Security
- **esbuild pinned to `^0.28.1`** via an npm `overrides` entry, resolving
  `GHSA-gv7w-rqvm-qjhr` (missing binary integrity verification). The project
  uses esbuild only as a build-time tool (via Vite) and does not ship it in the
  runtime image, so real-world exposure was low — patched as hygiene. The fix
  stays on Vite 6; the Vite 6→8 major upgrade is intentionally deferred.

### Changed
- Bumped `golang.org/x/crypto` 0.52→0.53 and `golang.org/x/term` 0.43→0.44
  (plus transitive `x/sys`, `x/text`, `x/net`).
- Updated the Alpine runtime base image from 3.20 to 3.24.

## [2.7.1] — 2026-06-15 — "git-sync backup health + note download"

PATCH release. A degraded git-sync backup no longer marks the container
unhealthy, and a Download button is added to the note header. Purely additive
plus a health-reporting fix; existing deployments need no migration.

### Added
- **Download original file** — a Download button in the note header saves the
  note's raw source file (`.md` or `.html`) as-is, fully client-side (no server
  round-trip).

### Fixed
- **A degraded git-sync backup no longer fails container health.** When the
  git-sync backup is degraded — for example a push blocked by remote divergence
  or local repository corruption — the container is no longer reported
  `unhealthy`: `/healthz` readiness is now gated on the index (core) alone,
  while backup degradation stays observable in the `git_sync.healthy` field and
  the Prometheus gauge `gosidian_gitsync_status`. git-sync additionally
  classifies local repository corruption (`bad object` / empty object file) and
  surfaces an actionable error in place of a raw git message.

## [2.7.0] — 2026-06-13 — "print to PDF + unified frontmatter detection"

MINOR release. Adds a Print / Save-as-PDF action for Markdown notes in the web
UI, and unifies frontmatter detection across the indexer, linter and MCP tools
so they can no longer disagree on a note — fixing cases where MCP tools ignored
the frontmatter of `.html` notes. Purely additive; existing deployments need no
migration.

### Added

- **Print / Save as PDF for Markdown notes.** A Print button in the note header
  (view mode) prints just that note through the browser's native print /
  Save-as-PDF dialog. A scoped `@media print` stylesheet shows only the rendered
  note — hiding the rest of the plancia and the browser's own header/footer
  chrome — so a single, clean page reaches the printout. HTML notes are not
  printable yet (the browser clips the sandboxed iframe to a single page).

### Fixed

- **Frontmatter detection unified across subsystems.** A single path-aware
  parser primitive now backs the indexer, the linter and every MCP tool, so they
  can never disagree on whether a note has frontmatter or what its tags are. This
  fixes several MCP tools (`memory_todos`, `memory_importance`,
  `memory_refresh_hot`, `memory_handoff`, and the discovery tools) that
  previously read no frontmatter from `.html` notes — silently skipping their
  tags, status, and plan metadata. A cross-subsystem consistency test guards
  against the two detection paths drifting again.

## [2.6.0] — 2026-06-12 — "graph analytics + HTML notes"

MINOR release. Two additive feature tracks: graph-analytics MCP tools over the
wiki-link graph gosidian already maintains, and support for single-file `.html`
documents as first-class notes rendered in a sandboxed iframe (opt-in, off by
default). Purely additive; existing deployments need no migration.

### Added

- **Graph analytics MCP tools.** `memory_hubs` ranks the most-connected notes by
  undirected wiki-link degree (the inverse of orphan notes — where the graph
  concentrates). `memory_path` returns the shortest path between two notes over
  the wiki-link graph. Both scope to a single project or run vault-wide.
- **`unlinked-mentions` lint rule (opt-in).** Flags prose that names another
  note's title or basename without linking it, surfacing missing
  `[[wiki-links]]`. It is advisory and higher-noise on a dense vault, so it is
  excluded from the default lint run — name it explicitly in `rules` to enable it.
- **First-class single-file HTML notes (opt-in, off by default).** With
  `[vault] html_notes` enabled (env `GOSIDIAN_VAULT_HTML_NOTES`), a `.html` file
  (HTML + inline CSS/JS) is indexed like a markdown note: frontmatter (a
  `--- YAML ---` block wrapped in a leading HTML comment), tags, wiki-links and a
  full-text body all participate in the graph, backlinks and search. The web UI
  renders such notes in a sandboxed `<iframe sandbox="allow-scripts">` with no
  same-origin access and an injected `Content-Security-Policy: default-src 'none'`,
  so author-supplied scripts run but cannot reach the viewer's session, the API,
  or the network.

### Fixed

- **Lint frontmatter rules now understand HTML notes.** The `frontmatter-missing`
  and `frontmatter-tag-unknown` checks read an HTML note's comment-wrapped
  frontmatter (dispatching on file type the same way the indexer does), so a
  well-formed `.html` note is no longer reported as missing its frontmatter.
- **`memory_lint` reports the rules it actually ran.** The response `rules` field
  now reflects the requested rule set (e.g. an explicit opt-in rule) instead of
  always echoing the default list.

### Notes

- All changes are additive and backward-compatible. HTML notes are **off by
  default**: enable `html_notes` consciously after reviewing the sandbox/CSP
  security model, since it lets one user author markup that another user renders.
  No migration is required.

## [2.5.0] — 2026-06-10 — "self-improve opt-in toggle"

MINOR release. Operators can now toggle a token's self-improvement opt-in
after the token was created — from the CLI, the admin API, or the admin SPA —
instead of recreating the token or hand-editing the store. Bootstrap also stops
flagging the agent instruction file as missing when it correctly lives outside
the vault. Purely additive; existing deployments need no migration.

### Added

- **Toggle self-improve opt-in on existing tokens.** A new `gosidian token
  opt-in --id <id> [--off]` CLI subcommand (with `--all`), a
  `PATCH /api/v1/admin/tokens/{id}` endpoint, and a "Self-improve" toggle in the
  admin Tokens view let owners enrol or withdraw an already-issued MCP token
  from the self-improvement loop. Previously the flag could only be set at
  token-creation time. The token API view now includes `self_improve_opt_in`,
  and token updates are recorded with a new `token_update` audit action.

### Changed

- **`memory_bootstrap` no longer reports the instruction file as missing.** In
  the stub model the agent instruction file lives in the agent's working
  directory, outside the vault, so its vault-absence is expected. Bootstrap now
  sets `agent_md.expected_external` instead of listing `AGENTS.md` in `missing`
  (which is reserved for genuinely absent vault scaffold such as `hot.md` /
  `README.md`). This supersedes the v2.4.0 behavior where `AGENTS.md` was always
  added to `missing`.

### Notes

- All changes are additive and backward-compatible: existing tokens keep their
  current opt-in state, and the new API / bootstrap fields are additive — a
  client that ignored them keeps working. No migration is needed.

## [2.4.0] — 2026-06-10 — "directives versioning"

MINOR release. The agent-facing memory protocol gains versioned operational
directives served at session start, a thin versioned instruction-file stub,
and agent-agnostic instruction-file detection. Purely additive — existing
deployments need no migration; the new response fields are additive and the
self-healing directives upgrade older projects automatically.

### Added

- **Bootstrap-served directives.** `memory_bootstrap` now returns
  `directives_block` (the full operational directives — vault folder map,
  ingest rules, plan/skill conventions, end-of-task workflow, tag vocabulary)
  plus `directives_version`. Projects read the rules fresh every session
  instead of duplicating them in their instruction file, so they can no longer
  drift out of date.
- **Versioned instruction-file stub.** `memory_init_agent` now emits a thin
  stub (session-bootstrap rule + local-specifics) carrying a
  `<!-- gosidian:stub v=N -->` marker, and `memory_bootstrap` reports
  `stub_version` so an agent knows when its (rarely changing) stub should be
  regenerated. The full directives live in `directives_block`, not in the file.
- **Self-healing of pre-stub projects.** The served directives instruct a
  project whose instruction file predates the stub system to convert itself
  once at bootstrap — existing projects upgrade with no manual rollout.

### Changed

- **Agent-agnostic instruction-file detection.** Bootstrap no longer assumes
  `CLAUDE.md`: it probes `AGENTS.md`, `CLAUDE.md`, `.cursor/rules.mdc`,
  `CONVENTIONS.md` and reports the detected file as `agent_md` (replacing the
  former `CLAUDE.md`-specific `claude_md` field). `missing` lists `AGENTS.md`
  when none is present.
- **Single source of truth for conventions.** The scaffold templates
  (`karpathy-wiki` prompt + index READMEs) and the non-Claude init prompts
  (Cursor / Codex / Aider / generic) no longer restate the tag vocabulary,
  plan format, or ingest rules — they point at the bootstrap `directives_block`.

### Notes

- New MCP response fields are additive; a client that ignored them keeps
  working. The `claude_md` → `agent_md` rename only affects callers that read
  that informational field directly.

## [2.3.2] — 2026-06-09 — "security patch follow-up"

PATCH release. Completes the note-titles allocation hardening started in
v2.3.1. No user-facing or API changes; existing deployments need no
migration.

### Security

- **CodeQL `go/uncontrolled-allocation-size` (HIGH), second occurrence.**
  v2.3.1 capped the first of the two result-slice allocations in the
  note-titles autocomplete endpoint to a constant, but the second one (on
  the search branch) was left request-derived, so the alert stayed open.
  Both allocations now use the constant cap; neither depends on the
  request-supplied `limit`. As before, the value was already clamped to
  `[1, 50]`, so there was no practical exploit.

## [2.3.1] — 2026-06-09 — "security patch"

PATCH release. Dependency and static-analysis security fixes, a dev-tooling
upgrade, and dead-code / release-notes housekeeping. No user-facing or API
changes; existing deployments need no migration.

### Security

- **CodeQL `go/uncontrolled-allocation-size` (HIGH).** The note-titles
  autocomplete endpoint pre-allocated its result slice with a
  request-supplied `limit`. The value was already clamped to `[1, 50]`, so
  there was no real exploit, but the allocation now uses a constant cap so
  its size never depends on unvalidated input.
- **`go-ntlmssp` 0.1.0 → 0.1.1.** Closes a medium advisory (NTLM challenges
  could panic on malformed payloads); indirect dependency of the LDAP
  client.

### Changed

- **Dev tooling: Vite 5 → 6, Vitest 2 → 3.** Closes the critical Vitest
  advisory (UI-server arbitrary file read/exec) and the dev-only
  Vite/esbuild advisories — `npm audit` is now clean. No bundle behaviour
  change for end users; the runtime SPA stack (Vue 3.5, vue-i18n 9 AOT,
  Tailwind 3.4) is unchanged.

### Removed

- Dead code: three unbundled leftover SPA views from the plancia refactor
  (`HomeView`, `PlaceholderView`, `ConflictDialog`) and two unused Go
  test/router helpers.
- The per-version `RELEASE_NOTES_*.md` files — release notes are now
  consolidated into this single CHANGELOG (see the note above).

## [2.3.0] — 2026-06-08 — "plancia"

MINOR release. The web UI becomes a tiling window manager, and two
opt-in features land — all backward compatible and off by default.

### Added

- **Plancia window manager.** A niri-style scrollable tiling workspace:
  notes, the graph, search, and config forms open as resizable windows
  side by side instead of one page at a time. Windows resize in steps
  (small / medium / full), minimize to a scrollable footer, and a
  per-window "direct links" button opens the one-hop ego-graph of a note.
  Open windows + focus are encoded in the URL (shareable, reload-safe),
  with a `localStorage` fallback. The app menu moved into a collapsible
  sidebar section; the command palette (⌘K) and wikilinks open windows.
- **Global projects** (opt-in, off by default). Shared `global` (public)
  and `global-private` projects whose skills, agents, and scaffold
  templates any project can reuse, with local-overrides-global merge.
  Gated by `GOSIDIAN_GLOBAL_ENABLED` and a per-project `use_globals`
  flag. See `docs/vault/global-projects.md`.
- **Self-improvement loop** (experimental, off by default). Agents record
  structured usage-friction insights via the `memory_self_improve` MCP
  tool, kept in a private `insights` project for the owner to triage.
  Opt-in per token. See `docs/mcp/self-improvement.md`.
- Two new MCP tools — `memory_self_improve` and `memory_global_check`
  (48 tools total).

### Changed

- A note now opens as a **single window with an in-place View/Edit
  toggle** — editing no longer navigates to a separate page. The editor
  (CodeMirror) mounts lazily and is hidden for read-only users.
- **Documentation corrected**: the web UI has been a Vue 3 SPA since
  v2.0; the README, web-UI overview, FAQ, and architecture pages that
  still described the old HTMX stack are now accurate.
- `modernc.org/sqlite` updated to 1.52.0.

### Notes

- Fully backward compatible. Global projects and the self-improvement
  loop are **off by default**; existing deployments need no migration.
  Deep-link routes (`/notes/<path>`, `/graph`, …) still work — they open
  the matching window in the plancia.

## [2.2.0] — 2026-06-08 — "auth & roles"

MINOR release. Adds role-based access control, two-factor (TOTP), and
LDAP / Active Directory login on top of the existing multi-user web
login. Fully backward compatible: TOTP defaults to `off` and LDAP to
disabled, so existing single-owner / member setups are unaffected and
need no migration.

### Added

- **Role-based access control.** Three roles — **owner**, **member**,
  **guest** — enforced by a centralized, fail-closed authorization layer
  (`internal/authz`). A read the role may not perform returns **404**
  (the resource's existence is hidden); a forbidden write returns
  **403**. An unrecognized role degrades to public-read only.
- **Public / private projects.** Each project carries a `public` flag
  (default **private**). Public projects are readable by guests; private
  ones are visible only to owners and members. Guests are filtered
  consistently across the sidebar, search, tags, note titles, and the
  graph.
- **TOTP two-factor authentication.** A global mode — `off` /
  `optional` / `required` — plus a per-user override (inherit / enabled
  / disabled). Self-service enrolment with confirm-before-activate;
  `off` is a lockout-proof master switch. Configurable via
  `webauth.totp_mode` / `GOSIDIAN_TOTP_MODE`.
- **LDAP / Active Directory login.** Search-then-bind against an
  external directory; the first successful login auto-provisions a local
  **guest** account (no password stored). A local username always
  shadows LDAP. LDAPS and StartTLS are supported, with a configurable
  user filter (OpenLDAP `(uid=%s)`, Active Directory
  `(sAMAccountName=%s)`). New `[ldap]` config block and `GOSIDIAN_LDAP_*`
  environment variables.
- **Docs**: a new [Authentication & roles](docs/web-ui/authentication.md)
  page, plus a disposable LDAP test harness under `deploy/ldap-test/`.

### Changed

- **Graph view** now honours per-role visibility and opens on the most
  recently edited project the user can see, instead of rendering the
  entire vault at once.
- **`modernc.org/sqlite`** 1.51.0 → 1.52.0.

### Security

- Guests can never hold MCP tokens — token creation is owner/member-only
  and demoting a user to guest cascade-revokes their tokens — so the
  read-only boundary holds across both the web UI and MCP, with no
  MCP-layer changes required.

### Notes

- LDAP is validated end-to-end against OpenLDAP over plain LDAP, LDAPS,
  and StartTLS. The Active Directory path is configuration-only on the
  same code; validate it against your domain controller.

## [2.1.2] — 2026-06-08 — "security bundle"

PATCH bundle. Closes six open Dependabot PRs (#23–#28) and resolves
six Dependabot security alerts. The only runtime-facing fix is axios
in the SPA bundle; the Go bumps are routine hygiene; the dev-only
vitest critical is deferred (see below).

### Security

- **`axios`** 1.15.2 → 1.16.0 — runtime dependency, ships in the SPA
  bundle. Patches three advisories: ReDoS via cookie-name injection
  (high), allocation of resources without limits / DoS (high), and a
  proxy-authorization header injection via prototype pollution (low).
  Closes #25.
- **`js-cookie`** 3.0.5 → 3.0.8 — transitive, dev-only (via
  `js-beautify`). Patches per-instance prototype hijack in `assign()`
  (high). Closes #23.

### Changed

- **`github.com/mark3labs/mcp-go`** 0.52.0 → 0.54.1 (closes #28).
- **`modernc.org/sqlite`** 1.50.0 → 1.51.0 (closes #27).
- **`golang.org/x/crypto`** 0.51.0 → 0.52.0 (closes #24).
- Transitive via `go mod tidy`: `golang.org/x/sys` 0.44 → 0.45,
  `modernc.org/libc` 1.72.0 → 1.72.3.

### Deferred

- **`vitest`** 2.1.9 → 4.1.0 (#26) is *not* merged. The advisory
  (Vitest UI server arbitrary file read/exec, **critical**) is patched
  only in 4.1.0, which requires `vite ^6 || ^7 || ^8` — pulling in the
  full Vite 5 → 8 major upgrade already tracked for the v2.2.x cycle
  (incremental 5 → 6 → 7 → 8 with runtime SPA validation per step).
  vitest is a **dev-only** test runner; the vulnerable surface
  (`vitest --ui`) is never built into the image nor exposed in
  production. The alert remains `dismissed: tolerable_risk` until the
  Vite upgrade lands. #26 is closed without merging.

### Notes

- Unlike v2.1.0 / v2.1.1, the SPA `dist/` **is** rebuilt here to pick
  up axios 1.16.0 — this is a security release, not a Go-only PATCH.
  Vitest 16/16 green, `npm audit --omit=dev` = 0 vulnerabilities
  (runtime), `go test -race ./...` green across all packages.

## [2.1.1] — 2026-05-11 — "deps cleanup #2"

PATCH bundle. Closes 4 open Dependabot Go-module PRs and overrides
the Node 22 → 26 (Current) proposal with Node 22 → 24 (LTS) for
stability long-term. No runtime behaviour change — the SPA bundle is
unaffected (only Go deps + Docker base modified).

### Changed

- **`github.com/fsnotify/fsnotify`** 1.10.0 → 1.10.1 (closes #17).
- **`golang.org/x/term`** 0.42.0 → 0.43.0 (closes #18).
- **`golang.org/x/crypto`** 0.50.0 → 0.51.0 (closes #19).
- **`github.com/mark3labs/mcp-go`** 0.50.0 → 0.52.0 (closes #20).
  Includes upstream fix mark3labs/mcp-go#830
  *"setTools may resulted in an empty tools"* (v0.51.0) — defensive
  improvement to the pattern gosidian uses in `internal/mcp/tools.go`.
- **`Dockerfile`** builder stage: `node:22-alpine` → `node:24-alpine`
  (overrides Dependabot PR #16 which proposed `node:26-alpine`).
  Node 24 is the current LTS line (support through October 2027),
  Node 26 is the Current release with a shorter support window. The
  override prioritises stability over latest.
- Transitive bumps via `go mod tidy`: `golang.org/x/sys` 0.43 → 0.44,
  `golang.org/x/text` 0.36 → 0.37.

### Deferred

- **PR #15 vite 5.4.21 → 8.0.11 + esbuild removal + vitest major**
  is closed without merging. The jump spans 3 major versions of
  Vite (5 → 6 → 7 → 8) and removes esbuild as a direct dependency.
  CI build is green but the runtime SPA behaviour under strict CSP +
  plugin compatibility (@vitejs/plugin-vue, @intlify/unplugin-vue-i18n,
  Tailwind 3.4) was not validated. A dedicated upgrade plan with
  incremental 5 → 6 → 7 → 8 steps and runtime testing per step is
  tracked for the v2.2.x cycle. The two related Dependabot alerts
  (vite GHSA-4w7w-66w2-5vf9 medium, esbuild GHSA-67mh-4wv8-2f99
  medium) remain `dismissed: tolerable_risk` — dev-only attack
  surface, GitHub-hosted CI runner is the only consumer of `vite
  dev`.

### Notes

- **No behaviour change for end users**. The SPA `dist/` bundle is
  not rebuilt by this PATCH — the embedded assets match v2.1.0
  byte-for-byte. Only the Go binary and the builder image base
  change.
- `go vet ./...`, `go test ./...` 16/16 packages green
- `npm audit --omit=dev` = 0 vulnerabilities (unchanged from v2.0.1)
- Vitest 16/16 green

## [2.1.0] — 2026-05-08 — "lint vocabulary extension"

MINOR. Extends the `frontmatter-tag-unknown` lint rule with a
config-driven allow-list, so vaults can document their structural
tag patterns without weakening the rule for everyone. No behaviour
change for vaults that do not configure it.

### Added

- **`[lint.frontmatter_tag_vocabulary] extra_allowed`** in
  `<vault>/.gosidian/config.toml` — additive allow-list for the
  closed vocabulary checked by the `frontmatter-tag-unknown` rule.
  Format: `<namespace>:<value>` or bare token. Built-in namespaces
  (type/topic/status/pinned/project-name) are always honoured;
  the extension is purely additive — a vault never weakens its own
  discipline by setting this. Malformed entries (empty,
  leading/trailing colon, internal whitespace, double colon) are
  skipped silently at load time so a typo in the config does not
  crash the lint. See `docs/configuration.md#lint-vocabulary-extension`.
- New chainable setter `Linter.WithExtraAllowedTags(extra)` in the
  `internal/lint` package; `isKnownTag` is now a method so each
  Linter instance carries its own per-vault vocabulary extension.
- New `Server.SetLintExtraAllowedTags()` setter in the MCP package;
  `memory_lint` wires the per-vault config into each run.
- Three new unit tests covering the extension behaviour: extras
  silence configured tags, malformed entries are skipped silently,
  extras don't mask other unknowns.

### Changed

- `cmd/gosidian/main.go` reads `cfg.Lint.FrontmatterTagVocabulary.ExtraAllowed`
  at startup and passes it to the MCP server. Vaults without a
  `[lint]` section get the same behaviour as before.

### Notes

- **Backward-compatible**. Vaults without `[lint.frontmatter_tag_vocabulary]`
  see the built-in vocabulary unchanged. No migration, no schema
  delta, no runtime impact for existing deploys.
- Use case: a vault that legitimately uses tags outside the built-in
  namespaces (e.g. `status:reference` for reference notes that
  aren't snapshot/draft/done/archived, `topic:agent-template` for
  template-folder index notes) can document those tags in
  `.gosidian/config.toml` instead of accumulating warnings on every
  `memory_lint` run.

## [2.0.1] — 2026-05-08 — "deps cleanup"

PATCH bundle. Closes the two open Dependabot Go-module PRs and three
high+critical Dependabot npm advisories on dev dependencies. No
runtime behaviour change — the SPA bundle output is byte-identical
to v2.0.0.

### Changed

- **`github.com/mark3labs/mcp-go`** 0.47.1 → 0.50.0. Closes #12.
- **`github.com/fsnotify/fsnotify`** 1.7.0 → 1.10.0. Closes #13.
- **`web/` devDependencies**:
  - `happy-dom` 15.0.0 → 20.9.0 — closes GHSA-37j7-fg3j-429f
    (critical, VM Context Escape RCE), GHSA-w4gp-fjgq-3q4g (high,
    fetch credentials sourced from page origin), GHSA-6q6h-j7hj-3r64
    (high, ECMAScript module compiler unsanitised export names).
  - `playwright` + `@playwright/test` 1.47.0 → 1.59.1 — closes
    GHSA-7mvr-c777-76hp (high, browsers downloaded without integrity
    verification).
- **Go toolchain directive** auto-bumped 1.25.0 → 1.25.5 (side-effect
  of `go get`). Build target still pinned to `golang:1.25-alpine` in
  `Dockerfile`.

### Notes

- **Vite 5 → 6** (GHSA-4w7w-66w2-5vf9 medium, path traversal in dev
  server `.map` handling) and **esbuild 0.21 → 0.25**
  (GHSA-67mh-4wv8-2f99 medium, dev-server CSRF) **deliberately
  deferred**: dev-only attack paths (production build pipeline never
  exposes either dev server), GitHub-hosted CI runner is the only
  consumer, and Vite 6 is a major upgrade with a cascading plugin
  refresh cost (Vue plugin, intlify, Tailwind). Tracked for the v2.1
  cycle when the broader bundle modernisation lands.
- `vite build` produces a byte-identical SPA `dist/` post-bump — the
  upgraded dev tooling does not change the bundle. End users see no
  behavioural delta vs v2.0.0.
- Vitest unit suite 16/16 green with happy-dom 20. `go test ./...`
  green across all 16 packages. `npm audit --omit=dev` =
  0 vulnerabilities.

## [2.0.0] — 2026-05-08 — "SPA cutover + REST API v1"

**MAJOR.** The HTMX-rendered web UI is gone; gosidian now ships a
Vue 3 single-page application backed by a versioned REST API at
`/api/v1/*`. End-user URLs (`/notes/<path>`, `/projects/<slug>`,
`/graph`, `/search`, ...) keep working unchanged thanks to
Vue Router history mode + the SPA shell catch-all. The MCP transport
at `/mcp/sse` and the file upload pipeline are unchanged.

This is breaking for callers that hit legacy HTMX endpoints
directly. See `docs/migration-v2.md` and the **Migration from v1.x**
section below.

### Added

- **Vue 3 SPA** (`web/`) — production-grade single-page application
  served as `go:embed all:dist` from the Go binary. Stack: Vue 3.5 +
  TypeScript 5.5 (strict) + Vite 5 + Pinia 2 + vue-router 4 +
  Tailwind 3.4. AppShell + TopBar + sidebar tree, 4-mode editor
  (preview / split / stacked / edit-only) on CodeMirror 6, optimistic
  locking with conflict dialog (ETag + `If-Match` → 412),
  DOMPurify-sanitised markdown, axios + EventSource SSE, Pinia state
  with `pinia-plugin-persistedstate`. Lighthouse 100/100/100 on
  Login / Note / Graph.
- **REST API v1** at `/api/v1/*` — versioned, stable, fully
  documented. New `internal/api/v1/` package replaces the per-page
  Go handlers under `internal/server/handlers_*`. Endpoints cover
  the full v1.x surface plus the SPA-specific shapes (see
  `docs/migration-v2.md` for the full v1.x → v2.0 route mapping).
  Bearer-token authentication, rate-limited, JSON error envelope.
- **Strict Content-Security-Policy** attached to the SPA shell:
  `default-src 'self'`, `script-src 'self'` (no `unsafe-eval`,
  no `unsafe-inline`), `frame-ancestors 'none'`, `object-src 'none'`.
  Defence in depth: `X-Content-Type-Options nosniff`,
  `X-Frame-Options DENY`,
  `Referrer-Policy strict-origin-when-cross-origin`, minimal
  `Permissions-Policy`.
- **Theme presets** (4) — Catppuccin Mocha (default dark),
  Catppuccin Latte (light), Tokyo Night (dark blue),
  Solarized Light. Switchable at runtime via `/settings`. Each
  theme uses semantic CSS variable tokens; the Cytoscape graph
  resolves them to comma-form `rgb()`.
- **i18n** (5 languages: IT, EN, ES, FR, DE) — AOT-precompiled at
  build time via `@intlify/unplugin-vue-i18n` so vue-i18n's
  runtime message compiler never reaches `new Function()`. The
  SPA honours the server's `default_lang` from
  `GET /api/v1/version` on first boot.
- **Graph view** rewrite — Cytoscape 3.30 + fcose layout, three
  searchable comboboxes (Project / Tag / Focus) with sensible
  default sort orders (mtime desc, count desc, recent edits desc).
  Hover bring-forward highlights the focused node and dims the
  complement; full title shown on hover.
- **Playwright canary** (chromium + firefox) — regression guard
  for the runtime-eval / CSP-blocked-script class of incidents.
- **`docs/migration-v2.md`** — single source of truth for the
  upgrade path from v1.x, including the full v1.x → v2.0 route
  mapping table.

### Changed

- **Frontend bundle** is now `go:embed`ed from the SPA Vite build
  (`internal/server/web/dist/*`). Single binary, no separate static
  asset deployment. SPA shell catch-all serves `/notes/<path>`-style
  URLs to the Vue Router.
- **Login flow** — JWT-style bearer token (`gsp_<base64url>`)
  returned by `POST /api/v1/login`, persisted in the SPA under
  `localStorage["gosidian.auth"]` via Pinia persistedstate. The old
  cookie-session HTMX flow is gone.
- **Graph endpoint** payload includes server-side `tag`, `focus`,
  `depth`, `min_degree`, `limit` filters; the client now sends
  these as query params instead of computing on the fly.

### Removed

- **HTMX UI**: 21 Go HTML templates under
  `internal/server/templates/`, the per-page Go handlers
  (`internal/server/handlers_*.go`, `internal/server/render.go`),
  and the ~1 KLOC of custom JS / icons / CSS under
  `internal/server/static/{js,css,icons}/`. The new SPA + REST
  equivalent at `/api/v1/*` covers the same surface.
- **`gosidian_session` cookie auth** — replaced by Bearer tokens.
- **`GOSIDIAN_SPA_MODE` env var** — was a feature flag during the
  v2-spa development branch. With the cutover complete, the SPA is
  the only frontend and the flag is gone. Operators who set it
  explicitly can drop it from their env files.

### Fixed

- **Cytoscape theme rendering** — Cytoscape doesn't accept the
  `rgb(R G B)` space-separated form some Tailwind builds emit.
  The theme resolver now emits comma-form `rgb(R, G, B)` for graph
  styling. Fixes the blank/black graph nodes some users saw on
  initial paint after a theme switch.
- **i18n CSP failure** — vue-i18n's default runtime compiler used
  `new Function()`, which strict CSP blocks. Switched to AOT
  precompile via `@intlify/unplugin-vue-i18n`; the runtime carries
  no eval surface.
- **UI store hydration race** — `pinia-plugin-persistedstate`
  re-hydrated after `main.ts` had already read the store, causing
  themes to flash to the default on first paint. Hydration now
  happens inside `App.vue setup`.

### Security

- **`script-src 'self'`** strictly enforced. No `unsafe-eval`,
  no `unsafe-inline`. Verified end-to-end via Playwright canary
  in chromium + firefox at every CI run.
- **CSP defence in depth**: 4 additional headers
  (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`,
  `Permissions-Policy`).
- **Bearer-token rotation** on every login; server-side validation
  on every `/api/v1/*` request.

### Migration from v1.x

If you were on v1.1.0 or earlier:

1. **End-user URLs** keep working. Bookmarks survive.
2. **Legacy HTMX endpoints** (`/api/preview`, `/api/render`,
   `/api/tree`, `/api/backlinks`, `/api/note-excerpt`,
   `/api/command-palette`, `/api/attach`, `/api/upload`,
   `/api/i18n`) are gone (404). Migrate clients to the
   namespaced equivalents under `/api/v1/*`. See
   `docs/migration-v2.md` for the full mapping.
3. **Login flow** is JWT REST POST instead of cookie session.
   Bookmarklets / scripts that POSTed to `/login` should target
   `POST /api/v1/login` and use the returned `token` as
   `Authorization: Bearer <token>`.
4. **`GOSIDIAN_SPA_MODE`** — drop from env / compose. Removed.
5. **Database / vault format** — no migration. SQLite vault
   format is unchanged.
6. **MCP transport at `/mcp/sse`** — unchanged. MCP clients keep
   working without reconfiguration.
7. **Containers** — pull `ghcr.io/daniele-chiappa/gosidian:v2.0.0`,
   recreate the container, no volume changes.

### Notes

- Internal release counterpart: aggregates private `v2.0.0-beta`
  (Phase 0–8 cutover, 2026-05-01) plus 10 post-beta commits
  stabilising the SPA in production. Production deployment date:
  2026-05-05. Soak: 7 days on the beta tag, 3 days on prod, zero
  incident reports.
- New `docs/demo.gif` recorded against the v2.0 SPA via Playwright
  on a synthetic vault, replacing the v1.0 capture.

## [1.1.0] — 2026-04-29 — "Agent workflow + single-port"

Two-bundle MINOR. MCP transport now co-locates with the web UI on a
single port; two new MCP tools land for agent adoption and decoupled
file staging; the upload pipeline gets magic-bytes verification.

### Added

- **`memory_init_agent` MCP tool** — replaces `/init`-style scaffolding
  for Claude Code, Cursor, Codex, Aider, generic. Two modes
  (augment / from-scratch) selected by `existing_content`. New
  package `internal/initprompt/` with renderer + per-profile
  prompts + a single shared `gosidian_block.tmpl.md`.
- **`memory_upload_resource` MCP tool** — pre-uploader twin of
  `memory_upload_attachment`. Same storage and validation, returns
  the resource handle (path, url, mime, kind, size, hash) without
  an embed markdown — agents stage files first, attach later.
- **MCP single-port mode**: SSE transport mounted on the web port at
  `/mcp/sse`. A single SSH tunnel forwarding port 8080 reaches both
  web UI and MCP transport. New client URL:
  `http://<host>:8080/mcp/sse`. The legacy standalone listener
  (`--mcp-addr` / `GOSIDIAN_MCP_ADDR`) is now opt-in (deprecated).
- **`docs/mcp/upload.md`** — single reference for the three
  equivalent upload paths: REST `POST /api/upload`, MCP
  `memory_upload_attachment`, MCP `memory_upload_resource`. Decision
  tree, request/response contracts, MIME tolerance per extension,
  unified error catalogue mapping the same validation failures to
  REST status codes and MCP error results.
- **`docs/examples/docker-compose.image.yml`** — annotated template
  for pull-from-GHCR deploys (no Go toolchain, no `docker login`
  required for the public image, single-port mode by default).

### Changed

- **`/api/upload` co-located with MCP** on the web port. Agents
  behind an SSH tunnel forwarding only 8080 can now reach both the
  REST upload endpoint and the MCP transport through the same
  forward, removing a class of remote-deployment confusion.
- **SQLite engine bumped to 3.53.0** via `modernc.org/sqlite`
  1.32.0 → 1.50.0. Pure correctness margin from upstream fixes
  (64-bit RowID ABI, `Deserialize` memory leak,
  `commitHookTrampoline` signature) — gosidian doesn't use the
  affected APIs but the pull-through is still net-positive.
  Includes new `ColumnInfo` API and sqlite-vec v0.1.9.

### Fixed

- **MCP `/mcp/sse` returned `500 Streaming unsupported`** when
  reached via the shared web mux. Root cause: the metrics middleware
  wrapped the response writer in a struct that did not propagate
  `Flush()` through Go's interface promotion; the SSE handler's
  `w.(http.Flusher)` assertion failed and the stream never opened.
  Fix: the wrapper now forwards `Flush()` and exposes `Unwrap()`
  for `http.NewResponseController` compatibility.
- **`source_path` upload errors hint at the `data` parameter for
  remote setups**. Users running gosidian behind an SSH tunnel hit
  a cryptic "not inside any allowed upload root" — `source_path`
  is resolved server-side, so a client-side path will never match
  the allow-list. Both that error and the file-not-found branch
  now point the caller at base64 `data` as the correct alternative
  for cross-host uploads. Same hint added to the MCP tool
  descriptions so callers see it in the schema.

### Security

- **Magic-bytes verification on uploads** rejects MIME-spoofed
  payloads. `attach.VerifyMIME` inspects the first 512 bytes with
  `http.DetectContentType` and confirms the detected MIME family
  matches the declared extension, with per-extension tolerance
  (SVG/text, DOCX/XLSX as zip containers, PDF/ZIP exact). Catches
  the classic "JS-as-PNG", "HTML-as-PDF", "plain-text-as-ZIP"
  spoof shapes the extension allowlist alone could not stop.

### Deprecated

- `--mcp-addr` flag and `GOSIDIAN_MCP_ADDR` env var. Will be removed
  in a future major release once `/mcp/sse` has been the default
  for at least two minor versions. Existing deployments that keep
  the env var set continue to work and expose both endpoints; new
  deployments should drop the env var and migrate clients to
  `/mcp/sse`.

## [1.0.1] — 2026-04-25 — "Security hardening"

Bundle of security findings raised by the first CodeQL run on the
public repo. No behaviour change for callers using the documented
APIs; only inputs that would have escaped the intended root via
traversal, absolute paths, or null bytes are now rejected up-front.

### Security

- **`internal/trash`**: every public entry point (`DiscardNote`,
  `DiscardProject`, `Restore`, `Purge`) now calls a new `validateName`
  helper before any `filepath.Join`. The guard rejects empty input,
  null bytes, absolute paths (`/foo`, `\foo`), and any `..` component
  on either separator. Defends against path traversal via user-supplied
  ids/names reaching the trash directory or vault root. Closes 10
  CodeQL `go/path-injection` alerts (CWE-22).
- **`internal/server/handlers_login.go`** (`safeNext`): redirect
  validation now also rejects `?next=/\evil.com` (backslash
  protocol-relative URL), in addition to the already-existing reject
  for `?next=//evil.com`. Closes `go/bad-redirect-check` alert.
- **`internal/server/handlers_i18n.go`**: `lang` cookie now sets
  `Secure: true` when the request is over TLS (matches the pattern
  already used by the webauth session cookie). Closes
  `go/cookie-secure-not-set` alert.
- **`internal/audit/audit.go`** (`Log.Write`): `f.Close()` is now
  observed via a deferred closure that promotes a close-time error
  to the function's return value (named return). Prevents silent
  data loss on a failed flush. Closes
  `go/unhandled-writable-file-close` alert.

### Internal

- New regression test `TestBin_RejectsPathTraversal` in
  `internal/trash/trash_test.go` covers eight bad-input shapes
  (empty, `..`, `../etc/passwd`, `foo/../../etc`, absolute
  Linux/Windows paths, `..\windows`, null byte) across all four
  trash entry points.
- `validateName` is intentionally strict: callers that need looser
  semantics (e.g. allowing forward-slash separators in legitimate
  rel paths) sanitize before reaching the trash module.

### Notes

- Zero schema changes, zero breaking changes for third-party users.
- 8 additional CodeQL alerts on `internal/vault/vault.go` were
  dismissed as false positives (path-injection guarded by the
  existing `sanitizeProjectName` helper, which CodeQL does not
  recognize as a sanitizer).

## [1.0.0] — 2026-04-25 — "Initial public release"

First public release of gosidian: a self-hosted, Obsidian-compatible
markdown vault with a first-class MCP server so AI agents get the same
memory you do. Nine internal iterations (v1.0.0 → v1.10.0, ~1 week of
intensive work) stabilised the surface; this release opens the doors.

### Highlights

- **44 MCP tools** covering the full read / write / search /
  discovery / workflow surface — every tool honours ETag optimistic
  locking and token-scoped project filtering.
- **HTMX web UI** (editor, full-text search, graph view, command
  palette, settings, admin) running on the same binary as the MCP
  server.
- **Multi-project vault layout**: `<project>/<note>.md`, with
  cross-project wiki-links, scoped tokens, and a single SQLite FTS5
  index across the whole tree.
- **Agent-first design**: `memory_bootstrap(project)` returns the
  session-start context in one call; ingest patterns (ADR, plan,
  skill, agent, docs) are pre-baked conventions, not free-form.

### Features

**MCP server (44 tools)**

- Core CRUD: `memory_get`, `memory_get_section`, `memory_get_outline`,
  `memory_get_frontmatter`, `memory_create`, `memory_update`,
  `memory_edit`, `memory_append`, `memory_delete`, `memory_batch_get`
- Search & discovery: `memory_search` (cross-project filter),
  `memory_list_notes`, `memory_list_projects`, `memory_list_tags`,
  `memory_notes_by_tag`, `memory_notes_by_importance`, `memory_recent`,
  `memory_backlinks`, `memory_outlinks` (cross-project flag),
  `memory_stale`, `memory_pinned`, `memory_plans`, `memory_skills`
- Workflow: `memory_bootstrap`, `memory_create_handoff`,
  `memory_pending_handoffs`, `memory_compact` (dry-run), `memory_self_stats`,
  `memory_refresh_hot`, `memory_todos`, `memory_lint`, `memory_ask`
- Scaffold: `memory_project_scaffold` (multi-template),
  `memory_list_bootstrap_templates`, `memory_create_project`,
  `memory_delete_project`, `memory_rename_project`, `memory_rename_note`,
  `memory_move_note`
- Attachments: `memory_upload_attachment`, `memory_list_attachments`,
  `memory_delete_attachment`, `memory_attachment_info`
- Audit: `memory_audit_tail`

**Web UI (HTMX)**

- Editor with live preview, full-text search, graph view (Cytoscape
  vendored — no CDN), command palette, theme editor, multi-project
  navigation, settings page.
- Three admin-level theme presets: Midnight Luxury (default), Light
  Clean, High Contrast WCAG-AAA. Custom 5-colour palette supported.
- Language selector in the topbar: IT (complete), EN (reference),
  ES / FR / DE (scaffolding stubs with EN fallback).
- 22 Lucide SVG icons inline, theme-aware via `currentColor`.

**Vault**

- Obsidian-compatible: YAML frontmatter, wiki-links (`[[note]]`,
  `[[project/note|alias]]`), cross-project links.
- Closed tag vocabulary: `type:*`, `status:*`, `topic:*`, `pinned`,
  plus numeric `importance: 1..5`.
- LRU cache for hot-file reads, ETag optimistic locking on all writes,
  synchronous index update after every write.
- Attachments: 6 image types + 7 document types; upload via HTTP or
  MCP base64 / source-path modes.

**Auth & multi-user**

- Web login with `owner` / `member` roles, invite-only signup,
  configurable session TTL + rate-limit window + failure cap.
- MCP bearer tokens scoped by project + scope (`read` / `write` /
  `admin`); token management via web `/admin/tokens`.
- Single-account files auto-migrate to multi-user schema on first
  start.

**Git sync**

- Auto-commit / push of the vault to a Gitea / GitHub remote.
- Graceful fail at boot: if git is unreachable, gosidian starts in
  local-only mode and logs a warning (never fatal).
- Configurable debounce, branch, author, token via env vars.

**Bootstrap templates**

- Three presets ship embedded and are seeded into
  `<vault>/.gosidian/templates/` on first start:
  `karpathy-wiki` (full layout, default), `minimal`, `team`.
- User-editable after seeding; `cp -r` a template, edit, use
  immediately — no rebuild.
- `_template.toml` meta with `name`, `description`, `prompt`,
  `[[variables]]` (required / default / auto=date / auto=project).

**Observability**

- Prometheus metrics on `/metrics` (request counter, latency
  histogram, payload bytes).
- Structured `slog` middleware for MCP calls + HTTP handlers.
- Append-only audit log with per-user filter
  (`memory_audit_tail` tool, `/audit` web page).
- `/healthz` returns `{status, version, vault, notes, git_sync}`.

**Docs**

- `docs/` tree with 19 reference files organised in four areas
  (`mcp/`, `web-ui/`, `vault/`, plus top-level getting-started,
  configuration, deployment, architecture, development, faq).
- 125-line landing README with tagline, 3-command quick start,
  audience blurbs, and a documentation index.
- `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`,
  `PROJECT-STORY.md`, `LICENSE` (MIT).

### Internal

- Single Go binary, multi-stage Docker image (`alpine:3.20` with
  `git` + `ca-certificates`, ~45 MB compressed).
- SQLite via `modernc.org/sqlite` — pure-Go, CGO-disabled, works
  out of the box on Alpine.
- Embedded HTML / CSS / JS + 22 SVG icons via `//go:embed`.
- Full test suite with `-race` detector; tests ship inside the
  repo with `testdata/` fixtures.
- CI pipeline: `go vet` + `go test -race -count=1` + `go build` +
  build & push Docker image to `ghcr.io/daniele-chiappa/gosidian`.

### Notes

- **Public versioning is independent from prior internal versioning.**
  See "Prior internal development" below. v1.0.0 public = snapshot of
  v1.10.0 internal, rebased as "first public release".
- Initial distribution: binary / Docker image on GHCR + source. Docker
  Hub mirroring is reactive (added if concrete demand emerges).
- Zero breaking changes on the MCP tool schemas across the 44 tools —
  SemVer applies from v1.0.0 forward for public consumers.

---

## Prior internal development

gosidian was developed as an internal tool through nine versions before
being published. The internal CHANGELOG is retained below for context
on how the codebase evolved. **Public versioning restarts at v1.0.0**;
the internal `v1.x` line is not part of the public release stream and
does not map onto future public releases one-to-one.

- **v1.10.0** (2026-04-23) — UI polish bundle: Lucide icon set (22
  SVG), three theme presets (Midnight Luxury / Light Clean / High
  Contrast), language selector (IT + EN complete; ES/FR/DE stubs),
  docs split (README 587 → 125 lines, 19 files under `docs/`), removal
  of the "Today" daily-note handler.
- **v1.9.0** (2026-04-22) — Four new MCP tools for agent workflow:
  `memory_todos` (GFM checkbox extraction with plan-status
  enrichment), `memory_lint` (5 baseline rules: broken-wikilink,
  orphan-note, frontmatter-missing, frontmatter-tag-unknown,
  status-incoherent), `memory_ask` (structured OQ-NNN append),
  `memory_search` gained `projects[]` cross-project filter.
- **v1.8.0** (2026-04-22) — Bootstrap templates system.
  `memory_project_scaffold` reads from
  `<vault>/.gosidian/templates/<name>/`; three presets ship embedded
  (`karpathy-wiki`, `minimal`, `team`) and are seeded idempotently.
  New tool `memory_list_bootstrap_templates`; scaffold extended with
  `template` + `variables` parameters.
- **v1.7.2** (2026-04-22) — `SECURITY.md` pointed to a platform-level
  vulnerability reporting flow (GitHub Security Advisories in the
  public version).
- **v1.7.1** (2026-04-22) — Full README rewrite in English with
  installation, configuration, CLI, MCP client wiring, vault layout,
  i18n, development, backup/DR sections. Added `CONTRIBUTING.md`,
  `SECURITY.md`, `CHANGELOG.md`.
- **v1.7.0** (2026-04-22) — Environment-variable coverage extended
  (trash / theme / webauth / vault cache / i18n default). Per-project
  tag filtering. `internal/i18n` package with embedded JSON
  catalogues; `Accept-Language` for MCP clients. `PROJECT-STORY.md`,
  MIT `LICENSE` added.
- **v1.6.0** (2026-04-22) — `memory_project_scaffold` (first
  iteration), `memory_refresh_hot`, `/api/graph?include_cross_project`.
  `auth.Store` hot-reloads `tokens.json` via mtime check — no server
  restart needed after a CLI `token create`. Cytoscape + layout libs
  vendored.
- **v1.5.0** (2026-04-22) — Agent-to-agent handoffs
  (`memory_create_handoff` + `memory_pending_handoffs`).
  `memory_compact` with dry-run. `memory_self_stats` for
  auto-throttling. `memory_outlinks` cross-project flag.
- **v1.4.0** (2026-04-22) — Multi-user web login with owner / member
  roles, invite-only signup, session eviction on user disable.
  `auth.Token.OwnerUserID` with cascade revoke. Per-user `audit.Entry`.
  `auth.json` schema migration v1 → v2 on load.
- **v1.3.0** (2026-04-22) — Frontmatter `importance: 1..5` +
  dedicated `memory_notes_by_importance`. `memory_search` gained
  `include_outline` + `include_frontmatter` flags.
- **v1.2.0** (2026-04-22) — `memory_bootstrap` (single-call
  session-start aggregate). Structured discovery: `memory_plans`,
  `memory_skills`, `memory_pinned`, `memory_stale`. `pinned` tag
  convention.
- **v1.1.0** (2026-04-21) — `/admin/tokens` web page for creating,
  listing, revoking MCP tokens without dropping to the shell.
- **v1.0.0** (internal, 2026-04-21) — First internal tagged release.
  HTTP web UI with HTMX sidebar / editor / search / graph /
  themes; MCP server with 17 tools; ETag optimistic locking; LRU
  vault cache; attachments; audit log; Prometheus metrics;
  structured slog; gitsync with graceful fail; multi-project vault
  layout. SQLite FTS5 as primary datastore.

[1.0.0]: #
