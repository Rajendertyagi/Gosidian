# gosidian v1.0.1 — Security hardening

Patch release that addresses the security findings raised by the
first CodeQL run on the public repo (~24 hours after `v1.0.0`).
**No behaviour change for documented APIs**; only inputs that would
have escaped the intended root via traversal, absolute paths, or
null bytes are now rejected up-front.

## What changed

### Trash bin path-injection guards (`internal/trash`)

Every public entry point — `DiscardNote`, `DiscardProject`,
`Restore`, `Purge` — now calls a new `validateName` helper before
any `filepath.Join`. The guard rejects:

- empty input
- absolute paths (`/foo`, `\foo`)
- any `..` component (after normalising backslashes to slashes)
- null bytes

The trash bin is opt-in (`[trash]` in `config.toml`) and the API
shape is unchanged. Callers that already pass clean rel-paths see
no difference. Closes 10 CodeQL `go/path-injection` alerts
([CWE-22][cwe22]).

### Login redirect (`safeNext`)

The post-login `?next=` validator already rejected `?next=//evil`
protocol-relative URLs. It now also rejects `?next=/\evil` (backslash
protocol-relative), which some browsers normalise into a host. Closes
the `go/bad-redirect-check` alert.

### i18n cookie

The `lang` UI-preference cookie now sets `Secure: true` when the
request is TLS (using the same `webauth.IsSecureRequest` helper
already used by the session cookie). On plain-HTTP development
servers the flag stays off. Closes `go/cookie-secure-not-set`.

### Audit log write

`audit.Log.Write` previously did `defer f.Close()` and discarded any
close-time error. The fix uses a named return + deferred closure
that promotes a `Close()` error to the function's return value
unless an earlier write already failed. Prevents silent data loss
on a failed flush. Closes `go/unhandled-writable-file-close`.

## Notes on dismissed alerts

8 additional CodeQL `go/path-injection` alerts on
`internal/vault/vault.go` were dismissed as false positives: those
paths are preceded by `sanitizeProjectName(name)`, which rejects
unsafe input — CodeQL just doesn't recognise rejection-based
sanitizers. The dismissal carries an explicit comment in the GitHub
Security tab pointing to the helper.

## Upgrade

Pure drop-in:

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v1.0.1
# or
docker pull ghcr.io/daniele-chiappa/gosidian:latest
```

No config migration, no schema change.

## Acknowledgements

CodeQL — for catching the four real findings above and prompting
the explicit `validateName` guard which is now first-class API
surface in `internal/trash`.

[cwe22]: https://cwe.mitre.org/data/definitions/22.html
