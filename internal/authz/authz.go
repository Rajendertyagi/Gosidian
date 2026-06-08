// Package authz centralizes gosidian's role-based access decisions so the HTTP
// API (internal/api/v1) and the MCP server (internal/mcp) share one source of
// truth for "who may see/do what". Spreading these checks across individual
// handlers is a security hazard — a single forgotten endpoint leaks data — so
// every project-scoped read and every mutation funnels through here.
package authz

import "github.com/gosidian/gosidian/internal/webauth"

// Principal is the authenticated identity behind a request, distilled to the
// fields that drive authorization. It is built from a SPA session user (HTTP)
// or from the webauth user that owns an MCP token.
type Principal struct {
	UserID string
	Role   webauth.Role
}

// CanWrite reports whether the principal may create/edit/delete notes and
// projects. Owner and member; guests are read-only.
func (p Principal) CanWrite() bool { return p.Role.CanWrite() }

// CanAdmin reports whether the principal may perform owner-only administration
// (users, tokens, invites, audit, global settings).
func (p Principal) CanAdmin() bool { return p.Role.CanAdmin() }

// IsGuest reports whether the principal holds the restricted guest role.
func (p Principal) IsGuest() bool { return p.Role.IsGuest() }

// CanSeeAllProjects reports whether the principal sees every project (owner or
// member). Guests and unrecognized roles see only public projects.
func (p Principal) CanSeeAllProjects() bool {
	return p.Role == webauth.RoleOwner || p.Role == webauth.RoleMember
}

// CanAccessProject reports whether the principal may read the named project.
// Owner and member see every project; a guest sees only projects for which
// isPublic returns true. isPublic must report a project's Public flag (e.g.
// projects.Store.IsPublic); passing nil treats every project as private.
func (p Principal) CanAccessProject(project string, isPublic func(string) bool) bool {
	if p.CanSeeAllProjects() {
		return true
	}
	// Guest — or any unrecognized/zero-value role — sees only public projects.
	// Failing closed here means a malformed Principal never widens access.
	return isPublic != nil && isPublic(project)
}

// VisibleProjects returns the subset of all that the principal may read,
// preserving input order. For owner/member this is all; for a guest it is the
// public subset.
func (p Principal) VisibleProjects(all []string, isPublic func(string) bool) []string {
	if p.CanSeeAllProjects() {
		return all
	}
	out := make([]string, 0, len(all))
	for _, name := range all {
		if isPublic != nil && isPublic(name) {
			out = append(out, name)
		}
	}
	return out
}
