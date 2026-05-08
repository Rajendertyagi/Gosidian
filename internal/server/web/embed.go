// Package web embeds the Vue 3 SPA build output (web/dist) into the
// gosidian binary so the single-binary deployment story survives the
// v2.0 frontend rewrite. The dist tree is produced by `npm run build`
// in the sibling web/ project (see web/vite.config.ts) and copied into
// internal/server/web/dist by the Dockerfile multi-stage build.
//
// During development the placeholder index.html in dist/ is enough to
// satisfy the //go:embed directive (Go fails to compile if the
// pattern matches zero files). At deploy time Vite overwrites the
// placeholder with the real bundle.
//
// The handlers_spa.go catch-all serves DistFS for any route the SPA
// owns; static asset paths (`/static/dist/*`) bypass the catch-all
// and are served directly via http.FileServerFS so caching and
// fingerprinting work as Vite intends.
package web

import "embed"

//go:embed all:dist
var DistFS embed.FS
