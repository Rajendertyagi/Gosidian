package v1

import (
	"net/http"
	"strings"
)

// CSPHeader is the Content-Security-Policy header applied to the SPA
// shell (index.html). Strict script-src 'self' (no inline, no eval),
// with 'unsafe-inline' on style-src because Tailwind utility runtime
// and Reka UI primitives inject scoped <style> blocks. img/data and
// blob support image previews and DOMPurify-sanitized blob URLs.
// connect-src 'self' constrains XHR/fetch + EventSource to
// same-origin, closing the cross-origin exfil route an XSS would
// normally use even before our defense-in-depth DOMPurify
// sanitisation.
var CSPHeader = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self'",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob:",
	"font-src 'self'",
	"connect-src 'self'",
	"worker-src 'self' blob:",
	"frame-ancestors 'none'",
	"form-action 'self'",
	"base-uri 'self'",
	"object-src 'none'",
}, "; ")

// SecurityHeaders is the set of defense-in-depth headers attached to
// every response served by this package. Apart from CSP, they are
// cheap, well-understood, and friendly to old clients (browsers that
// don't understand them just ignore them).
//
// Intentionally NOT included:
//   - Strict-Transport-Security (HSTS): the operator may run gosidian
//     behind HTTPS via reverse proxy or directly via http; we don't
//     know which, and a wrong HSTS header can lock users out for a
//     year. Configurable in a future phase if a use case appears.
//   - X-XSS-Protection: deprecated in modern browsers; replaced by
//     CSP nonce/strict-dynamic patterns. Skipped to avoid noise.
//
// X-Frame-Options is duplicated by CSP frame-ancestors 'none' but
// older browsers (pre-Chrome 76, IE) read only XFO; cheap insurance.
var SecurityHeaders = map[string]string{
	"X-Content-Type-Options": "nosniff",
	"X-Frame-Options":        "DENY",
	"Referrer-Policy":        "strict-origin-when-cross-origin",
	"X-Permitted-Cross-Domain-Policies": "none",
}

// applySecurityHeaders is the package-level entry point used by the
// v1 router and the SPA shell handler. Idempotent — calling it twice
// just rewrites the same values.
func applySecurityHeaders(w http.ResponseWriter) {
	for k, v := range SecurityHeaders {
		w.Header().Set(k, v)
	}
}

// applyCSP attaches the CSP header. Used only by the SPA shell
// handler — applying CSP to JSON API responses adds no value and
// confuses tooling that flags Content-Type: application/json with a
// CSP as misconfigured.
func applyCSP(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", CSPHeader)
}

// SetSPAShellHeaders is the public entry point used by
// internal/server/handlers_spa.go to attach both the security
// headers and CSP to the index.html response. Lives here so the
// header set has a single source of truth — bumps to CSP/HSTS land
// in this package and propagate to every consumer.
func SetSPAShellHeaders(w http.ResponseWriter) {
	applySecurityHeaders(w)
	applyCSP(w)
}

// securityHeadersMW wraps an http.Handler so every response passes
// through applySecurityHeaders. Used in the v1 router middleware
// stack after the JSON content-type middleware.
func securityHeadersMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applySecurityHeaders(w)
		next.ServeHTTP(w, r)
	})
}
