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

	"github.com/gosidian/gosidian/internal/vault"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerBootstrapTool adds the memory_bootstrap tool. Called from
// registerTools() alongside the other v1.2 tools.
func (s *Server) registerBootstrapTool() {
	s.impl.AddTool(mcp.NewTool("memory_bootstrap",
		mcp.WithDescription("Aggregate session-start payload for a project: hot.md + README.md + CLAUDE.md content (when present), active plans (type:plan + status:in-progress), available skills (type:skill), agents (type:agent), 5 most recent notes, and summary stats (note count, top tags). Prefer this over 3-4 separate memory_get + memory_notes_by_tag calls when starting a task. `missing` lists convention files that are absent so the caller knows what scaffold is lacking. When the self-improvement loop is enabled, `pending_insights` lists un-triaged agent-sourced insights (status:pending) awaiting your review."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project (top-level folder) to bootstrap. Scoped tokens are forced to their project.")),
	), s.handleBootstrap)
}

type bootstrapFile struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	ETag    string `json:"etag,omitempty"`
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
	{"CLAUDE.md", "claude_md"},
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

	payload := map[string]any{
		"project": project,
	}
	var missing []string

	for _, f := range conventionFiles {
		full := path.Join(project, f.rel)
		file := s.loadBootstrapFile(full)
		payload[f.key] = file
		if !file.Present {
			missing = append(missing, f.rel)
		}
	}

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
	ProjectFilter() string
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
