# gosidian v1.1.0 — Agent workflow + single-port

Two themes in this release: agents get a richer set of MCP tools for
memory bootstrap and decoupled file staging, and operators get a
single-port deployment shape that simplifies remote / tunneled setups.

## What changed

### For agents using gosidian as memory layer

- **`memory_init_agent`** replaces ad-hoc `/init` scaffolding for the
  five built-in agent profiles (Claude Code, Cursor, Codex, Aider,
  generic). Pick *augment* to merge gosidian's memory block into
  output the agent's native init produced; pick *from-scratch* when
  there is no init to extend. See [docs/mcp/tools.md](docs/mcp/tools.md).
- **`memory_upload_resource`** is the pre-uploader twin of
  `memory_upload_attachment`. Use it when you stage multiple files
  before deciding which note each one attaches to. The full upload
  flow — REST and MCP, same storage path — is now documented in
  [docs/mcp/upload.md](docs/mcp/upload.md), including a unified error
  catalogue mapping the same validation failures to REST status
  codes and MCP error results.

### For operators

- **Single-port mode is now the default**: the MCP SSE transport is
  mounted on the web port at `/mcp/sse`. New client URL:
  `http://<host>:8080/mcp/sse`. The legacy standalone listener on
  port 8765 is deprecated but still supported for backward
  compatibility (set `GOSIDIAN_MCP_ADDR` to enable it).
- An SSH tunnel forwarding port 8080 alone now reaches the web UI,
  the `/api/upload` REST endpoint, and the MCP transport. Drop the
  second `-L` from your SSH commands.
- A new compose template under
  [`docs/examples/docker-compose.image.yml`](docs/examples/docker-compose.image.yml)
  shows the recommended pull-from-GHCR deployment shape: anonymous
  pull, no Go toolchain, single-port mode by default.

### For everyone

- The upload pipeline now performs **magic-bytes verification**: a
  payload declared as `.png` but containing JS / HTML / random text
  is rejected with `MIME mismatch`. Catches spoofed uploads the
  extension allowlist alone could not stop. Per-extension tolerance
  rules keep legitimate edge cases (SVG as text/XML, DOCX/XLSX as
  zip containers) accepted.
- **SQLite engine bumped to 3.53.0** via `modernc.org/sqlite`
  1.32.0 → 1.50.0. Pure correctness margin from upstream fixes
  (64-bit RowID ABI, `Deserialize` memory leak,
  `commitHookTrampoline` signature) — gosidian doesn't call the
  affected APIs but the pull-through is still net-positive.

## Upgrade notes

- **No breaking changes for existing clients.** Clients pinned to
  the legacy `:8765/sse` URL keep working as long as
  `GOSIDIAN_MCP_ADDR` is set in your env.
- **Recommended migration**:
  1. Change client URL to `http://<host>:8080/mcp/sse`.
  2. Drop the `8765:8765` port binding from your compose / deployment.
  3. Unset `GOSIDIAN_MCP_ADDR` to silence the boot-time deprecation
     warning.
- **Database compatibility**: SQLite 3.53.0 reads / writes 3.x
  databases unchanged. No schema migration required.
- **Source-path uploads behind a tunnel**: if you previously hit
  "not inside any allowed upload root" when calling
  `memory_upload_attachment` / `memory_upload_resource` with
  `source_path` from a remote client, the error message now points
  you at base64 `data` as the correct parameter for cross-host
  setups. The fix is purely UX — the underlying constraint
  (`source_path` is resolved server-side) is unchanged.

## Quick start (single-port)

```bash
docker run -d \
  --name gosidian \
  -p 8080:8080 \
  -v "$(pwd)/vault:/vault" \
  ghcr.io/daniele-chiappa/gosidian:v1.1.0
```

Then connect your agent to `http://localhost:8080/mcp/sse` with a
bearer token from `/admin/tokens`. See
[docs/mcp/client-setup.md](docs/mcp/client-setup.md) for
client-specific setup (Claude Code, Zed, Cursor, Continue).

## Acknowledgements

- The single-port issue was diagnosed end-to-end through an SSH
  tunnel using a vault-shared diagnostic note pattern between two
  Claude Code instances — a process that turned a hard-to-reproduce
  remote bug (`500 Streaming unsupported`) into a localized
  `*statusRecorder` `Flush()` forwarding fix in under an hour.
