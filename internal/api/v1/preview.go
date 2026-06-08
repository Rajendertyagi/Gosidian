package v1

import (
	"net/http"

	"github.com/gosidian/gosidian/internal/authz"
	"github.com/gosidian/gosidian/internal/parser"
)

// previewRequest carries raw markdown from the editor split-pane.
type previewRequest struct {
	Markdown string `json:"markdown"`
}

// previewResponse returns sanitized HTML the SPA can drop into a
// preview pane via DOMPurify.sanitize. Server-side goldmark is
// already configured safe-by-default, but the SPA passes it through
// DOMPurify for defense in depth.
type previewResponse struct {
	HTML string `json:"html"`
}

// handlePreview renders markdown to HTML through the same parser
// stack used by the v1.x HTML handlers and the MCP `memory_get`
// rendered-body output. Wikilinks are resolved against the live
// index so the preview shows resolved links and dangling-link
// indicators identically to the read view.
func (r *Router) handlePreview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Renderer == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "renderer not configured")
		return
	}
	var body previewRequest
	if err := DecodeJSON(req, &body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, err.Error())
		return
	}

	html, err := r.deps.Renderer.Render([]byte(body.Markdown), r.wikilinkResolver(principalFromContext(req)))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "render: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, previewResponse{HTML: html})
}

// wikilinkResolver returns a parser.Resolver that consults the index
// to translate `[[Note Title]]` into a vault path. The resolver
// returns "" for unknown targets so the renderer marks them as
// dangling links — matching the existing HTML view convention.
func (r *Router) wikilinkResolver(p authz.Principal) parser.Resolver {
	return parser.ResolverFunc(func(target string) string {
		if r.deps.Index == nil {
			return ""
		}
		// Try by exact path first (allows wikilinks like [[folder/Note]]).
		if rows, err := r.deps.Index.NotesByPrefix(target); err == nil {
			for _, n := range rows {
				if (n.Path == target || n.Path == target+".md" || n.Title == target) && r.canSee(p, n.Path) {
					return n.Path
				}
			}
		}
		// Fall back to a title scan via search — bounded result set. Resolve
		// only to notes the principal may see, so a guest's preview of a
		// public note renders private targets as dangling, not as live links.
		if rows, err := r.deps.Index.Search(target, 5); err == nil {
			for _, h := range rows {
				if h.Title == target && r.canSee(p, h.Path) {
					return h.Path
				}
			}
		}
		return ""
	})
}
