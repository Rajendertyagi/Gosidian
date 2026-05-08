package server

import (
	"io/fs"
	"net/http"
	"strings"

	apiv1 "github.com/gosidian/gosidian/internal/api/v1"
	"github.com/gosidian/gosidian/internal/server/web"
)

// handleSPA serves the embedded Vue 3 build for any URL that does
// not belong to the API or static asset surface. Vue Router runs in
// history mode, so a refresh on /notes/foo or /admin/users must
// return index.html with a 200 — the client-side router then
// resolves the route and renders the matching view.
//
// Routes that already have a server-side handler (API, static, MCP,
// healthz, metrics, vault-files) match earlier in the mux and never
// reach this fallback. We only guard against accidentally serving
// the SPA shell for asset-style requests that fell through (e.g. a
// missing /static/foo.js): those return 404 instead of the HTML
// shell so DevTools shows the real failure rather than an opaque
// "module loaded but is HTML".
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// Don't capture /api/* paths that fell through (anything not
	// under /api/v1/). Returning the SPA shell on legacy /api/preview
	// etc. masks the migration breakage; 404 makes it visible.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	if looksLikeAsset(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(web.DistFS, "dist/index.html")
	if err != nil {
		http.Error(w, "SPA build missing — run `npm run build` in web/", http.StatusInternalServerError)
		return
	}
	apiv1.SetSPAShellHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store") // shell rotates per release
	_, _ = w.Write(data)
}

// handleSpaStatic serves /static/dist/* directly from the embedded
// FS. Vite's manifest fingerprints filenames so we can cache
// aggressively.
func (s *Server) handleSpaStatic(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/static/dist")
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.FileServerFS(sub).ServeHTTP(w, r2)
}

// looksLikeAsset returns true for path shapes the SPA shell should
// not capture. The list mirrors Vite output conventions: .js/.css/
// fonts/images requested outside /static/dist/ are misconfigured
// callers, not router targets.
func looksLikeAsset(p string) bool {
	if !strings.Contains(p, ".") {
		return false
	}
	p = strings.TrimSuffix(p, "/")
	for _, ext := range assetExtensions {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

var assetExtensions = []string{
	".js", ".mjs", ".css", ".map",
	".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
	".woff", ".woff2", ".ttf", ".eot",
	".json", ".wasm",
}
