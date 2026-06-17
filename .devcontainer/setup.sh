#!/usr/bin/env bash
#
# One-shot provisioning for the gosidian demo Codespace.
#
# Runs once, as the devcontainer `postCreateCommand`. It builds the Vue
# SPA and the Go binary from the checked-out source (so the demo always
# reflects the branch you opened), seeds a disposable demo vault, and
# provisions a single "demo" owner so the web UI is usable immediately.
#
# A login is required because, with zero users, the v2 SPA cannot read
# any data (every /api/v1 route is behind requireAuth). The historical
# "open mode" is not wired in the SPA — see the private bug tracker.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VAULT="$HOME/gosidian-demo-vault"
DEMO_USER="demo"
DEMO_PASS="gosidian-demo"

echo "==> [1/4] Building the Vue 3 SPA (npm ci + vite build)…"
# The @catalogs Vite alias is repo-relative, so a full checkout builds
# without the Dockerfile's catalog-copy dance. `npm run build` already
# embeds the binary's --outDir (internal/server/web/dist).
( cd web && npm ci --no-audit --no-fund && npm run build )

echo "==> [2/4] Compiling the gosidian binary…"
CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -X main.version=codespaces-demo" \
  -o ./gosidian ./cmd/gosidian

echo "==> [3/4] Seeding the demo vault at ${VAULT}…"
rm -rf "$VAULT"
cp -r .devcontainer/demo-vault "$VAULT"

echo "==> [4/4] Provisioning the demo account (${DEMO_USER} / ${DEMO_PASS})…"
printf '%s\n' "$DEMO_PASS" | ./gosidian user setup \
  --vault "$VAULT" --username "$DEMO_USER" --password-stdin

cat <<EOF

============================================================
  gosidian demo is ready.

  It starts automatically and a browser tab will open on
  the forwarded port 8080. Log in with:

      username:  ${DEMO_USER}
      password:  ${DEMO_PASS}

  This is your own private, throwaway instance — edit, break,
  and explore freely. Nothing here is shared or persisted
  beyond this Codespace.
============================================================
EOF
