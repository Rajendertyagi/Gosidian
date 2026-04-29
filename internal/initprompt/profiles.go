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

// sharedGosidianBlock is the single parametric block embedded into every
// profile output. Paths are relative to the embed.FS root (which is
// rooted at "assets").
const sharedGosidianBlock = "assets/_common/gosidian_block.tmpl.md"

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
