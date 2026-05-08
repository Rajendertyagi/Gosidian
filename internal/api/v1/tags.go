package v1

import (
	"net/http"
	"strings"
)

type tagView struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// handleTags returns the tag set with counts. Filter by `?project=`
// (top-level dir) to scope results to a single project's notes.
func (r *Router) handleTags(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Index == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "index not configured")
		return
	}
	project := strings.TrimSpace(req.URL.Query().Get("project"))
	var rows []indexTagCount
	var err error
	if project != "" {
		raw, e := r.deps.Index.TagsByProject(project)
		err = e
		for _, t := range raw {
			rows = append(rows, indexTagCount{Tag: t.Tag, Count: t.Count})
		}
	} else {
		raw, e := r.deps.Index.Tags()
		err = e
		for _, t := range raw {
			rows = append(rows, indexTagCount{Tag: t.Tag, Count: t.Count})
		}
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	out := make([]tagView, 0, len(rows))
	for _, t := range rows {
		out = append(out, tagView{Tag: t.Tag, Count: t.Count})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

// handleTagByName lists notes carrying a specific tag. The tag name
// is everything after `/api/v1/tags/`. Trailing slashes are tolerated.
// Filter by `?project=` to intersect with a project; useful for
// "show me all status:in-progress notes in this project" queries.
func (r *Router) handleTagByName(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if r.deps.Index == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "index not configured")
		return
	}
	tag := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/v1/tags/"), "/")
	if tag == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationRequired, "tag name required")
		return
	}
	project := strings.TrimSpace(req.URL.Query().Get("project"))

	rows, err := r.deps.Index.NotesByTag(tag)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	out := make([]noteSummary, 0, len(rows))
	for _, n := range rows {
		if project != "" && !strings.HasPrefix(n.Path, project+"/") && n.Path != project {
			continue
		}
		out = append(out, noteSummary{Path: n.Path, Title: n.Title})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

// indexTagCount mirrors index.TagCount field-for-field so the handler
// can iterate without re-importing the type into the response shape.
type indexTagCount struct {
	Tag   string
	Count int
}
