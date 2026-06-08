package v1

import "net/http"

// SelfImproveEnabled and SelfImproveProject are wired by main from the
// [self_improve] config, mirroring the package-level Version / DefaultLang
// pattern. They drive the owner-facing pending-insights endpoint that backs
// the SPA "new insights" badge. See plan 20260608-self-improve-feedback-loop.
var (
	SelfImproveEnabled bool
	SelfImproveProject = "insights"
)

type insightRef struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type pendingInsightsResponse struct {
	Enabled bool         `json:"enabled"`
	Project string       `json:"project"`
	Count   int          `json:"count"`
	Notes   []insightRef `json:"notes"`
}

// handleInsightsPending returns the owner's un-triaged self-improvement
// insights (type:insight + status:pending) in the configured project.
// Owner-only. When the loop is disabled it returns enabled:false with an
// empty list (200) so the SPA can hide the badge cleanly rather than 404.
func (r *Router) handleInsightsPending(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	resp := pendingInsightsResponse{
		Enabled: SelfImproveEnabled,
		Project: SelfImproveProject,
		Notes:   []insightRef{},
	}
	if !SelfImproveEnabled || r.deps.Index == nil {
		WriteJSON(w, http.StatusOK, resp)
		return
	}
	pending, err := r.deps.Index.NotesByTagInProject("status:pending", SelfImproveProject)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "pending insights lookup failed")
		return
	}
	insights, err := r.deps.Index.NotesByTagInProject("type:insight", SelfImproveProject)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "pending insights lookup failed")
		return
	}
	isInsight := make(map[string]struct{}, len(insights))
	for _, n := range insights {
		isInsight[n.Path] = struct{}{}
	}
	for _, n := range pending {
		if _, ok := isInsight[n.Path]; !ok {
			continue
		}
		resp.Count++
		if len(resp.Notes) < 20 {
			resp.Notes = append(resp.Notes, insightRef{Path: n.Path, Title: n.Title})
		}
	}
	WriteJSON(w, http.StatusOK, resp)
}
