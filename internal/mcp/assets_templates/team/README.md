---
title: {{PROJECT}} Project Index
description: Landing page for the persistent memory of the {{PROJECT}} project.
tags: [{{PROJECT}}, type:index, topic:meta]
type: index
updated: {{TODAY}}
---

# {{PROJECT}} — Project Memory

Persistent memory for the **{{PROJECT}}** project. Karpathy-Wiki-Stack pattern (stable, but updated incrementally).

## Map

| Folder | Purpose |
|---|---|
| `memory/` | Stable knowledge (architecture, decisions, conventions, glossary, environments) |
| `agents/` | Active specialised roles |
| `plans/` | Plans for non-trivial tasks, `YYYYMMDD-<slug>.md`, with a post-execution `Outcome` |
| `skills/` | Repeatable procedures |
| `docs/` | Q&A, open questions, improvements, bug tracker |
| `hot.md` | Session cache |
| `log.md` | Append-only log |

## Session bootstrap

1. `memory_bootstrap({project: "{{PROJECT}}"})` — full aggregate in one call
2. If needed, `memory_plans(project, status:"in-progress")` + `memory_skills(project)`
3. For focused tasks, targeted reads via `memory_get`
