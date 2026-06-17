#!/usr/bin/env bash
#
# Launches the gosidian demo server, as the devcontainer
# `postStartCommand` (runs on every Codespace start, including resumes).
# Idempotent: exits early if the server is already serving.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if curl -sf http://localhost:8080/healthz >/dev/null 2>&1; then
  echo "gosidian already running on :8080"
  exit 0
fi

export GOSIDIAN_VAULT="$HOME/gosidian-demo-vault"
export GOSIDIAN_ADDR=":8080"
export GOSIDIAN_MCP_ADDR=""               # MCP off: the demo is web-only
export GOSIDIAN_I18N_DEFAULT_LANG="en"    # public README is English
export GOSIDIAN_VAULT_HTML_NOTES="true"   # render the .html demo note
export GOSIDIAN_VAULT_MEDIA_NOTES="true"  # render the media demo note

# Detach into a new session with setsid. A plain `&` child is reaped when
# the postStartCommand process group is torn down, so the server would die
# before it ever binds :8080 (manifesting as a 502 on the forwarded port).
# setsid reparents it to init so it survives the lifecycle hook's exit.
setsid ./gosidian >"$HOME/gosidian-demo.log" 2>&1 </dev/null &

# Best-effort readiness wait so the creation log shows a clear outcome.
for _ in $(seq 1 20); do
  if curl -sf http://localhost:8080/healthz >/dev/null 2>&1; then
    echo "gosidian is up on :8080 — log in with demo / gosidian-demo"
    exit 0
  fi
  sleep 1
done

echo "gosidian did not become healthy in time; see ~/gosidian-demo.log" >&2
tail -n 20 "$HOME/gosidian-demo.log" >&2 || true
exit 0
