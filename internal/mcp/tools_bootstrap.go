// Package mcp — memory_bootstrap tool (v1.2, IMP-009).
//
// Single-call aggregate that collapses the Regola-Zero session-start sequence
// (hot, README, tag:status:in-progress, tag:type:skill, recent) into one JSON
// payload. Read-only, scoped by the caller's token project filter.
package mcp

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/gosidian/gosidian/internal/initprompt"
	"github.com/gosidian/gosidian/internal/parser"
	"github.com/gosidian/gosidian/internal/vault"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerBootstrapTool adds the memory_bootstrap tool. Called from
// registerTools() alongside the other v1.2 tools.
func (s *Server) registerBootstrapTool() {
	s.impl.AddTool(mcp.NewTool("memory_bootstrap",
		mcp.WithDescription("Aggregate session-start payload for a project: hot.md + README.md + CLAUDE.md content (when present), active plans (type:plan + status:in-progress), available skills (type:skill), agents (type:agent), 5 most recent notes, and summary stats (note count, top tags). Prefer this over 3-4 separate memory_get + memory_notes_by_tag calls when starting a task. `missing` lists convention files that are absent so the caller knows what scaffold is lacking. When the self-improvement loop is enabled, `pending_insights` lists un-triaged agent-sourced insights (status:pending) awaiting your review. `directives_block` carries the full gosidian operational directives (vault folder map, ingest rules, plan/skill conventions, end-of-task workflow, tag vocabulary) rendered for this project — read and FOLLOW it; it is served fresh every session, so the on-disk instruction file only needs to be a thin stub pointing here. `directives_version` and `stub_version` are the versions of those directives and of the stub template (regenerate your stub via memory_init_agent only when stub_version is ahead of your `<!-- gosidian:stub v=N -->` marker). `agent_md` reports the project's instruction file under any recognised name (AGENTS.md, CLAUDE.md, …) — gosidian does not assume a single filename. In the stub model the instruction file lives in the agent's working dir (outside the vault); when it is not a vault note, `agent_md.expected_external` is true and it is NOT reported in `missing` (which lists only absent vault scaffold such as hot.md / README.md)."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project (top-level folder) to bootstrap. Scoped tokens are forced to their project.")),
		mcp.WithString("profile", mcp.Description("Optional CLI/agent profile for local agent-anchor materialisation (\"claude\", \"generic\", …). Default \"claude\". When the master switch and the project's use_anchors flag are on and the profile supports native subagents, the response includes an `anchors` block: the set of thin agent-anchor files (each pulls its canonical role from the vault at spawn) to write/reconcile in the agent's cwd, plus a `reconcile` directive. Flags off, or a profile without subagent support, yield no anchors.")),
		mcp.WithNumber("known_directives_version", mcp.Description("Token economy: the directives_version you already hold (from a previous bootstrap this session family). When it matches the server's current version, directives_block is omitted from the payload — directives_version is always present so you can detect the match. Omit or pass 0 to always receive the block.")),
		mcp.WithObject("known_etags", mcp.Description("Token economy: map of vault-relative path → etag from a previous bootstrap (e.g. {\"proj/hot.md\": \"...\"}). Files whose etag still matches come back with unchanged:true and no content — re-read only what actually changed. Applies to hot_md, readme and agent_md.")),
		mcp.WithString("mode", mcp.Description("full (default) or lite. lite replaces the hot.md body with its frontmatter + heading outline — a fraction of the tokens; fetch sections you need via memory_get_section. Combine with known_etags for minimal repeat bootstraps.")),
	), s.handleBootstrap)
}

type bootstrapFile struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	ETag    string `json:"etag,omitempty"`
	// ExpectedExternal marks an instruction file that is absent from the vault
	// but expected to live in the agent's working dir (the stub model, ADR-010).
	// Only ever set on the agent_md payload, never on hot.md/README.md.
	ExpectedExternal bool `json:"expected_external,omitempty"`
	// Unchanged marks a file whose etag matched the caller's known_etags —
	// the body was deliberately omitted (Content empty ≠ empty file).
	Unchanged bool `json:"unchanged,omitempty"`
	// Frontmatter+Headings replace Content in lite mode (hot.md only).
	Frontmatter string           `json:"frontmatter,omitempty"`
	Headings    []outlineHeading `json:"headings,omitempty"`
}

type bootstrapStats struct {
	NotesCount int                 `json:"notes_count"`
	TopTags    []bootstrapTagCount `json:"top_tags"`
}

type bootstrapTagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// bootstrapPendingInsights is the owner-facing surface for un-triaged
// self-improvement insights (status:pending). Count is the full total;
// Notes is capped to the first few for a quick preview.
type bootstrapPendingInsights struct {
	Project string    `json:"project"`
	Count   int       `json:"count"`
	Notes   []noteRef `json:"notes"`
}

