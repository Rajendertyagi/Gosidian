package v1

import (
	"net/http"

	"github.com/gosidian/gosidian/internal/authz"
	"github.com/gosidian/gosidian/internal/vault"
)

// pathWithNoteExt reports whether path equals target plus one of the
// recognised note extensions (extension-less wikilink form).
func pathWithNoteExt(path, target string) bool {
	for _, e := range vault.NoteExtensions() {
		if path == target+e {
			return true
		}
	}
	return false
}

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

	html, err := r.deps.Renderer.Render([]byte(body.Markdown), previewResolver{r: r, p: principalFromContext(req)})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "render: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, previewResponse{HTML: html})
}

// previewResolver resolves both `[[wiki-links]]` (Resolve) and `![[image]]`
// embeds (ResolveImage) for the preview renderer, consulting the live index and
// vault. It implements parser.Resolver and parser.ImageResolver.
type previewResolver struct {
	r *Router
	p authz.Principal
}

// Resolve translates `[[Note Title]]` into a vault path the principal may see,
// or "" for unknown targets (rendered as a dangling link).
func (pr previewResolver) Resolve(target string) string {
	r := pr.r
	if r.deps.Index == nil {
		return ""
	}
	// Try by exact path first (allows wikilinks like [[folder/Note]]). The
	// extension-less form is tried against every note extension, not just .md,
	// so links to .html notes resolve like they do in the index (BUG-024).
	if rows, err := r.deps.Index.NotesByPrefix(target); err == nil {
		for _, n := range rows {
			if (n.Path == target || pathWithNoteExt(n.Path, target) || n.Title == target) && r.canSee(pr.p, n.Path) {
				return n.Path
			}
		}
	}
	// Fall back to a title scan via search — bounded result set. Resolve only to
	// notes the principal may see, so a guest's preview of a public note renders
	// private targets as dangling, not as live links.
	if rows, err := r.deps.Index.Search(target, 5); err == nil {
		for _, h := range rows {
			if h.Title == target && r.canSee(pr.p, h.Path) {
				return h.Path
			}
		}
	}
	return ""
}

// ResolveImage maps an `![[target]]` embed to an image URL: first an attachment
// by name/path, then a media note (type:image) referenced by name → its image.
// The actual byte access stays gated by the /vault-files handler.
func (pr previewResolver) ResolveImage(target string) string {
	r := pr.r
	if r.deps.Vault == nil {
		return ""
	}
	if rel, ok := r.deps.Vault.ResolveAttachmentByName(target); ok {
		return "/vault-files/" + rel
	}
	if path := pr.Resolve(target); path != "" {
		if note, err := r.deps.Vault.Load(path); err == nil {
			if ref, kind := r.deps.Vault.MediaRefForNote(note.Path, note.Content); kind == "image" && !ref.Broken {
				return ref.URL
			}
		}
	}
	return ""
}
