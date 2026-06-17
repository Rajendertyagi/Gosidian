package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/webauth"
)

// TestRequireAuth_OpenMode locks the BUG-018 contract: with OpenMode on, a
// token-less request is admitted as an anonymous guest (the RBAC then keeps it
// read-only / public-only downstream); with OpenMode off it is still a 401.
func TestRequireAuth_OpenMode(t *testing.T) {
	var seen *RequestUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Open-mode: token-less → anonymous guest, 200.
	rec := httptest.NewRecorder()
	(&AuthDeps{OpenMode: true}).requireAuth(next).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("open-mode token-less: status=%d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if seen == nil || seen.Role != webauth.RoleGuest {
		t.Fatalf("expected anonymous guest principal, got %+v", seen)
	}
	if seen.principal().CanWrite() {
		t.Errorf("anonymous guest must not be able to write")
	}

	// Off (default): token-less → 401.
	rec2 := httptest.NewRecorder()
	(&AuthDeps{OpenMode: false}).requireAuth(next).ServeHTTP(
		rec2, httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil))
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("open-mode off token-less: status=%d, want 401", rec2.Code)
	}
}

// TestRequestUser_isAnonymous locks the sentinel: the synthetic open-mode guest
// is flagged (so per-account routes reject it), but a REAL guest account is not
// (they keep self-service like TOTP). Nil-safe.
func TestRequestUser_isAnonymous(t *testing.T) {
	if !(&RequestUser{ID: anonymousUserID, Role: webauth.RoleGuest}).isAnonymous() {
		t.Error("synthetic anonymous principal not detected")
	}
	if (&RequestUser{ID: "abcdef0123456789", Role: webauth.RoleGuest}).isAnonymous() {
		t.Error("a real guest account must NOT be flagged anonymous")
	}
	var nilUser *RequestUser
	if nilUser.isAnonymous() {
		t.Error("nil user must not be anonymous")
	}
}

// TestVersion_OpenModeFlag confirms the public /version endpoint advertises
// open_mode so the SPA can allow a token-less guest session at boot.
func TestVersion_OpenModeFlag(t *testing.T) {
	prev := OpenMode
	t.Cleanup(func() { OpenMode = prev })

	OpenMode = true
	rec := httptest.NewRecorder()
	(&Router{}).handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"open_mode":true`) {
		t.Errorf("expected open_mode:true in /version, got %s", rec.Body.String())
	}

	OpenMode = false
	rec2 := httptest.NewRecorder()
	(&Router{}).handleVersion(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	if strings.Contains(rec2.Body.String(), "open_mode") {
		t.Errorf("open_mode should be omitted when false, got %s", rec2.Body.String())
	}
}
