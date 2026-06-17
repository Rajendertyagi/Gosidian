---
title: Markdown showcase
description: A quick render test for the editor and live preview.
tags: [type:doc, topic:webui]
type: doc
---

# Markdown showcase

A grab-bag of what the editor renders. Open this note and toggle the live
preview.

## Text

**Bold**, *italic*, ~~strikethrough~~, `inline code`, and a
[link to the docs](https://github.com/daniele-chiappa/gosidian). Internal
links work too: [[demo/memory/glossary]].

## Lists & tasks

- A bullet
  - A nested bullet
- [x] A completed task
- [ ] An open task

## Quote & code

> Notes are plain Markdown. The tools are the wiki you maintain while you
> work.

```go
func Hello(name string) string {
    return fmt.Sprintf("hello, %s", name)
}
```

## Table

| Surface | Audience | Protocol |
|---|---|---|
| Web UI  | Humans   | HTTP     |
| MCP     | Agents   | MCP/SSE  |

## Math & more

Inline `E = mc^2`, footnotes, callouts, and an embedded image below — all
the usual Markdown, plus first-class [[demo/media/architecture-diagram]]
image notes.

← Back to [[demo/README]]
