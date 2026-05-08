// Package mcp — memory_init_agent tool (v1.11).
//
// Returns an init-prompt payload for adopting gosidian as the memory
// layer in a new project. The caller (an AI agent) uses the prompt to
// create or augment its agent-native instruction file (CLAUDE.md /
// AGENTS.md / .cursor/rules.mdc / CONVENTIONS.md / …) with the
// gosidian_block (Regola Zero + ingest rules + workflow end-of-task).
//
// Design notes:
//   - Read-only. The tool does NOT write to the vault and does NOT read
//     the agent's cwd. All scanning and file materialisation are
//     delegated to the agent itself.
//   - The filename is NOT chosen by the server. The agent determines it
//     (the tool surfaces an optional filename_hint but doesn't validate).
//   - Two modes: augment (existing_content provided → merge preserving
//     existing sections) and from-scratch (no existing_content → create
//     a new file with cwd-scan guidance).
//   - needs_scaffold flags whether the agent must call
//     memory_project_scaffold before materialising the instruction file.
package mcp

import (
	"context"
	"strings"

	"github.com/gosidian/gosidian/internal/initprompt"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerInitAgentTool() {
	s.impl.AddTool(mcp.NewTool("memory_init_agent",
		mcp.WithDescription("Produce an init-prompt payload for adopting gosidian as the memory layer in a new project. Returns a multi-section `prompt` that instructs the caller to create or update the agent-native instruction file (CLAUDE.md / AGENTS.md / .cursor/rules.mdc / CONVENTIONS.md / …), plus a parametric `gosidian_block` to innest into it. Two modes, selected automatically by the presence of `existing_content`: **augment** (preferred — the caller already ran the agent's native /init and passes its output; the prompt instructs a merge preserving existing sections) and **from-scratch** (fallback — no existing content; the prompt includes cwd-scan instructions). The server does NOT read the caller's filesystem: all scanning and writing happen on the agent side. The tool does NOT choose the filename — the agent determines it (optional `filename_hint` is surfaced but not validated). If the project doesn't exist in the vault yet, `needs_scaffold=true` in the response tells the agent to call `memory_project_scaffold` first."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project (top-level folder) to initialise. May not exist yet; check `needs_scaffold` in the response to know whether to call `memory_project_scaffold` first. Scoped tokens are forced to their project.")),
		mcp.WithString("agent_profile", mcp.Description("Target agent identifier. Known values: \"claude\", \"cursor\", \"codex\", \"aider\", \"generic\". Default \"generic\". Influences only the prompt tone and tool references — the gosidian_block is identical across profiles.")),
		mcp.WithString("existing_content", mcp.Description("Content of the agent's native instruction file when it already exists (the output of /init). If non-empty the tool switches to augment mode and the prompt will instruct a merge that preserves every existing section.")),
		mcp.WithString("filename_hint", mcp.Description("Optional filename the agent plans to use, e.g. \"CLAUDE.md\", \"AGENTS.md\", \".cursor/rules.mdc\". Surfaced in the prompt but never validated server-side.")),
		mcp.WithString("cwd_hint", mcp.Description("Absolute path of the agent's cwd. Used informatively in the prompt; the server does not read it.")),
		mcp.WithObject("user_hints", mcp.Description("Optional map with keys {language, code_language, project_type, stack, hot_files, agent_name}. Non-empty values are substituted into the gosidian_block placeholders server-side so the agent doesn't need to ask the user for them later.")),
	), s.handleInitAgent)
}

type initAgentResponse struct {
	Mode               string   `json:"mode"`
	NeedsScaffold      bool     `json:"needs_scaffold"`
	Prompt             string   `json:"prompt"`
	GosidianBlock      string   `json:"gosidian_block"`
	SuggestedQuestions []string `json:"suggested_questions"`
}

func (s *Server) handleInitAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tok, errRes := s.authorizeRead(ctx)
	if errRes != nil {
		return errRes, nil
	}
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return mcp.NewToolResultError("project must not be empty"), nil
	}
	if scope := tok.ProjectFilter(); scope != "" && project != scope {
		return mcp.NewToolResultErrorf("project %q is outside the token's scope %q", project, scope), nil
	}
	if res := s.rejectIfHidden(project); res != nil {
		return res, nil
	}

	profileStr := strings.TrimSpace(req.GetString("agent_profile", "generic"))
	if profileStr == "" {
		profileStr = "generic"
	}
	profile := initprompt.Profile(profileStr)
	if !initprompt.IsKnownProfile(profile) {
		return mcp.NewToolResultErrorf("unknown agent_profile %q; known values: claude, cursor, codex, aider, generic", profileStr), nil
	}

	existing := req.GetString("existing_content", "")
	mode := initprompt.ModeFromScratch
	if strings.TrimSpace(existing) != "" {
		mode = initprompt.ModeAugment
	}

	hintsMap := extractStringMap(req.GetArguments()["user_hints"])
	hints := initprompt.Hints{
		Language:     hintsMap["language"],
		CodeLanguage: hintsMap["code_language"],
		ProjectType:  hintsMap["project_type"],
		Stack:        hintsMap["stack"],
		HotFiles:     hintsMap["hot_files"],
		AgentName:    hintsMap["agent_name"],
		FilenameHint: req.GetString("filename_hint", ""),
		CwdHint:      req.GetString("cwd_hint", ""),
	}

	projects, err := s.vault.Projects()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("projects lookup failed", err), nil
	}
	exists := false
	for _, p := range projects {
		if p.Name == project {
			exists = true
			break
		}
	}

	res, err := initprompt.Render(project, profile, mode, hints, exists)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("render failed", err), nil
	}

	return mcp.NewToolResultJSON(initAgentResponse{
		Mode:               string(res.Mode),
		NeedsScaffold:      res.NeedsScaffold,
		Prompt:             res.Prompt,
		GosidianBlock:      res.GosidianBlock,
		SuggestedQuestions: res.SuggestedQuestions,
	})
}
