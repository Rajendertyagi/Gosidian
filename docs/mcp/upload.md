# Upload flow

gosidian exposes several ways to put a binary file (image, PDF, CSV, …)
into a vault project under `<project>/attachments/`:

| Strada | Quando preferirla |
|---|---|
| **HTTP upload (`/upload`)** ⭐ | **Primary for agents.** Multipart upload authenticated by your **MCP bearer token** — the bytes travel over HTTP, never through the model context as base64. The path mirrors your `/sse` endpoint (`/sse` → `/upload`); returns `{path, url, mime, kind, size}`. Pass the returned `path` to `memory_create_media_note` / `memory_create_table_note` as `attachment`, or splice the `url` into a note. |
| **MCP `bridge_filename`** | Co-located deploys: stage the file in `GOSIDIAN_MCP_BRIDGE_DIR` and pass its name; the server reads + consumes it. No HTTP call needed. |
| **MCP `memory_upload_attachment`** | Single-step: upload (base64 `data` / `source_path` / `bridge_filename`) + return a ready-to-splice markdown embed. base64 is costly — prefer the HTTP POST for large files. |
| **MCP `memory_upload_resource`** | Two-step: upload first, decide note placement later. Same sources; same base64 caveat. |
| REST `/api/v1/upload` | Web-UI editor path (drag-and-drop). Authenticated by a **SPA** token (from login), not the MCP token. |

All three converge on the same code path (`internal/attach.Store`):
identical 10 MiB cap, identical extension allowlist, identical
content-addressed filename (`<sha256[:16]>.<ext>`), identical
magic-bytes verification (rejects MIME spoofs).

## Decision tree

```
Are you a Claude Code / Cursor / Zed agent with MCP tools available?
├─ no  → REST /api/upload (multipart, no bearer required by default)
└─ yes
   ├─ Will I attach this to a known note in the next turn?
   │   └─ yes → memory_upload_attachment (returns markdown ready to splice)
   └─ Otherwise (staging, batch, deferred linking)
       └─ memory_upload_resource (returns handle: path, url, hash, mime, kind, size)
```

Agents behind an SSH tunnel that forwards only the web port (e.g.
`ssh -L 58080:server:8080`) reach all three paths through the same
forward — REST at `localhost:58080/api/upload`, MCP at
`localhost:58080/mcp/sse`. This is the deployment shape solved by the
[single-port mode](client-setup.md).

## HTTP upload endpoint (primary)

The cheapest way for an MCP agent to ingest a file: POST the bytes over
HTTP, authenticated by the **same bearer token** used for the SSE stream.
No base64 through the model context, no login, no shared filesystem.

**The path mirrors your `/sse` endpoint** — replace `/sse` with `/upload`:

| Your MCP `/sse` URL | Upload endpoint |
|---|---|
| `https://host/mcp/sse` (single-port web) | `https://host/mcp/upload` |
| `http://host:8765/sse` (legacy listener) | `http://host:8765/upload` |

```bash
# $UPLOAD = your /sse URL with /sse -> /upload
curl -X POST "$UPLOAD?project=Work" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -F "file=@diagram.png"
```

```json
{
  "path": "Work/attachments/3a7b9c4d2e1f5a6b.png",
  "url": "/vault-files/Work/attachments/3a7b9c4d2e1f5a6b.png",
  "mime": "image/png", "kind": "image", "size": 84213,
  "hash": "3a7b9c4d2e1f5a6b"
}
```

- Mounted on the single web port next to `/mcp/sse` — one SSH tunnel
  forwards both. `$BASE` is the same origin you point the MCP client at.
- Enforces the token's **write scope** and **project scope** (the
  `?project=` is intersected with a scoped token's project).
- Same `internal/attach.Store` pipeline: 10 MiB cap, extension
  allowlist, magic-bytes MIME verification.
- **Compose with media notes**: POST the image → take the returned
  `path` → `memory_create_media_note({attachment: path, caption: …})`
  (or `memory_create_table_note` for a CSV).
  The image lives once; agents read only the caption (ADR-013).

## REST `/api/upload`

### Contract