// conventionFiles maps the relative-to-project filename to the key we expose
// in the JSON payload. Order matters: the `missing` list is emitted in this
// order so repeated calls produce stable output.
var conventionFiles = []struct {
	rel string
	key string
}{
	{"hot.md", "hot_md"},
	{"README.md", "readme"},
}

// agentFileCandidates lists the agent-native instruction file names gosidian
// recognises, in priority order. AGENTS.md is the generic cross-agent default
// (ADR-010); CLAUDE.md / .cursor/rules.mdc / CONVENTIONS.md are detected for
// agent-specific harnesses. gosidian no longer assumes a single name.
var agentFileCandidates = []string{"AGENTS.md", "CLAUDE.md", ".cursor/rules.mdc", "CONVENTIONS.md"}

// anchorReconcileDirective tells the agent how to apply the desired anchor set
// against its cwd. The server is cwd-blind: it only computes the desired files
// + their meta_version; the agent (which has filesystem access) does the diff.
const anchorReconcileDirective = "Riconcilia gli agenti referenziati in `target_dir` (file gitignored). Per ogni item: se il file non esiste → scrivilo con `content`; se esiste con un marker `<!-- gosidian:anchor ... meta=X -->` e X≠`meta_version` → riscrivilo; se X=`meta_version` → lascialo. Rimuovi i file con marker `gosidian:anchor` il cui `canonical` non è più fra gli items (orphan). NON toccare file SENZA il marker `gosidian:anchor` (foreign, scritti a mano): segnalali e proponi l'adozione con `memory_promote_agent`."

// anchorsEnabledFor reports whether the bootstrap should surface agent anchors
// for (project, profile): master switch on, the project opted into use_anchors,
// and the profile supports native subagents. Default-off on every axis keeps
// existing bootstrap behaviour unchanged.
func (s *Server) anchorsEnabledFor(project string, profile initprompt.Profile) bool {
	return s.anchorsEnabled && s.projects != nil && s.projects.UsesAnchors(project) && initprompt.SupportsAnchors(profile)
}

