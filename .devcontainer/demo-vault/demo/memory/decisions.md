---
title: Decision Log (ADRs)
description: Architectural decisions for the demo project.
tags: [type:memory, topic:architecture]
type: memory
---

# Decision Log

Short, append-only architectural decision records. Each one captures a
binding choice and *why*, so future sessions don't re-litigate it.

## ADR-001 — Plain Markdown is the source of truth

**Decision.** Notes are stored as plain `.md` files. The SQLite index is
a rebuildable cache, never the system of record.

**Why.** Zero lock-in and full interoperability: the same vault opens in
Obsidian, VS Code, or any editor. If the index is lost, it rebuilds from
the files. See [[demo/memory/architecture]].

## ADR-002 — Links are first-class

**Decision.** `[[wikilinks]]` define the knowledge graph; backlinks and
the graph view are derived from them, not maintained by hand.

**Why.** It keeps navigation honest — structure emerges from the writing.
Terms like [[demo/memory/glossary#Backlink]] stay connected automatically.

← Back to [[demo/README]]
