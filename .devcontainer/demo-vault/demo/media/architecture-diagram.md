---
title: Architecture diagram
type: image
media: demo/attachments/architecture.svg
tags: [type:image, topic:architecture]
---

The gosidian model in one picture: a single **vault** of `.md` files, a
rebuildable **SQLite index** over it, and two surfaces — the **web UI**
for humans and the **MCP server** for agents.

This is a **media note**: a normal `.md` file whose frontmatter declares
`type: image` and a `media:` pointer. It renders as an image, but the body
you're reading is the **caption** — it goes into full-text search and is
what an agent retrieves (the image bytes do not). It participates in tags,
links, and the graph like any other note. See
[[demo/memory/glossary#Media note]] and [[demo/memory/architecture]].

← Back to [[demo/README]]
