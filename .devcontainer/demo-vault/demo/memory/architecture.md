---
title: Architecture
description: How the demo project's knowledge is structured and indexed.
tags: [type:memory, topic:architecture]
type: memory
---

# Architecture

gosidian keeps three views over the **same files on disk**:

1. **The vault** — `.md` files in folders. The source of truth.
2. **The index** — a SQLite [[demo/memory/glossary#FTS5]] cache for
   search, tags, and the link graph. Drop it and it rebuilds.
3. **The surfaces** — the web UI (humans) and the MCP server (agents),
   both reading and writing the vault.

## The link graph

Every `[[wikilink]]` becomes an edge. That graph powers:

- **Backlinks** — who points at this note (see [[demo/memory/glossary#Backlink]]).
- **The graph view** — a visual map of the vault.
- **Graph analytics** — e.g. finding the most-linked "hub" notes.

This note, for instance, is reachable from [[demo/README]] and
[[demo/hot]], and it links onward to the [[demo/memory/glossary]].

## Why plain Markdown

No lock-in. Open the same folder in Obsidian, VS Code, or `vim`. The
decision is recorded in [[demo/memory/decisions#ADR-001]].

← Back to [[demo/README]]