func (s *Server) handleBootstrap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tok, errRes := s.authorizeRead(ctx)
	if errRes != nil {
		return errRes, nil
	}
	project, err := s.resolveProject(tok, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	mode := strings.TrimSpace(req.GetString("mode", "full"))
	switch mode {
	case "full", "lite":
		// ok
	default:
		return mcp.NewToolResultErrorf("unknown mode %q (expected full or lite)", mode), nil
	}
	knownEtags := extractStringMap(req.GetArguments()["known_etags"])

	payload := map[string]any{
		"project": project,
		// Directives are SERVED here (ADR-010): directives_block is the full
		// operational ruleset, rendered generic (parameterised on project).
		// The instruction file on disk is a thin stub that points here, so
		// projects never embed — and never drift from — the directives.
		// stub_version lets an agent know when its (rarely changing) stub
		// must be regenerated via memory_init_agent.
		"directives_version": initprompt.DirectivesVersion,
		"stub_version":       initprompt.StubVersion,
	}
	// Version negotiation: a caller that already holds the current directives
	// skips the whole block — directives_version above is the match signal.
	if req.GetInt("known_directives_version", 0) != initprompt.DirectivesVersion {
		if block, _, derr := initprompt.RenderDirectives(project); derr == nil {
			payload["directives_block"] = block
		}
	}
	var missing []string

	for _, f := range conventionFiles {
		full := path.Join(project, f.rel)
		file := s.loadBootstrapFile(full)
		file = applyKnownEtag(file, knownEtags)
		if mode == "lite" && f.key == "hot_md" && file.Present && !file.Unchanged {
			// Lite: the session cache tends to be the payload's heaviest part.
			// Serve its shape (frontmatter + outline), not its body; the agent
			// pulls the sections it needs via memory_get_section.
			content := []byte(file.Content)
			file.Frontmatter = parser.FrontmatterRawForPath(file.Path, content)
			for _, h := range parser.ExtractHeadings(content) {
				file.Headings = append(file.Headings, outlineHeading{Level: h.Level, Text: h.Text, ID: h.ID})
			}
			file.Content = ""
		}
		payload[f.key] = file
		if !file.Present {
			missing = append(missing, f.rel)
		}
	}

	// agent_md: agent-agnostic detection of the project's instruction file
	// (ADR-010) — no longer assumes CLAUDE.md. Reports the first existing
	// candidate (as a vault note) and its name. In the stub model the
	// instruction file lives in the agent's working dir, OUTSIDE the vault, so
	// its absence from the vault is expected — not a missing scaffold (IMP-050).
	// Flag it expected_external instead of polluting `missing` with a perpetual
	// false positive; `missing` stays reserved for real vault files
	// (hot.md / README.md).
	agentFile, agentName := s.detectAgentFile(project)
	if agentFile.Present {
		payload["agent_md_name"] = agentName
		agentFile = applyKnownEtag(agentFile, knownEtags)
	} else {
		agentFile.ExpectedExternal = true
	}
	payload["agent_md"] = agentFile

	active, err := s.filterByTagAndProject("status:in-progress", project, tok)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("active_plans lookup failed", err), nil
	}
	// Intersect with type:plan — only plans count as "active plans" here.
	active = s.intersectWithTag(active, "type:plan")
	payload["active_plans"] = active

	skills, err := s.filterByTagAndProject("type:skill", project, tok)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("skills lookup failed", err), nil
	}
	payload["available_skills"] = s.mergeGlobals(skills, "type:skill", project, tok)

	agents, err := s.filterByTagAndProject("type:agent", project, tok)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("agents lookup failed", err), nil
	}
	payload["available_agents"] = s.mergeGlobals(agents, "type:agent", project, tok)

	// anchors: local agent-anchor materialisation set (plan 20260630). Gated by
	// the master switch + the project's use_anchors flag + profile capability.
	// Cwd-blind: the server returns the desired set + a reconcile directive; the
	// agent diffs against its cwd using each file's marker. Default-off → the key
	// is absent and bootstrap behaves exactly as before.
	if profile := initprompt.Profile(strings.TrimSpace(req.GetString("profile", "claude"))); s.anchorsEnabledFor(project, profile) {
		items, aerr := s.buildAgentAnchors(project, profile, tok)
		if aerr != nil {
			return mcp.NewToolResultErrorFromErr("anchors lookup failed", aerr), nil
		}
		payload["anchors"] = map[string]any{
			"profile":    string(profile),
			"target_dir": initprompt.AnchorDir(profile),
			"items":      items,
			"reconcile":  anchorReconcileDirective,
		}
	}

	recent, err := s.index.RecentNotes(project, 0, 5)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("recent lookup failed", err), nil
	}
	recentOut := make([]recentNoteResponse, 0, len(recent))
	for _, n := range recent {
		if !tok.AllowsPath(n.Path) {
			continue
		}
		recentOut = append(recentOut, recentNoteResponse{Path: n.Path, Title: n.Title, Mtime: n.Mtime})
	}
	payload["recent_notes"] = recentOut

	projNotes, err := s.index.NotesByPrefix(project)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("notes count failed", err), nil
	}
	tagCounts, err := s.index.TagsByProject(project)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("tag counts failed", err), nil
	}
	top := make([]bootstrapTagCount, 0, 5)
	for i, t := range tagCounts {
		if i >= 5 {
			break
		}
		top = append(top, bootstrapTagCount{Tag: t.Tag, Count: t.Count})
	}
	payload["stats"] = bootstrapStats{
		NotesCount: len(projNotes),
		TopTags:    top,
	}

	// pending_insights surfaces the owner's un-triaged self-improvement
	// insights (status:pending) regardless of which project is being
	// bootstrapped, so they're seen at every session start. Only present
	// when the loop is enabled, and only populated for tokens that can read
	// the insights project. Best-effort: a lookup error degrades to absent
	// rather than failing the whole bootstrap.
	if s.selfImproveEnabled {
		insProject := s.selfImproveProject
		if insProject == "" {
			insProject = "insights"
		}
		if pending, perr := s.filterByTagAndProject("status:pending", insProject, tok); perr == nil {
			pending = s.intersectWithTag(pending, "type:insight")
			notes := pending
			if len(notes) > 10 {
				notes = notes[:10]
			}
			payload["pending_insights"] = bootstrapPendingInsights{
				Project: insProject,
				Count:   len(pending),
				Notes:   notes,
			}
		}
	}

	if missing == nil {
		missing = []string{}
	}
	payload["missing"] = missing

	return mcp.NewToolResultJSON(payload)
}

// loadBootstrapFile reads one convention file into a bootstrapFile, including
// its etag. A missing file is not an error — it returns {Present: false}.
// Any other error (permission denied, index mismatch) also surfaces as absent
// so the tool never fails the whole call on a single missing file.
func (s *Server) loadBootstrapFile(rel string) bootstrapFile {
	note, err := s.vault.Load(rel)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || isNoteNotFound(err) {
			return bootstrapFile{Present: false}
		}
		return bootstrapFile{Present: false}
	}
	return bootstrapFile{
		Present: true,
		Path:    rel,
		Content: string(note.Content),
		ETag:    note.ETag(),
	}
}

