package v1

import (
	"net/http"

	"github.com/gosidian/gosidian/internal/audit"
	"github.com/gosidian/gosidian/internal/webauth"
)

// handleAuthConfig is a PUBLIC endpoint the LoginView fetches before
// authentication to decide whether to render the TOTP field (and, in Phase 3,
// surface the LDAP option). It returns only booleans — never which usernames
// have TOTP — so probing it leaks nothing.
func (r *Router) handleAuthConfig(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	totp := false
	if r.deps.Auth != nil && r.deps.Auth.WebAuth != nil {
		totp = r.deps.Auth.WebAuth.TOTPMode() != webauth.TOTPOff
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"totp": totp,
		"ldap": r.deps.Auth != nil && r.deps.Auth.LDAP != nil,
	})
}

type totpEnrollResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURI string `json:"otpauth_uri"`
}

// handleTOTPEnroll generates a fresh secret + provisioning URI. The secret is
// NOT persisted until the user confirms a valid code via handleTOTPConfirm.
func (r *Router) handleTOTPEnroll(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	user := UserFromContext(req.Context())
	if user == nil || r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusUnauthorized, CodeAuthTokenInvalid, "no user in context")
		return
	}
	if user.isAnonymous() {
		WriteError(w, http.StatusForbidden, CodeAuthForbidden, "anonymous session has no account to manage")
		return
	}
	secret, uri, err := r.deps.Auth.WebAuth.GenerateTOTPSecret(user.Username, "gosidian")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "totp: "+err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, totpEnrollResponse{Secret: secret, OTPAuthURI: uri})
}

type totpConfirmRequest struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

// handleTOTPConfirm validates a code against the candidate secret from
// /totp/enroll and, on success, activates it for the user.
func (r *Router) handleTOTPConfirm(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	user := UserFromContext(req.Context())
	if user == nil || r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusUnauthorized, CodeAuthTokenInvalid, "no user in context")
		return
	}
	if user.isAnonymous() {
		WriteError(w, http.StatusForbidden, CodeAuthForbidden, "anonymous session has no account to manage")
		return
	}
	var body totpConfirmRequest
	if err := DecodeJSON(req, &body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, err.Error())
		return
	}
	if body.Secret == "" || body.Code == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationRequired, "secret and code are required")
		return
	}
	if !webauth.ValidateTOTPCode(body.Secret, body.Code) {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, "invalid code")
		return
	}
	if err := r.deps.Auth.WebAuth.SetTOTPSecret(user.ID, body.Secret); err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	if r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{Source: audit.SourceHTTP, Actor: user.Username, UserID: user.ID, Action: "totp_enroll", Path: user.ID})
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTOTPDisenroll removes the user's TOTP secret, unless their effective
// policy requires it (403). DELETE /api/v1/totp.
func (r *Router) handleTOTPDisenroll(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}
	user := UserFromContext(req.Context())
	if user == nil || r.deps.Auth == nil || r.deps.Auth.WebAuth == nil {
		WriteError(w, http.StatusUnauthorized, CodeAuthTokenInvalid, "no user in context")
		return
	}
	if user.isAnonymous() {
		WriteError(w, http.StatusForbidden, CodeAuthForbidden, "anonymous session has no account to manage")
		return
	}
	// Re-load the full account: RequestUser carries only id/username/role, but
	// the required-policy check needs the per-user TOTPPolicy.
	if full, ok := r.deps.Auth.WebAuth.UserByID(user.ID); ok && r.deps.Auth.WebAuth.TOTPRequired(full) {
		WriteError(w, http.StatusForbidden, CodeAuthForbidden, "TOTP is required for your account and cannot be removed")
		return
	}
	if err := r.deps.Auth.WebAuth.SetTOTPSecret(user.ID, ""); err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	if r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{Source: audit.SourceHTTP, Actor: user.Username, UserID: user.ID, Action: "totp_disenroll", Path: user.ID})
	}
	w.WriteHeader(http.StatusNoContent)
}
