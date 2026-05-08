package v1

import (
	"net/http"
	"strings"
	"time"

	"github.com/gosidian/gosidian/internal/audit"
)

// inviteDefaultTTL mirrors the value the v1.x HTML handler used so
// invites generated from the SPA admin page age the same way as
// those minted from the legacy form. 24h is long enough for the
// owner to share the link asynchronously and short enough to limit
// blast radius if the token leaks.
const inviteDefaultTTL = 24 * time.Hour

type adminUserView struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	CreatedAt  string `json:"created_at"`
	DisabledAt string `json:"disabled_at,omitempty"`
}

type inviteView struct {
	Token      string `json:"token"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at"`
	ConsumedBy string `json:"consumed_by,omitempty"`
	ConsumedAt string `json:"consumed_at,omitempty"`
	Pending    bool   `json:"pending"`
}

// ---- /api/v1/admin/users ----

func (r *Router) handleAdminUsers(w http.ResponseWriter, req *http.Request) {
	if r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "web auth not configured")
		return
	}
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	users := r.deps.Auth.WebAuth.ListUsers()
	out := make([]adminUserView, 0, len(users))
	for _, u := range users {
		uv := adminUserView{
			ID:        u.ID,
			Username:  u.Username,
			Role:      string(u.Role),
			CreatedAt: u.CreatedAt.UTC().Format(rfc3339Z),
		}
		if u.DisabledAt != nil {
			uv.DisabledAt = u.DisabledAt.UTC().Format(rfc3339Z)
		}
		out = append(out, uv)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

// handleAdminUserItem covers DELETE → DisableUser. The webauth API
// has no symmetric Enable, so re-activating a disabled user requires
// editing accounts.json directly. That's intentional — disabling is a
// safety action and the explicit lockout is a feature, not a gap.
func (r *Router) handleAdminUserItem(w http.ResponseWriter, req *http.Request) {
	if r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "web auth not configured")
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/v1/admin/users/"), "/")
	if id == "" || strings.Contains(id, "/") {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, "expected /api/v1/admin/users/{id}")
		return
	}
	if req.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if err := r.deps.Auth.WebAuth.DisableUser(id); err != nil {
		// DisableUser returns a clear "not found" or "owner cannot be
		// disabled" — surface verbatim so the SPA can render a useful
		// message.
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, http.StatusNotFound, CodeNotFound, err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, err.Error())
		return
	}
	// Cascade: revoke MCP tokens owned by this user. The cascade hook
	// is normally wired by main.go but the API path also calls it
	// inline so admin-initiated disable doesn't depend on cmd/main
	// wiring order.
	if r.deps.Auth.MCPTokens != nil {
		_ = r.deps.Auth.MCPTokens.RevokeByOwner(id)
	}
	if r.deps.Auth.SpaAuth != nil {
		_ = r.deps.Auth.SpaAuth.RevokeByUser(id)
	}
	if user := UserFromContext(req.Context()); user != nil && r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{
			Source: audit.SourceHTTP,
			Actor:  user.Username,
			UserID: user.ID,
			Action: "user_disable",
			Path:   id,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- /api/v1/admin/invites ----

func (r *Router) handleAdminInvites(w http.ResponseWriter, req *http.Request) {
	if r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "web auth not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		r.listInvites(w, req)
	case http.MethodPost:
		r.createInvite(w, req)
	default:
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
	}
}

func (r *Router) listInvites(w http.ResponseWriter, _ *http.Request) {
	invites := r.deps.Auth.WebAuth.ListInvites()
	out := make([]inviteView, 0, len(invites))
	for _, iv := range invites {
		view := inviteView{
			Token:     iv.Token,
			CreatedBy: iv.CreatedBy,
			CreatedAt: iv.CreatedAt.UTC().Format(rfc3339Z),
			ExpiresAt: iv.ExpiresAt.UTC().Format(rfc3339Z),
			Pending:   iv.Pending(),
		}
		if iv.ConsumedAt != nil {
			view.ConsumedBy = iv.ConsumedBy
			view.ConsumedAt = iv.ConsumedAt.UTC().Format(rfc3339Z)
		}
		out = append(out, view)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

func (r *Router) createInvite(w http.ResponseWriter, req *http.Request) {
	user := UserFromContext(req.Context())
	if user == nil {
		WriteError(w, http.StatusUnauthorized, CodeAuthTokenInvalid, "no user in context")
		return
	}
	// Body is optional; accept empty or {} as defaults.
	var body struct {
		TTLMS int64 `json:"ttl_ms,omitempty"`
	}
	if req.ContentLength > 0 {
		if err := DecodeJSON(req, &body); err != nil {
			WriteError(w, http.StatusBadRequest, CodeValidationFormat, err.Error())
			return
		}
	}
	ttl := inviteDefaultTTL
	if body.TTLMS > 0 {
		ttl = time.Duration(body.TTLMS) * time.Millisecond
	}
	inv, err := r.deps.Auth.WebAuth.CreateInvite(user.ID, ttl)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	if r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{
			Source: audit.SourceHTTP,
			Actor:  user.Username,
			UserID: user.ID,
			Action: "invite_create",
			Path:   inv.Token,
		})
	}
	WriteJSON(w, http.StatusCreated, inviteView{
		Token:     inv.Token,
		CreatedBy: inv.CreatedBy,
		CreatedAt: inv.CreatedAt.UTC().Format(rfc3339Z),
		ExpiresAt: inv.ExpiresAt.UTC().Format(rfc3339Z),
		Pending:   true,
	})
}

func (r *Router) handleAdminInviteItem(w http.ResponseWriter, req *http.Request) {
	if r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "web auth not configured")
		return
	}
	token := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/v1/admin/invites/"), "/")
	if token == "" || strings.Contains(token, "/") {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, "expected /api/v1/admin/invites/{token}")
		return
	}
	if req.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	if err := r.deps.Auth.WebAuth.RevokeInvite(token); err != nil {
		WriteError(w, http.StatusNotFound, CodeNotFound, err.Error())
		return
	}
	if user := UserFromContext(req.Context()); user != nil && r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{
			Source: audit.SourceHTTP,
			Actor:  user.Username,
			UserID: user.ID,
			Action: "invite_revoke",
			Path:   token,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}
