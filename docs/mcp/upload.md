# Upload flow

gosidian exposes three equivalent ways to put a binary file (image,
PDF, CSV, …) into a vault project under `<project>/attachments/`:

| Strada | Quando preferirla |
|---|---|
| **REST `/api/upload`** | The agent only speaks HTTP. Already used by the web UI editor for drag-and-drop. The single-port mount means the same SSH tunnel that forwards `/mcp/sse` also forwards this endpoint — no extra port to expose. |
| **MCP `memory_upload_attachment`** | Single-step: upload + return a ready-to-splice markdown embed. Best when the agent immediately knows which note the file goes into. |
| **MCP `memory_upload_resource`** | Two-step: upload first, decide note placement later. Best when the agent stages multiple files (`upload all → verify → attach selectively`) or wants to keep the resource orphaned for later GC. |

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
