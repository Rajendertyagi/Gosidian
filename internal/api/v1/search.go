package v1

import (
	"net/http"
	"strconv"
	"strings"
)

type searchHit struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// handleSearch runs the FTS index against the query and returns
// `hits[]`. Filters: `?project=` scopes to a top-level dir; `?limit=`
// caps results (default 20, max 200). Snippets are stripped of the
// `<mark>` highlight tags the index returns — the SPA emphasizes
// matches client-side via the same query terms, so the wire payload
// stays plain text.
//
// `?include_outline=true` and `?include_frontmatter=true` are
// accepted but currently ignored — they're documented in the OpenAPI
// for parity with the MCP `memory_search` tool, and the enrichment
// layer arrives in Phase 1.3 along with the rest of admin/discovery
// endpoints.
func (r *Router) handleSearch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Index == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "index not configured")
		return
	}
	q := strings.TrimSpace(req.URL.Query().Get("q"))
	if q == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationRequired, "q is required")
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	project := strings.TrimSpace(req.URL.Query().Get("project"))
	p := principalFromContext(req)

	// Fetch extra hits when filtering (project scope or guest visibility) so
	// the response still has up to `limit` entries after rejection.
	fetchLimit := limit
	if project != "" || !r.seesAllProjects(p) {
		fetchLimit = limit * 4
		if fetchLimit > 800 {
			fetchLimit = 800
		}
	}
	hits, err := r.deps.Index.Search(q, fetchLimit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}

	out := make([]searchHit, 0, len(hits))
	for _, h := range hits {
		if project != "" && !strings.HasPrefix(h.Path, project+"/") && h.Path != project {
			continue
		}
		if !r.canSee(p, h.Path) {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, searchHit{
			Path:    h.Path,
			Title:   h.Title,
			Snippet: stripMarkAPI(h.Snippet),
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"hits": out})
}

// stripMarkAPI mirrors mcp.stripMarkTags but lives in this package to
// avoid the api/v1 → mcp dependency. The two implementations are
// trivial duplications; if a third audience needs the same stripping
// it should be lifted into internal/index alongside the Search call.
func stripMarkAPI(s string) string {
	s = strings.ReplaceAll(s, "<mark>", "")
	s = strings.ReplaceAll(s, "</mark>", "")
	return s
}
