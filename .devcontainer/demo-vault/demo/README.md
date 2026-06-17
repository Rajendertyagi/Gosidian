---
title: Demo Project
description: Index of the sample "demo" project — a Karpathy-Wiki-Stack layout.
tags: [type:index, topic:meta]
type: index
---

# Demo Project

A **project** in gosidian is just a top-level folder. This one shows the
[Karpathy-Wiki-Stack](https://github.com/daniele-chiappa/gosidian)
layout an agent uses to keep durable, cross-session memory.

## Map

| Folder | Purpose | Example |
|---|---|---|
| `memory/` | Stable knowledge: architecture, decisions, glossary | [[demo/memory/architecture]] |
| `plans/` | Plans for non-trivial tasks, with an `Outcome` | [[demo/plans/20260101-search-improvements]] |
| `skills/` | Repeatable procedures | [[demo/skills/run-the-demo]] |
| `notes/` | Free-form notes | [[demo/notes/markdown-showcase]] |
| `media/` | Image notes (first-class) | [[demo/media/architecture-diagram]] |
| `hot.md` | Session cache: current focus, recent decisions | [[demo/hot]] |

## How an agent uses this

At the start of a session an agent reads **[[demo/hot]]** for orientation,
then drills into `memory/` for context. As it works, it writes discoveries
back: a new decision lands in **[[demo/memory/decisions]]**, a new term in
**[[demo/memory/glossary]]**, a repeated procedure becomes a skill like
**[[demo/skills/run-the-demo]]**.

The notes *are* the memory — compiled while working, not an afterthought.

← Back to [[welcome]]
