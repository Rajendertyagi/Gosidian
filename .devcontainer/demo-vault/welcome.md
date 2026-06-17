---
title: Welcome to gosidian
description: Start here — a 2-minute tour of this live demo.
tags: [type:index, topic:meta]
type: index
---

# Welcome to gosidian 👋

You're looking at a **live, throwaway gosidian instance** running just for
you. Edit anything, break anything — it resets when this Codespace ends.

> gosidian is a markdown vault with a built-in **MCP server** and a
> **web UI**. Humans edit here; AI agents read and write the same `.md`
> files over MCP. Everything is plain Markdown on disk.

## Take the tour

- **[[demo/README]]** — a sample project laid out the gosidian way
  (the "Karpathy-Wiki-Stack"). This is how an agent keeps its memory.
- **[[demo/notes/markdown-showcase]]** — what the editor and live
  preview can render.
- **[[demo/media/architecture-diagram]]** — a *media note*: an image
  that participates in search, tags, and the graph like any other note.
- **[[demo/interactive/lifecycle]]** — an *HTML note*, rendered live in
  a sandboxed iframe.

## Things to try

1. Open the **graph view** (sidebar) and watch this note connect to the
   rest of the vault through `[[wikilinks]]`.
2. Hit **search** (the command palette) and look for *backlink* or
   *MCP* — full-text search is instant (SQLite FTS5).
3. Open **[[demo/memory/glossary]]** and follow a backlink back here.
4. Edit this file and watch the preview update live.

When you're done, the real thing is one `docker run` away — see the
project README on GitHub.