| Field | Value |
|---|---|
| Method | `POST` |
| Path | `/api/upload` |
| Content-Type | `multipart/form-data` |
| Auth | Web session cookie when web auth is enabled (`GOSIDIAN_LOGIN_*`); none otherwise. **No bearer token** — REST is browser-shaped, MCP-side bearer auth is enforced at the SSE handshake on `/mcp/sse`. |
| Size cap | 10 MiB (`attach.MaxBytes`) |

### Query params

| Param | Default | Notes |
|---|---|---|
| `project` | — | **Required.** Vault project name. Empty → `400 project query param is required`. |
| `kind` | `auto` | Informational hint — one of `image`, `document`, `auto`. Echoed back as `kind` in the response. Other values → `400 kind must be one of: image, document, auto`. |

### Body

Single multipart field:

- `file` — the binary payload. The multipart `filename` header
  determines the extension, which must be in the allowlist:
  - **Images**: `.png .jpg .jpeg .gif .webp .svg`
  - **Documents**: `.pdf .csv .json .txt .zip .docx .xlsx`

### Example

```bash
curl -i -X POST \
  -F "file=@/path/to/report.pdf" \
  "http://localhost:8080/api/upload?project=Work&kind=document"
```

Success (`200 OK`, `Content-Type: application/json`):

```json
{
  "path": "Work/attachments/3a7b9c4d2e1f5a6b.pdf",
  "url": "/vault-files/Work/attachments/3a7b9c4d2e1f5a6b.pdf",
  "mime": "application/pdf",
  "kind": "document",
  "size": 124589,
  "original_filename": "report.pdf",
  "hash": "3a7b9c4d2e1f5a6b"
}
```

### Response fields

- `path` — vault-relative location (`<project>/attachments/<hash>.<ext>`).
- `url` — relative URL served by gosidian under `/vault-files/...` with
  one-year immutable cache. Intentionally relative so it resolves
  correctly against whatever host:port the caller used (`localhost`
  via tunnel, `127.0.0.1` direct, public hostname behind reverse proxy).
- `mime` — canonical MIME from the allowlist (not the
  client-declared `Content-Type`, which is ignored).
- `kind` — `image` if the extension is in the image group, `document`
  otherwise. Reflects the allowlist, not the `?kind=` hint.
- `size` — bytes read from the multipart body.
- `original_filename` — preserved for use as link text in markdown
  references.
- `hash` — first 16 hex chars of the payload SHA-256. **Upload is
  idempotent by content**: the same bytes always produce the same path,
  so retries do not duplicate.

## MCP `memory_upload_attachment`

Single-step upload returning a ready-to-splice markdown embed.

```jsonc
{
  "name": "memory_upload_attachment",
  "arguments": {
    "project": "Work",
    "filename": "report.pdf",
    "data": "JVBERi0xLjQK..."   // base64 of file bytes
    // OR: "source_path": "/mnt/uploads/report.pdf"
  }
}
```

Returns:

```json
{
  "path": "Work/attachments/3a7b9c4d2e1f5a6b.pdf",
  "markdown": "[report.pdf](/vault-files/Work/attachments/3a7b9c4d2e1f5a6b.pdf)"
}
```

`markdown` is the embed in canonical form — image notation
`![](url)` for images, link notation `[name](url)` for documents — so
the caller can splice it directly into a `memory_edit` or
`memory_append` body.

Use `data` (base64) for cross-host setups (SSH tunnel, separate
container without shared volume); use `source_path` only when gosidian
and the agent share a filesystem (local install, mounted volume).
`source_path` is validated against
`GOSIDIAN_MCP_ALLOWED_UPLOAD_ROOTS` (vault root always allowed).

### Cheap ingestion: the bridge dir

`data` (base64) routes the file's bytes **through the model context**
(~1 token/char), which makes large images and illustrated notes
impractical to author. When `GOSIDIAN_MCP_BRIDGE_DIR` is set to a
directory shared with the agent's host, stage the file there and pass
**`bridge_filename`** (its basename) to any upload tool or to
`memory_create_media_note`:

