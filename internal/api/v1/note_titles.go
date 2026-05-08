package v1

import (
	"net/http"
	"strconv"
	"strings"
)

// noteTitleHit is the wire shape consumed by the CodeMirror wikilink
// autocomplete extension. Includes both Title and Path because the
// dropdown shows the title and inserts a relative path for nested
// notes.
type noteTitleHit struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// handleNoteTitles powers wikilink autocomplete. The SPA fires this
// endpoint as the user types `[[<prefix>` in the editor, debounced at
// 200ms. We use the existing FTS Search index — title-only matches
// would require a dedicated index, which is overkill for the current
// vault sizes. Limit defaults to 10 (just enough for the dropdown)
// and is capped at 50.
func (r *Router) handleNoteTitles(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Index == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "index not configured")
		return
	}
	q := strings.TrimSpace(req.URL.Query().Get("q"))
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Empty query returns the most recently modified notes — useful as
	// the editor's "show me anything" fallback when the user opens the
	// `[[` autocomplete with no prefix yet, and as the default ranking
	// for the graph view's "focus" picker (last-edited first). Calls
	// the existing index.RecentNotes (used elsewhere by the MCP recent
	// tool) with empty project + zero `since` to mean "all notes,
	// limit-many, mtime desc".
	if q == "" {
		rows, err := r.deps.Index.RecentNotes("", 0, limit)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
			return
		}
		out := make([]noteTitleHit, 0, len(rows))
		for _, n := range rows {
			out = append(out, noteTitleHit{Title: n.Title, Path: n.Path})
		}
		WriteJSON(w, http.StatusOK, map[string]any{"items": out})
		return
	}

	rows, err := r.deps.Index.Search(q, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	out := make([]noteTitleHit, 0, len(rows))
	for _, h := range rows {
		out = append(out, noteTitleHit{Title: h.Title, Path: h.Path})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}
