---
title: Glossary
description: Key terms used across the demo project.
tags: [type:memory, topic:meta]
type: memory
---

# Glossary

### Backlink

The reverse of a `[[wikilink]]`: every note shows which other notes point
*at* it. Try it — this term is referenced from
[[demo/memory/architecture]]. Open this note and check its backlinks panel.

### FTS5

SQLite's full-text search extension. gosidian uses it to make vault-wide
search instant. The index is a cache over the `.md` files; see
[[demo/memory/architecture]].

### Karpathy-Wiki-Stack

An opinionated project layout (`memory/`, `plans/`, `skills/`, `hot.md`,
`log.md`) where an agent maintains a wiki of compiled knowledge as it
works. The whole [[demo/README]] is an example of it.

### MCP

The Model Context Protocol. gosidian exposes ~50 typed tools so an agent
can bootstrap, search, read, write, link, and self-check against the
vault — over the same files humans edit.

### Media note

A regular `.md` note whose frontmatter declares `type: image` and a
`media:` pointer to an attachment. It renders as an image but behaves like
any note in search and the graph. Live example:
[[demo/media/architecture-diagram]].

← Back to [[demo/README]]
