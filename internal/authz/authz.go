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

// AccessConfig carries the per-request inputs the project-access predicate needs
// beyond the principal's role: the public-flag lookup, the membership lookups,
// and whether per-project membership is enforced (member_scope = members). The
// membership funcs are resolved by the caller (which owns the projects store) so
// this package stays free of storage concerns and the level→bool mapping.
type AccessConfig struct {
	// Enforced is true under member_scope = members: private projects are gated
	// behind explicit membership. When false (legacy default) owner/member see
	// every project, preserving pre-feature behavior.
	Enforced bool
	// IsPublic reports a project's Public flag. nil treats everything as private.
	IsPublic func(project string) bool
	// IsMember reports whether userID holds any membership of project.
	IsMember func(userID, project string) bool
	// MemberCanWrite reports whether userID's membership of project grants write.
	MemberCanWrite func(userID, project string) bool
}

// CanAccessProject reports whether the principal may read the named project.
// Owner always; public projects are visible to everyone (guests included); in
// legacy mode member sees every project; under enforcement access requires an
// explicit membership. Fails closed for an unrecognized/zero-value role.
func (p Principal) CanAccessProject(project string, cfg AccessConfig) bool {
	if p.Role == webauth.RoleOwner {
		return true
	}
	if cfg.IsPublic != nil && cfg.IsPublic(project) {
		return true
	}
	if !cfg.Enforced {
		return p.Role == webauth.RoleMember
	}
	return cfg.IsMember != nil && cfg.IsMember(p.UserID, project)
}

// CanWriteProject reports whether the principal may mutate notes/attachments in
// the project. Owner always; guests never (read-only regardless of membership);
// in legacy mode member writes anywhere; under enforcement a member needs a
// write-level membership. Fails closed.
func (p Principal) CanWriteProject(project string, cfg AccessConfig) bool {
	if p.Role == webauth.RoleOwner {
		return true
	}
	if !p.Role.CanWrite() {
		return false
	}
	if !cfg.Enforced {
		return true
	}
	return cfg.MemberCanWrite != nil && cfg.MemberCanWrite(p.UserID, project)
}
