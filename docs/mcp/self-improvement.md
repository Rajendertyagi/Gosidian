# Self-improvement loop (experimental)

> **Status: experimental, disabled by default.** This is an early
> capability under active evaluation. The knobs and behaviour described
> here may change; fuller documentation will land as it matures.

gosidian can — optionally — collect structured feedback about its *own*
ergonomics from the LLM agents that use it. The idea: agents are the
sensors. When one hits friction while using gosidian itself, it can
record a short, abstract **insight** that lands in a private project for
you to review and fold into the roadmap.

It exists so that an instance you use heavily quietly tells you where the
tool gets in the way. For now it is meant for dogfooding your own
projects rather than as a turnkey feature.

## What you should know

- **Off by default.** With `[self_improve] enabled = false` (the
  default) there is no new behaviour, no prompts, and no insights
  project — gosidian works exactly as before.
- **Opt-in per token.** Even when the master switch is on, only MCP
  tokens explicitly marked as participating ever contribute. Mint one
  with `gosidian token create … --self-improve`.
- **Private-first.** Insights go to a **private** project (owner-only).
  Agents are instructed to describe friction in the abstract — no note
  content, project names, paths, or user data — and the only identity
  recorded is a token's hashed id prefix. Nothing is published
  automatically; sharing a sanitised digest is a separate, manual step.

## Enabling it (if you want to try)

```toml
[self_improve]
enabled = true                 # master switch (default false)
target_project = "insights"    # private project for raw insights
every_n_calls = 25             # nudge cadence per session
```

The full set of knobs and their `GOSIDIAN_SELF_IMPROVE_*` env overrides
is in [Configuration](../configuration.md). Restart gosidian after
editing. Once enabled and a token is opted in, pending insights surface
at session start (`memory_bootstrap`), as an owner-only badge in the web
UI, and — optionally — in a scheduled email digest.

## See also

- [Configuration](../configuration.md) — the `[self_improve]` env/TOML reference
- [Authentication](authentication.md) — MCP tokens and scopes