// applyKnownEtag omits the body of a file whose etag matches the caller's
// known_etags entry, flagging it Unchanged so the client can tell "omitted
// because you have it" from "empty file". Keys are vault-relative paths as
// returned by a previous bootstrap.
func applyKnownEtag(file bootstrapFile, known map[string]string) bootstrapFile {
	if !file.Present || file.Path == "" || len(known) == 0 {
		return file
	}
	if et, ok := known[file.Path]; ok && et != "" && et == file.ETag {
		file.Content = ""
		file.Unchanged = true
	}
	return file
}

// detectAgentFile returns the first existing instruction file (as a vault note)
// among agentFileCandidates under the project, plus the name that matched.
// Returns {Present:false}, "" when none exists. Agent-agnostic (ADR-010).
func (s *Server) detectAgentFile(project string) (bootstrapFile, string) {
	for _, name := range agentFileCandidates {
		f := s.loadBootstrapFile(path.Join(project, name))
		if f.Present {
			return f, name
		}
	}
	return bootstrapFile{Present: false}, ""
}

// filterByTagAndProject returns note refs tagged with `tag`, restricted to
// paths under `<project>/` and allowed by the caller's token scope.
func (s *Server) filterByTagAndProject(tag, project string, tok tokenScoped) ([]noteRef, error) {
	notes, err := s.index.NotesByTag(tag)
	if err != nil {
		return nil, err
	}
	out := make([]noteRef, 0)
	prefix := project + "/"
	for _, n := range notes {
		if !strings.HasPrefix(n.Path, prefix) {
			continue
		}
		if !tok.AllowsPath(n.Path) {
			continue
		}
		out = append(out, noteRef{Path: n.Path, Title: n.Title})
	}
	return out, nil
}

// intersectWithTag returns the subset of `candidates` whose path also carries
// the given tag. Used to intersect `status:in-progress` with `type:plan`
// without a second full scan: the candidate list is already small.
func (s *Server) intersectWithTag(candidates []noteRef, tag string) []noteRef {
	tagged, err := s.index.NotesByTag(tag)
	if err != nil {
		return candidates
	}
	set := make(map[string]struct{}, len(tagged))
	for _, n := range tagged {
		set[n.Path] = struct{}{}
	}
	out := make([]noteRef, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := set[c.Path]; ok {
			out = append(out, c)
		}
	}
	return out
}

// mergeGlobals augments a project's local skills/agents with those from the
// shared global projects, when the global feature is on and the project opted
// in (projects.Flags.UseGlobals). global-public entries are shared with every
// token; global-private entries only with tokens that can read that project
// (admin / scoped to it). Deduplication is local-overrides-global by title: a
// local entry shadows a global one with the same title. Each entry carries its
// source (local | global | global-private).
func (s *Server) mergeGlobals(local []noteRef, tag, project string, tok tokenScoped) []noteRef {
	if !s.globalEnabled || s.projects == nil || !s.projects.UsesGlobals(project) {
		return local
	}
	seen := make(map[string]struct{}, len(local))
	out := make([]noteRef, 0, len(local))
	for _, r := range local {
		r.Source = "local"
		seen[r.Title] = struct{}{}
		out = append(out, r)
	}
	add := func(globalProject, source string, applyScope bool) {
		if globalProject == "" || globalProject == project {
			return
		}
		if applyScope && !tok.AllowsPath(globalProject+"/") {
			return
		}
		rows, err := s.index.NotesByTagInProject(tag, globalProject)
		if err != nil {
			return
		}
		tmplPrefix := globalProject + "/templates/"
		for _, n := range rows {
			// Template definition files live under <global>/templates/ and
			// are scaffold sources (often with {{PLACEHOLDER}} content), not
			// usable skills/agents — never surface them in the merge.
			if strings.HasPrefix(n.Path, tmplPrefix) {
				continue
			}
			if _, dup := seen[n.Title]; dup {
				continue
			}
			seen[n.Title] = struct{}{}
			out = append(out, noteRef{Path: n.Path, Title: n.Title, Source: source})
		}
	}
	add(s.globalPublic, "global", false)         // shared surface: bypass token scope
	add(s.globalPrivate, "global-private", true) // gated: admin / owner-scoped only
	return out
}

// tokenScoped is a narrow subset of *auth.Token, declared locally so helpers
// don't depend on the concrete type and tests can stub if needed.
type tokenScoped interface {
	AllowsPath(path string) bool
}

// isNoteNotFound checks for the "file does not exist" condition coming from
// vault.Load when the underlying os.Stat returns ENOENT.
func isNoteNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist)
}

// compile-time check: vault.Note is what Load returns.
var _ = func() *vault.Note { return nil }
