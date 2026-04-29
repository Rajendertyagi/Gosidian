package initprompt

import (
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"
)

//go:embed all:assets/*
var assetsFS embed.FS

// Mode selects between the "augment an existing init file" flow and the
// "create from scratch" flow. ModeAugment is preferred when the caller
// already has the output of the agent's native /init.
type Mode string

const (
	ModeAugment     Mode = "augment"
	ModeFromScratch Mode = "from-scratch"
)

// Hints bundles optional per-call context. Fields that are non-empty are
// substituted into gosidian_block placeholders server-side, so the agent
// does not have to resolve them afterwards. Fields left empty keep their
// {{PLACEHOLDER}} form and the agent fills them after its Q&A.
type Hints struct {
	Language     string // used for {{LANGUAGE}} (vault notes language)
	CodeLanguage string // used for {{CODE_LANGUAGE}} (code / commit language)
	ProjectType  string // used for {{PROJECT_TYPE}} (app / CLI / library / infra / docs)
	Stack        string // used for {{STACK}} (framework / runtime summary)
	HotFiles     string // used for {{HOT_FILES}}
	FilenameHint string // used for {{FILENAME_HINT}} (e.g. "CLAUDE.md")
	CwdHint      string // used for {{CWD_HINT}} (agent's cwd, informative only)
	AgentName    string // used for {{AGENT_NAME}}; falls back to profile DisplayName
}

// Result is the value returned by Render and exposed as JSON by the MCP
// handler. `Prompt` is the multi-section instruction set the agent
// executes; `GosidianBlock` is the parametric markdown to innest into the
// agent-native instruction file.
type Result struct {
	Mode               Mode     `json:"mode"`
	NeedsScaffold      bool     `json:"needs_scaffold"`
	Prompt             string   `json:"prompt"`
	GosidianBlock      string   `json:"gosidian_block"`
	SuggestedQuestions []string `json:"suggested_questions"`
}

// ErrUnknownProfile is returned when a caller requests a profile that is
// not registered in profilesMap.
var ErrUnknownProfile = errors.New("unknown agent profile")

// ErrUnknownMode is returned for an invalid mode value.
var ErrUnknownMode = errors.New("unknown mode")

// Render produces the init payload for (project, profile, mode).
// projectExists controls the NeedsScaffold flag — true when the project
// already exists in the vault, false when the agent should call
// memory_project_scaffold first.
func Render(project string, profile Profile, mode Mode, hints Hints, projectExists bool) (Result, error) {
	if strings.TrimSpace(project) == "" {
		return Result{}, errors.New("project is required")
	}
	if mode != ModeAugment && mode != ModeFromScratch {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownMode, mode)
	}
	prof, ok := profilesMap[profile]
	if !ok {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownProfile, profile)
	}

	promptAsset := prof.PromptAugment
	if mode == ModeFromScratch {
		promptAsset = prof.PromptFromScratch
	}
	promptBytes, err := assetsFS.ReadFile(promptAsset)
	if err != nil {
		return Result{}, fmt.Errorf("read prompt asset %s: %w", promptAsset, err)
	}
	blockBytes, err := assetsFS.ReadFile(sharedGosidianBlock)
	if err != nil {
		return Result{}, fmt.Errorf("read gosidian block: %w", err)
	}

	vars := map[string]string{
		"PROJECT":       project,
		"TODAY":         time.Now().UTC().Format("2006-01-02"),
		"AGENT_PROFILE": string(profile),
		"AGENT_NAME":    coalesce(hints.AgentName, prof.DisplayName),
	}
	if hints.Language != "" {
		vars["LANGUAGE"] = hints.Language
	}
	if hints.CodeLanguage != "" {
		vars["CODE_LANGUAGE"] = hints.CodeLanguage
	}
	if hints.ProjectType != "" {
		vars["PROJECT_TYPE"] = hints.ProjectType
	}
	if hints.Stack != "" {
		vars["STACK"] = hints.Stack
	}
	if hints.HotFiles != "" {
		vars["HOT_FILES"] = hints.HotFiles
	}
	if hints.FilenameHint != "" {
		vars["FILENAME_HINT"] = hints.FilenameHint
	}
	if hints.CwdHint != "" {
		vars["CWD_HINT"] = hints.CwdHint
	}

	return Result{
		Mode:               mode,
		NeedsScaffold:      !projectExists,
		Prompt:             applyVars(string(promptBytes), vars),
		GosidianBlock:      applyVars(string(blockBytes), vars),
		SuggestedQuestions: defaultSuggestedQuestions(),
	}, nil
}

func applyVars(body string, vars map[string]string) string {
	for k, v := range vars {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
	}
	return body
}

func coalesce(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func defaultSuggestedQuestions() []string {
	return []string{
		"In quale lingua preferisci le note del vault (italiano / inglese / …)?",
		"Che tipo di progetto è (applicazione web / CLI / libreria / infra / docs-only)?",
		"Quali sono i 2-3 file o cartelle più hot (che cambiano spesso)?",
		"Ci sono convenzioni di build / test / code style già stabilite da rispettare?",
	}
}
