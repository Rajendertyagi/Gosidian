# MCP client setup

Connect any MCP-compatible client to the SSE endpoint, passing the
token as a bearer header. Language preference uses the standard
`Accept-Language` header.

The default endpoint is **`/mcp/sse` on the web port** (single-port
mode). The legacy standalone listener at `/sse` on a separate port
(typically 8765) is still supported for backward compatibility but is
deprecated — see the migration note at the bottom of this page.

## Generic JSON config

```json
{
  "mcpServers": {
    "gosidian": {
      "url": "http://127.0.0.1:8080/mcp/sse",
      "transport": "sse",
      "headers": {
        "Authorization": "Bearer gosidian_XXXXXXXXXXXXXXXXXXXXXXXX",
        "Accept-Language": "en"
      }
    }
  }
}
```

Replace `127.0.0.1` with your server hostname and the token with the
plaintext printed by `gosidian token create`.

## Claude Code

```bash
claude mcp add gosidian http://127.0.0.1:8080/mcp/sse \
  --transport sse \
  --header "Authorization: Bearer gosidian_XXXXXXXXXXXXXXXXXXXXXXXX"
```

After restart, the tools appear under `gosidian__memory_*` and are
callable from any conversation. See
[Agent patterns](patterns.md) for the recommended session opening.

## Zed

Add to your `settings.json`:

```json
{
  "context_servers": {
    "gosidian": {
      "source": "custom",
      "command": {
        "type": "sse",
        "url": "http://127.0.0.1:8080/mcp/sse",
        "headers": {
          "Authorization": "Bearer gosidian_..."
        }
      }
    }
  }
}
```

## Cursor / Continue / other clients

Any MCP-compatible client that supports the SSE transport with
custom headers works. Typical configuration fields:

- `url` — `http://<host>:<port>/mcp/sse` (single-port; recommended)
  or the legacy `http://<host>:<port>/sse` when the standalone
  listener is enabled
- `transport` — `"sse"`
- `headers.Authorization` — `Bearer <plaintext>`
- `headers.Accept-Language` — optional; `en`, `it`, `es`, `fr`, `de`
  available in v1.10

## Custom clients

If you build your own MCP client:

- SSE contract follows the standard MCP spec — gosidian adds no
  custom framing.
- Errors are JSON objects with `error.code` and `error.message` from
  the [internationalized error catalogue](../../internal/i18n/catalogs/errors.en.json).
- Responses include an `etag` field on every read tool; pass it back
  as `if_match` on the matching write to get optimistic locking.
- Check `memory_self_stats()` for the token's rate-limit headroom
  before doing anything aggressive.

## Verification

Once connected, every client should see 55 tools in `tools/list`. A
good smoke sequence for your first call:

```
memory_bootstrap(project="<name-of-a-project>")
```

A healthy response contains `hot_md_content`, `readme_content`,
`active_plans`, `available_skills`, `recent_notes`, and
`project_stats`. If it comes back with an auth error, the bearer
token is wrong; with an empty project list, your vault has no
top-level folders yet (see
[Agent patterns → Bootstrap a project](patterns.md#bootstrap-a-new-project)).

## Migrating from the legacy standalone port

Versions before the single-port change exposed MCP on its own port
(typically 8765) at path `/sse`. That deployment shape is still
supported when `--mcp-addr` / `GOSIDIAN_MCP_ADDR` is set, but is
deprecated. To migrate:

1. **Update the client URL** from `http://<host>:<legacy-port>/sse` to
   `http://<host>:<web-port>/mcp/sse`. Bearer header unchanged.
2. **Drop the second port mapping** from your Docker / compose config
   (the line that bound `8765:8765`).
3. **Unset `GOSIDIAN_MCP_ADDR`** to silence the deprecation warning at
   boot. The standalone listener will not start; clients must use the
   web-port path.

The motivation: a single tunnel (SSH `-L 8080`, reverse proxy, or any
other single-port forwarder) now serves both the web UI and the agent
transport. This removes a class of remote-deployment misconfiguration
where clients reached the web port for `/api/upload` but not the
agent port for MCP.
