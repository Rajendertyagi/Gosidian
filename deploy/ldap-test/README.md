# LDAP test harness

A throwaway OpenLDAP directory for exercising gosidian's LDAP login end to end.

## 1. Start the directory + seed users

```bash
docker compose -f deploy/ldap-test/docker-compose.yml up -d
./deploy/ldap-test/seed.sh   # ldapadd users alice / bob
```

Creates two users under `ou=users,dc=example,dc=org`:

| username | password   |
|----------|------------|
| `alice`  | `alicepass`|
| `bob`    | `bobpass`  |

Service account: `cn=admin,dc=example,dc=org` / `adminpassword`. LDAP listens on
`ldap://127.0.0.1:1389` (and `ldaps://127.0.0.1:1636`).

## 2. Point gosidian at it

Either in `<vault>/.gosidian/config.toml`:

```toml
[ldap]
enabled       = true
url           = "ldap://127.0.0.1:1389"
bind_dn       = "cn=admin,dc=example,dc=org"
bind_password = "adminpassword"          # prefer the env var below for secrets
user_base_dn  = "ou=users,dc=example,dc=org"
user_filter   = "(uid=%s)"
```

…or via environment (handy for Docker / secrets — overrides the file):

```
GOSIDIAN_LDAP_ENABLED=true
GOSIDIAN_LDAP_URL=ldap://127.0.0.1:1389
GOSIDIAN_LDAP_BIND_DN=cn=admin,dc=example,dc=org
GOSIDIAN_LDAP_BIND_PASSWORD=adminpassword
GOSIDIAN_LDAP_USER_BASE_DN=ou=users,dc=example,dc=org
GOSIDIAN_LDAP_USER_FILTER=(uid=%s)
```

Restart gosidian — the boot log prints `ldap: enabled (...)`.

## 3. Test

Log in to the SPA as `alice` / `alicepass`. On first login gosidian
**auto-provisions a local account with role `guest`** (`auth_source: ldap`,
no password hash) — the guest sees only projects flagged *public*. An owner can
promote the user to `member` from **Admin → Users**. A local username always
takes precedence over LDAP (an existing local account is never checked against
the directory).

## TLS (LDAPS / StartTLS)

The OpenLDAP container also listens on `ldaps://127.0.0.1:1636` and supports
StartTLS on `:1389`. To exercise gosidian's encrypted paths, swap the URL and
add the TLS flags (the self-signed test cert needs `insecure_skip_verify`):

```
# LDAPS
GOSIDIAN_LDAP_URL=ldaps://127.0.0.1:1636
GOSIDIAN_LDAP_INSECURE_SKIP_VERIFY=true
# …or StartTLS over the plain port
GOSIDIAN_LDAP_URL=ldap://127.0.0.1:1389
GOSIDIAN_LDAP_START_TLS=true
GOSIDIAN_LDAP_INSECURE_SKIP_VERIFY=true
```

> **Gotcha:** osixia defaults `LDAP_TLS_VERIFY_CLIENT=demand`, which makes slapd
> ask every TLS client for a certificate and drop the handshake (`Network
> Error: EOF`) when a search-then-bind client doesn't present one. The compose
> sets it to `never`. `insecure_skip_verify` is **dev-only** — in production use
> a real CA-signed cert and leave it off.

## Active Directory

The flow is identical; only the filter/attributes differ. For AD, set:

```toml
user_filter  = "(sAMAccountName=%s)"
user_base_dn = "OU=Users,DC=corp,DC=example,DC=com"
bind_dn      = "CN=svc-gosidian,OU=ServiceAccounts,DC=corp,DC=example,DC=com"
```

For an AD test directory use the Samba AD DC compose (`docker-compose.ad.yml`)
instead of OpenLDAP — it provisions a real domain (sAMAccountName, Kerberos).
AD typically requires an encrypted transport for binds, so combine the
`(sAMAccountName=%s)` filter above with the LDAPS/StartTLS flags from the TLS
section.

> **Nested-LXC caveat:** `samba-tool domain provision` panics with *"Security
> context active token stack underflow!"* inside an unprivileged/nested Proxmox
> LXC (confirmed on Samba 4.15 and 4.24) — a kernel-namespace limitation, not a
> config error. Run `docker-compose.ad.yml` on a bare-metal/VM Docker host, or
> validate AD mode directly against the target domain controller.

## Teardown

```bash
docker compose -f deploy/ldap-test/docker-compose.yml down -v
```
