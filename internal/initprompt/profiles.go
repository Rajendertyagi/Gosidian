// Package initprompt renders the prompt-guide + gosidian_block payload
// returned by the memory_init_agent MCP tool.
//
// Design: the tool does NOT scan the caller's filesystem (the server has
// no view of the agent's cwd). Instead it produces a multi-section prompt
// that the agent itself executes, plus a parametric markdown block the
// agent innests into the agent-native instruction file (CLAUDE.md /
// AGENTS.md / .cursor/rules.mdc / …). Filename resolution is delegated to
// the agent; the tool only surfaces optional hints.
//
// Two modes:
//   - ModeAugment (preferred): the agent has already run its native /init
//     and passes the produced content via existing_content. The prompt
//     instructs a merge that preserves existing sections.
//   - ModeFromScratch (fallback): no existing content. The prompt includes
//     cwd-scan instructions so the agent can synthesize a new file.
package initprompt

// Profile identifies the target AI agent. It only influences prompt tone /
// tool references (Claude's AskUserQuestion vs. a generic "ask the user"
// fallback, for example); the gosidian_block is identical across profiles.
type Profile string

const (
	ProfileClaude  Profile = "claude"
	ProfileCursor  Profile = "cursor"
	ProfileCodex   Profile = "codex"
	ProfileAider   Profile = "aider"
	ProfileGeneric Profile = "generic"
)

type profileMeta struct {
	DisplayName       string
	PromptAugment     string
	PromptFromScratch string
}

// sharedStubTemplate is the thin instruction-file stub emitted by
// memory_init_agent: Regola Zero (call bootstrap, follow directives_block) +
// local-specifics placeholders. The full operational directives are NOT here —
// they are served by memory_bootstrap. Paths are relative to the embed.FS root
// (rooted at "assets").
const sharedStubTemplate = "assets/_common/stub.tmpl.md"

// sharedDirectivesTemplate is the generic operational directives block served
// by memory_bootstrap (directives_block field), parameterised only by
// {{PROJECT}} and {{DIRECTIVES_VERSION}}.
const sharedDirectivesTemplate = "assets/_common/directives.tmpl.md"

// StubVersion is the version of the instruction-file stub (stub.tmpl.md). It is
// substituted into the `<!-- gosidian:stub v=N -->` marker and surfaced by
// memory_bootstrap as `stub_version`, so an agent knows when its (rarely
// changing) stub must be regenerated via memory_init_agent. Bump when the stub
// contract changes. Guarded by TestStubVersion_PinnedToContent.
//
// v2 (2026-06-09, IMP-048): the in-stub "Specifiche locali" section is now an
// explicit signpost ("write local specifics BELOW the closing marker") instead
// of an editable placeholder — content inside the markers is regenerated on
// every bump, so inviting edits there was a footgun. Already-converted projects
// self-heal on their next bootstrap (stub_version advances → regeneration).
const StubVersion = 2

// DirectivesVersion is the version of the operational directives
// (directives.tmpl.md). It is substituted into the
// `<!-- gosidian:directives v=N -->` marker and surfaced by memory_bootstrap as
// `directives_version`. Since the directives are served fresh each session,
// projects never go stale on content — this version is informational + caching.
// Bump on every meaningful change. Guarded by TestDirectivesVersion_PinnedToContent.
//
// v2 (2026-06-09, IMP-049): directives now instruct pre-stub projects to
// self-convert their instruction file at bootstrap — breaks the first-conversion
// chicken-and-egg so existing projects heal without a manual rollout.
const DirectivesVersion = 2

var profilesMap = map[Profile]profileMeta{
	ProfileClaude: {
		DisplayName:       "Claude Code",
		PromptAugment:     "assets/claude/prompt_augment.md",
		PromptFromScratch: "assets/claude/prompt_fromscratch.md",
	},
	ProfileCursor: {
		DisplayName:       "Cursor",
		PromptAugment:     "assets/cursor/prompt_augment.md",
		PromptFromScratch: "assets/cursor/prompt_fromscratch.md",
	},
	ProfileCodex: {
		DisplayName:       "OpenAI Codex",
		PromptAugment:     "assets/codex/prompt_augment.md",
		PromptFromScratch: "assets/codex/prompt_fromscratch.md",
	},
	ProfileAider: {
		DisplayName:       "Aider",
		PromptAugment:     "assets/aider/prompt_augment.md",
		PromptFromScratch: "assets/aider/prompt_fromscratch.md",
	},
	ProfileGeneric: {
		DisplayName:       "AI coding agent",
		PromptAugment:     "assets/generic/prompt_augment.md",
		PromptFromScratch: "assets/generic/prompt_fromscratch.md",
	},
}

// Profiles returns the list of registered profiles in a stable order.
// Useful for discovery / documentation.
func Profiles() []Profile {
	return []Profile{
		ProfileClaude, ProfileCursor, ProfileCodex, ProfileAider, ProfileGeneric,
	}
}

// IsKnownProfile reports whether p is registered.
func IsKnownProfile(p Profile) bool {
	_, ok := profilesMap[p]
	return ok
}
