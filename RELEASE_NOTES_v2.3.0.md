# gosidian v2.3.0 — plancia

MINOR release. The web UI becomes a **tiling window manager**, and two
opt-in features land — **global projects** and an experimental
**self-improvement loop**. Everything is backward compatible and the new
subsystems are **off by default**; existing setups need no migration.

## What changed

### The plancia — a window-manager UI

The web UI is now a **"plancia"**: a niri-style scrollable tiling
workspace. Instead of one page at a time, notes, the graph, search, and
config forms open as **windows** side by side.

- **Open** from the sidebar tree, a wikilink, search, or the command
  palette (`⌘K`); each window opens to the right of the focused one.
- **Resize** in discrete steps — small → medium → full.
- **Minimize** to a horizontally-scrollable footer; click to restore.
- **Direct links**: a window's link button opens the *ego-graph* — the
  one-hop neighbourhood of that note — as its own window.
- **Edit in place**: a note opens in view mode with a View/Edit toggle;
  the editor mounts in the same window (lazily, and hidden for read-only
  users) — no second page.
- **Navigate** focus with `Alt-←` / `Alt-→`.

Open windows + focus are encoded in the URL (`?w=…&f=…`), so a workspace
is shareable and survives reload; an empty URL restores your last
workspace from `localStorage`. The app menu moved into a collapsible
sidebar section. Deep-link routes (`/notes/<path>`, `/graph`, …) still
work — visiting one opens the matching window.

### Global projects (opt-in, off by default)

Two optional shared projects — `global` (public) and `global-private` —
hold **skills, agents, and scaffold templates** that any project can
reuse, with **local-overrides-global** merge semantics. Enable with
`GOSIDIAN_GLOBAL_ENABLED=true` and opt a project in via its `use_globals`
flag. A `memory_global_check` MCP tool reports which `global-private`
notes a project references, for private→public promotion. See
[docs/vault/global-projects.md](docs/vault/global-projects.md).

### Self-improvement loop (experimental, off by default)

Agents can record structured **usage-friction insights** via the
`memory_self_improve` MCP tool; they land in a private `insights` project
for the owner to triage. Private-first and off by default — opt in per
MCP token. See [docs/mcp/self-improvement.md](docs/mcp/self-improvement.md).

### Documentation correction

The web UI has been a **Vue 3 SPA** since v2.0, but several docs still
described the old HTMX stack. The README, web-UI overview, FAQ, and
architecture pages are now accurate (and document the plancia).

### Other

- Two new MCP tools (`memory_self_improve`, `memory_global_check`) — 48
  total.
- `modernc.org/sqlite` updated to 1.52.0.

## Upgrade notes

- **No migration required.** Global projects and the self-improvement
  loop are off until you enable them; the vault format is unchanged.
- The Docker image (`ghcr.io/daniele-chiappa/gosidian:v2.3.0`) is the
  recommended artifact. Existing bookmarks and shared note URLs keep
  working.
- The plancia is desktop-first; on narrow viewports it falls back to one
  full-width window at a time.

## Acknowledgements

Thanks to everyone who kicked the tires on the auth release and asked
for a faster way to keep several notes and the graph in view at once —
that's exactly what the plancia is for.
