// Package mcp — memory_promote_agent tool (plan 20260630-agent-anchors, M4).
//
// Adopts a foreign, hand-written local agent file (a CLI subagent definition
// NOT produced by gosidian — no `gosidian:anchor` marker) into a canonical
// vault `type:agent` note, so it becomes a shared, version-controlled role. The
// agent then replaces the local file with the returned anchor (which pulls the
// role from the vault at spawn). Human-gated: the agent invokes this only on
// the user's explicit confirmation, surfaced by the bootstrap reconcile flow.
package mcp

import (
	"context"
	"strings"

	"github.com/gosidian/gosidian/internal/audit"
	"github.com/gosidian/gosidian/internal/initprompt"
	"github.com/gosidian/gosidian/internal/parser"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPromoteAgentTool() {
	s.impl.AddTool(mcp.NewTool("memory_promote_agent",
		mcp.WithDescription("Adopt a foreign, hand-written local agent file (a CLI subagent definition not produced by gosidian — no `gosidian:anchor` marker) into a canonical vault `type:agent` note, then return the anchor to replace it with. Use this when the bootstrap `anchors.reconcile` flow reports a foreign file and the USER confirms adoption: the file's content becomes a shared, version-controlled role at `<project>/agents/<slug>.md` (its system-prompt body preserved; name/tools captured into a `harness:` block), and the returned `anchor` is the thin local file (pulling the role from the vault) to write in place of the original. One source of truth, no content lost. Writes to the vault (requires write access)."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project (top-level folder) that will own the promoted agent. Scoped tokens are forced to their project.")),
		mcp.WithString("slug", mcp.Required(), mcp.Description("Agent slug — the note basename without extension, e.g. \"rc-database\". The canonical note is created at <project>/agents/<slug>.md.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Full content of the foreign local agent file (frontmatter name/description/tools + system-prompt body). Parsed to build the canonical note; the body is preserved verbatim.")),
		mcp.WithString("profile", mcp.Description("CLI profile for the returned anchor. Default \"claude\".")),
	), s.handlePromoteAgent)
}

type promoteAgentResponse struct {
	Path   string     `json:"path"`             // canonical vault note created
	Anchor *anchorRef `json:"anchor,omitempty"` // anchor to write in place of the foreign file
}

func (s *Server) handlePromoteAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	project = strings.TrimSpace(project)
	slug := strings.TrimSpace(req.GetString("slug", ""))
	if slug == "" || strings.ContainsAny(slug, "/\\") {
		return mcp.NewToolResultError("slug must be a non-empty basename without slashes"), nil
	}
	foreign, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if res := s.rejectIfHidden(project); res != nil {
		return res, nil
	}

	rel, err := s.vault.Rel(project + "/agents/" + slug + ".md")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid path", err), nil
	}
	tok, errRes := s.authorizeWrite(ctx, rel)
	if errRes != nil {
		return errRes, nil
	}
	if !tok.AllowsProject(project) {
		return mcp.NewToolResultErrorf("project %q is outside the token's scope %q", project, tok.ScopeLabel()), nil
	}
	if _, err := s.vault.Load(rel); err == nil {
		return mcp.NewToolResultErrorf("canonical agent %q already exists; nothing to promote", rel), nil
	}

	canonical := buildCanonicalAgentNote(project, slug, []byte(foreign))
	if errRes := s.checkWriteLimits(tok, len(canonical)); errRes != nil {
		return errRes, nil
	}
	if err := s.writeAndIndex(rel, []byte(canonical)); err != nil {
		return mcp.NewToolResultErrorFromErr("write failed", err), nil
	}
	s.auditWrite(ctx, audit.ActionCreate, rel, "", int64(len(canonical)))
	if fresh, lerr := s.vault.Load(rel); lerr == nil {
		s.publishNoteChange("create", rel, fresh.ETag(), true)
	}

	resp := promoteAgentResponse{Path: rel}
	profile := initprompt.Profile(strings.TrimSpace(req.GetString("profile", "claude")))
	if initprompt.SupportsAnchors(profile) {
		if ar, aerr := initprompt.RenderAgentAnchor(profile, anchorInputFromNote(rel, []byte(canonical))); aerr == nil {
			resp.Anchor = &anchorRef{Path: ar.Path, Content: ar.Content, MetaVersion: ar.MetaVersion, Canonical: rel}
		}
	}
	return mcp.NewToolResultJSON(resp)
}

// buildCanonicalAgentNote turns a foreign local agent file (CLI subagent
// frontmatter name/description/tools + body) into a canonical vault type:agent
// note: gosidian frontmatter (title/description/tags/type) + an optional
// harness: block preserving the foreign name/tools, + the original body. The
// role text is not lost; it becomes the shared source of truth.
func buildCanonicalAgentNote(project, slug string, foreign []byte) string {
	raw := parser.FrontmatterRawForPath(slug+".md", foreign)
	fields := parser.ParseFrontmatterFields(raw)
	name := slug
	if v, ok := fields["name"].(string); ok && strings.TrimSpace(v) != "" {
		name = strings.TrimSpace(v)
	}
	title := name
	if v, ok := fields["title"].(string); ok && strings.TrimSpace(v) != "" {
		title = strings.TrimSpace(v)
	}
	desc := ""
	if v, ok := fields["description"].(string); ok {
		desc = strings.TrimSpace(v)
	}
	var tools []string
	if v, ok := fields["tools"].(string); ok && strings.TrimSpace(v) != "" {
		for _, t := range strings.Split(v, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tools = append(tools, t)
			}
		}
	}
	body := strings.TrimLeft(parser.BodyAfterFrontmatter(foreign), "\n")

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + title + "\n")
	if desc != "" {
		b.WriteString("description: " + desc + "\n")
	}
	b.WriteString("tags: [" + project + ", type:agent]\n")
	b.WriteString("type: agent\n")
	if name != slug || len(tools) > 0 {
		b.WriteString("harness:\n")
		if name != slug {
			b.WriteString("  name: " + name + "\n")
		}
		if len(tools) > 0 {
			b.WriteString("  tools: [" + strings.Join(tools, ", ") + "]\n")
		}
	}
	b.WriteString("---\n\n")
	if body != "" {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
	} else {
		b.WriteString("# " + title + "\n")
	}
	return b.String()
}
