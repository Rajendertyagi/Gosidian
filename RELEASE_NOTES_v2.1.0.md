# gosidian v2.1.0 — Lint vocabulary extension

MINOR release. Extends the `frontmatter-tag-unknown` lint rule with
a per-vault config-driven allow-list. Vaults document their own
structural tag patterns without weakening the rule for everyone.

## What changed

### For users running `memory_lint`

If your vault uses tags that are part of your structural conventions
but fall outside the built-in vocabulary (`type:` / `topic:` /
`status:` / `pinned` / project-name), you can now declare them once
in `<vault>/.gosidian/config.toml` instead of accumulating warnings
on every lint run:

```toml
[lint.frontmatter_tag_vocabulary]
extra_allowed = [
  "status:reference",
  "topic:agent-template",
  "topic:bootstrap",
]
```

Each entry is either a `<namespace>:<value>` pair or a bare token.
The list is **additive**: built-in tags are still allowed, and the
rule still flags anything outside the combined vocabulary.

Malformed entries (empty, leading/trailing colon, internal
whitespace, double colon) are skipped silently at load time so a
typo in the config does not crash the lint.

### For developers extending gosidian

- `lint.Linter.WithExtraAllowedTags([]string)` — chainable setter
  installs a per-instance extension; `isKnownTag` is now a method.
- `mcp.Server.SetLintExtraAllowedTags([]string)` — per-server setter
  the binary wires from `cfg.Lint.FrontmatterTagVocabulary.ExtraAllowed`
  at startup.

## Migration

**No migration required.** Vaults without a `[lint]` section keep
the same behaviour they had on v2.0.x. This is a purely additive
feature.

To start using the extension on an existing vault, add a `[lint]`
section to `.gosidian/config.toml` and restart gosidian.

## Quick start

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v2.1.0
# Same volume + container shape as v2.0.x:
docker run -d --name gosidian -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v2.1.0
```

## Verification

- `go vet ./...`, `go test ./...` green across all 16 packages
- 3 new unit tests in `internal/lint/lint_test.go` cover the
  extension behaviour end-to-end
- `npm audit --omit=dev` = 0 vulnerabilities (unchanged from v2.0.1)
- Vitest 16/16 green
- `vite build` produces a byte-identical SPA `dist/` (the SPA bundle
  is not affected by this release)

## Closes

- **OQ-004** — open-question on how to extend the closed tag
  vocabulary, opened 2026-04-22.
