# gosidian v2.2.0 — auth & roles

MINOR release. gosidian's multi-user web login grows up: **role-based
access control**, **TOTP two-factor**, and **LDAP / Active Directory**
login, plus a graph view that respects who's looking. Fully backward
compatible — TOTP defaults to `off`, LDAP to disabled, and existing
setups need no migration.

## What changed

### Role-based access control

Three roles, in decreasing privilege: **owner → member → guest**.

- **owner** — full admin: users, invites, roles, settings, plus
  read/write on every project.
- **member** — read/write all projects and create MCP tokens; no admin.
- **guest** — read-only, and only on projects flagged **public**.

Enforcement is centralized and **fail-closed** (`internal/authz`): a
read the role may not perform returns **404** (existence-hiding), a
forbidden write returns **403**, and an unknown role degrades to
public-read only.

### Public / private projects

Every project now has a `public` flag (default **private**). Public
projects are visible read-only to guests; private projects only to
owners and members. "Public" means *visible to any signed-in user* — not
anonymous. Guests are filtered everywhere: sidebar, search, tags, note
titles, and the graph. This is what lets you hand a stakeholder a guest
account that sees exactly the projects you choose, and nothing else.

### TOTP two-factor

- **Global mode** (`off` / `optional` / `required`), set from
  `/settings` or `GOSIDIAN_TOTP_MODE`.
- **Per-user override** (inherit / enabled / disabled) in **Admin →
  Users**.
- **Self-service enrolment** with confirm-before-activate — a mistyped
  setup can't lock the account.
- `off` is a deliberate **master switch**: flip it and 2FA enforcement
  stops immediately, so a misconfiguration can never lock the team out.

### LDAP / Active Directory login

- **Search-then-bind** against an external directory; on the first
  successful login gosidian **auto-provisions a local guest account**
  (no password stored). An owner can promote it to member.
- **Local accounts always win** — an existing local username is never
  checked against the directory, so the bootstrap owner keeps working
  even if the directory is down.
- **LDAPS** and **StartTLS** supported, with a configurable user filter:
  OpenLDAP `(uid=%s)`, Active Directory `(sAMAccountName=%s)`.

See [Authentication & roles](docs/web-ui/authentication.md) for the full
setup, and `deploy/ldap-test/` for a throwaway OpenLDAP harness.

### Graph

The graph view now honours per-role visibility and opens on the most
recently edited project you can see, rather than loading the whole vault
at once — much friendlier for accounts with many projects.

### Dependencies

- `modernc.org/sqlite` 1.51.0 → 1.52.0.

## For end users

Nothing breaks. If you do nothing:

- Existing owner / member accounts keep working exactly as before.
- TOTP stays **off** and LDAP stays **disabled** until you turn them on.
- All existing projects remain **private** (owner/member only).

To adopt the new features: flip a project to public from **Projects**,
set the TOTP mode in **Settings**, or configure `[ldap]` (see the docs).

## Upgrade notes

- No schema migration. Drop-in replacement for v2.1.x.
- LDAP is validated end-to-end against OpenLDAP over plain LDAP, LDAPS,
  and StartTLS. The Active Directory path is configuration-only on the
  same validated code — point it at your domain controller with
  `user_filter = "(sAMAccountName=%s)"` and an encrypted transport.

## Verifying the image

```bash
docker pull ghcr.io/daniele-chiappa/gosidian:v2.2.0
```