```jsonc
{
  "name": "memory_create_media_note",
  "arguments": {
    "project": "Work",
    "bridge_filename": "screenshot.png",   // staged in the bridge dir
    "caption": "Login screen after the redesign"
  }
}
```

The server reads the staged file directly (near-zero token cost), runs
the same magic-byte/MIME validation and 10 MiB cap, and **consumes**
(deletes) the staged copy on success. The bridge dir is automatically an
allowed `source_path` root; a base64 upload larger than ~128 KiB returns
a `hint` redirecting here. Pair it with image **media notes**
(ADR-013): upload once, then reference the media note by link — agents
read only the caption, never the bytes. *Fetch-by-URL ingestion is a
planned follow-up (IMP-059).*

## MCP `memory_upload_resource`

Pre-uploader for the "stage, then attach" pattern.

```jsonc
{
  "name": "memory_upload_resource",
  "arguments": {
    "project": "Work",       // required
    "kind": "document",      // optional hint, default auto
    "filename": "report.pdf",
    "data": "JVBERi0xLjQK..."
  }
}
```

Returns the resource handle, no embed markdown:

```json
{
  "path": "Work/attachments/3a7b9c4d2e1f5a6b.pdf",
  "url": "/vault-files/Work/attachments/3a7b9c4d2e1f5a6b.pdf",
  "mime": "application/pdf",
  "kind": "document",
  "size": 124589,
  "filename": "3a7b9c4d2e1f5a6b.pdf",
  "hash": "3a7b9c4d2e1f5a6b"
}
```

Identical storage and validation to `memory_upload_attachment` — the
difference is purely that the caller decides when and how to embed the
file. Typical pattern: upload N resources, then call `memory_edit` once
with all the embeds composed by hand.

## Errors

All three paths share the same validation pipeline; only the error
shape differs (HTTP `text/plain` body for REST, MCP error result for
the tools).

| Cause | REST status / body | MCP error |
|---|---|---|
| Missing `project` | `400 Bad Request` `project query param is required` | `project must not be empty` (resource only — attachment allows empty) |
| Bad `kind` value | `400 Bad Request` `kind must be one of: image, document, auto` | `kind must be one of: image, document, auto` |
| Unparseable multipart | `400 Bad Request` `bad multipart: <details>` | n/a — MCP encodes args as JSON |
| Missing `file` field | `400 Bad Request` `missing file field: ...` | `provide either data (base64) or source_path` |
| Wrong method | `405 Method Not Allowed` `method not allowed` | n/a |
| File too large (> 10 MiB) | `413 Request Entity Too Large` `file too large (max 10 MiB)` | same message |
| Extension not allowlisted | `415 Unsupported Media Type` `unsupported file type: .exe` | same message |
| Magic-bytes mismatch | `400 Bad Request` `MIME mismatch: declared extension ".png" expects image/png, content detected as text/plain; charset=utf-8` | same message |
| Disk / vault I/O failure | `500 Internal Server Error` `save: <details>` | `<details>` |

The magic-bytes check (added in v1.11) inspects the first 512 bytes
with `http.DetectContentType` and rejects payloads whose detected MIME
family does not match the declared extension. SVG is treated as a
text/XML format, DOCX/XLSX as zip containers — see
`internal/attach/attach.go:VerifyMIME` for the per-extension tolerance
rules.

## Layout on disk

After a successful upload (any path), the file lives at:

```
<vault>/<project>/attachments/<hash>.<ext>
```

For uploads with no project (REST omitting `?project=` is rejected;
the MCP `memory_upload_attachment` allows it):

```
<vault>/attachments/<hash>.<ext>
```

Two callers uploading the same bytes produce the same hash → the same
file → no duplication. To delete an attachment, use
`memory_delete_attachment` (does **not** rewrite notes that reference
it — check `memory_attachment_info` first to find references).

## See also

- [Tool catalogue](tools.md#attachments) — full tool list with
  signatures
- [Client setup](client-setup.md) — the single-port endpoint that
  serves both `/api/upload` and `/mcp/sse` on the web port
- [Authentication](authentication.md) — bearer token scoping for the
  MCP tools (REST does not use bearer auth)
