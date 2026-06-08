package v1

import (
	"net/http"
	"strings"

	"github.com/gosidian/gosidian/internal/authz"
	"github.com/gosidian/gosidian/internal/webauth"
)

// principal projects a RequestUser onto the authz.Principal used by the
// shared authorization predicate.
func (ru *RequestUser) principal() authz.Principal {
	return authz.Principal{UserID: ru.ID, Role: ru.Role}
}

// principalFromContext returns the Principal for the current request. Every
// authed route runs requireAuth first, so a user is normally present; if it
// is somehow missing we fall back to a guest-role principal, which fails
// closed (sees only public projects, cannot write).
func principalFromContext(req *http.Request) authz.Principal {
	if u := UserFromContext(req.Context()); u != nil {
		return u.principal()
	}
	return authz.Principal{Role: webauth.RoleGuest}
}

// isPublic reports a project's Public flag, nil-safe. With no projects store
// configured nothing is public, so guests see nothing (fail closed).
func (r *Router) isPublic(name string) bool {
	if r.deps.Projects == nil {
		return false
	}
	return r.deps.Projects.IsPublic(name)
}

// projectOf returns the top-level project folder of a vault-relative note
// path ("gosidian/plans/x.md" -> "gosidian"). A path without a slash has no
// project folder and is returned as-is; since such a name won't match a
// Public project, guests are denied — consistent with private-by-default.
func projectOf(path string) string {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

// canSee reports whether the principal may read the note/path. Centralizes
// the projectOf + CanAccessProject pairing used by every read handler.
func (r *Router) canSee(p authz.Principal, path string) bool {
	return p.CanAccessProject(projectOf(path), r.isPublic)
}

// denyGuestWrite writes a 403 and returns true when the user may not mutate
// (guest or any non-writing role). Mutating handlers call it right after the
// user nil-check: `if denyGuestWrite(w, user) { return }`.
func denyGuestWrite(w http.ResponseWriter, user *RequestUser) bool {
	if user != nil && user.principal().CanWrite() {
		return false
	}
	WriteError(w, http.StatusForbidden, CodeAuthForbidden, "read-only role cannot write")
	return true
}
