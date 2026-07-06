// Package mcp — per-token tool profiles (plan 20260706-tool-profile-per-token).
//
// A token created with --tool-profile core sees only the worker subset of the
// tool catalogue: mcp-go's WithToolFilter applies the cut to tools/list AND
// enforces it on tools/call (a filtered-out tool is "not found" even when
// called by name), so the profile is an access-control boundary, not a
// visibility hint. Empty/"full" profiles — including admin tokens and
// auth-disabled deployments — see everything, keeping existing setups intact.
package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// coreToolSet is the worker subset: session start, note CRUD, targeted reads,
// search/discovery basics, uploads and the handoff lifecycle. Everything else
// (admin, audit, lint, compact, scaffold, init/promote, advanced discovery)
// stays full-profile — Prometheus shows ~30 tools with 0-3 calls/30d, all
// outside this set.
var coreToolSet = map[string]struct{}{
	"memory_bootstrap":         {},
	"memory_get":               {},
	"memory_get_section":       {},
	"memory_get_outline":       {},
	"memory_get_frontmatter":   {},
	"memory_batch_get":         {},
	"memory_create":            {},
	"memory_update":            {},
	"memory_edit":              {},
	"memory_append":            {},
	"memory_search":            {},
	"memory_list_notes":        {},
	"memory_list_projects":     {},
	"memory_notes_by_tag":      {},
	"memory_upload_attachment": {},
	"memory_upload_resource":   {},
	"memory_create_handoff":    {},
	"memory_pending_handoffs":  {},
	"memory_claim_handoff":     {},
	"memory_complete_handoff":  {},
	"memory_wait_changes":      {},
}

// filterToolsByProfile is the ToolFilterFunc wired into the MCPServer. The
// token travels in ctx for every client→server message (WithSSEContextFunc),
// so the same decision applies to tools/list and tools/call. Beyond the
// static core set, a few tools are admitted dynamically: the media/table
// creators only when their vault flag is on (no point paying their schema on
// an instance that would refuse the call), and memory_self_improve only for
// tokens opted into that loop.
func (s *Server) filterToolsByProfile(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	tok := s.tokenFromContext(ctx)
	if tok == nil || !tok.IsCoreProfile() {
		return tools
	}
	out := make([]mcp.Tool, 0, len(coreToolSet)+3)
	for _, t := range tools {
		if _, ok := coreToolSet[t.Name]; ok {
			out = append(out, t)
			continue
		}
		switch t.Name {
		case "memory_create_media_note":
			if s.vault.MediaNotesEnabled() {
				out = append(out, t)
			}
		case "memory_create_table_note":
			if s.vault.TableNotesEnabled() {
				out = append(out, t)
			}
		case "memory_self_improve":
			if tok.SelfImproveOptIn {
				out = append(out, t)
			}
		}
	}
	return out
}
