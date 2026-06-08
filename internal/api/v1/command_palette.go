package v1

import (
	"net/http"
	"path"
	"strings"
)

// commandPaletteResponse is the dataset Cmd+K consumes. The SPA
// caches it after the first open; a `Cache-Control: no-cache` header
// lets the browser revalidate cheaply on subsequent opens. Mirrors
// the v1.x /api/command-palette shape so a one-line client change
// suffices for the SPA migration.
type commandPaletteResponse struct {
	Notes    []paletteNote    `json:"notes"`
	Projects []paletteProject `json:"projects"`
	Tags     []paletteTag     `json:"tags"`
}

type paletteNote struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type paletteProject struct {
	Name      string `json:"name"`
	NoteCount int    `json:"noteCount"`
}

type paletteTag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func (r *Router) handleCommandPalette(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Index == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "index not configured")
		return
	}

	princ := principalFromContext(req)

	notes := []paletteNote{}
	if rows, err := r.deps.Index.AllNotes(); err == nil {
		notes = make([]paletteNote, 0, len(rows))
		for _, n := range rows {
			if !r.canSee(princ, n.Path) {
				continue
			}
			title := n.Title
			if title == "" {
				b := path.Base(n.Path)
				title = strings.TrimSuffix(b, path.Ext(b))
			}
			notes = append(notes, paletteNote{Path: n.Path, Title: title})
		}
	}

	projects := []paletteProject{}
	if r.deps.Vault != nil {
		if ps, err := r.deps.Vault.Projects(); err == nil {
			projects = make([]paletteProject, 0, len(ps))
			for _, prj := range ps {
				if !princ.CanAccessProject(prj.Name, r.isPublic) {
					continue
				}
				projects = append(projects, paletteProject{Name: prj.Name, NoteCount: prj.NoteCount})
			}
		}
	}

	tags := []paletteTag{}
	if rows, err := r.visibleTags(princ); err == nil {
		tags = make([]paletteTag, 0, len(rows))
		for _, t := range rows {
			tags = append(tags, paletteTag{Tag: t.Tag, Count: t.Count})
		}
	}

	w.Header().Set("Cache-Control", "no-cache")
	WriteJSON(w, http.StatusOK, commandPaletteResponse{
		Notes:    notes,
		Projects: projects,
		Tags:     tags,
	})
}
