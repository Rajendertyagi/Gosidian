package v1

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/webauth"
)

// TestAdminCreateUser covers POST /api/v1/admin/users: the owner-driven account
// creation that complements the invite + /signup self-service flow.
func TestAdminCreateUser(t *testing.T) {
	f := newNotesFixture(t)

	// Happy path: member with an initial TOTP policy applied at creation.
	body := `{"username":"alice","password":"alice-pass-123","role":"member","totp_policy":"enabled"}`
	rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", body, nil)
	if rec.code != http.StatusCreated {
		t.Fatalf("create status %d want 201: %s", rec.code, rec.body)
	}
	var uv struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		Role       string `json:"role"`
		TOTPPolicy string `json:"totp_policy"`
	}
	if err := json.Unmarshal([]byte(rec.body), &uv); err != nil {
		t.Fatal(err)
	}
	if uv.Username != "alice" || uv.Role != "member" {
		t.Errorf("unexpected created view: %s", rec.body)
	}
	if u, ok := f.webauth.UserByID(uv.ID); !ok || u.TOTPPolicy != webauth.TOTPEnabled {
		t.Errorf("initial totp policy not applied at creation")
	}

	// The new account is listed.
	if rec := f.doAuthRecorder(http.MethodGet, "/api/v1/admin/users", "", nil); !strings.Contains(rec.body, `"alice"`) {
		t.Errorf("created user not listed: %s", rec.body)
	}

	// Duplicate username → 409.
	if rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", body, nil); rec.code != http.StatusConflict {
		t.Errorf("duplicate username status %d want 409: %s", rec.code, rec.body)
	}

	// Guest role is allowed; role defaults aside, owner is rejected.
	if rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", `{"username":"bob","password":"bob-pass-1234","role":"guest"}`, nil); rec.code != http.StatusCreated {
		t.Errorf("guest create status %d want 201: %s", rec.code, rec.body)
	}
	if rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", `{"username":"dave","password":"dave-pass-1234","role":"owner"}`, nil); rec.code != http.StatusBadRequest {
		t.Errorf("owner create status %d want 400: %s", rec.code, rec.body)
	}

	// Password shorter than 8 chars → 400 (webauth.AddUser validation).
	if rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", `{"username":"carol","password":"short","role":"member"}`, nil); rec.code != http.StatusBadRequest {
		t.Errorf("weak password status %d want 400: %s", rec.code, rec.body)
	}

	// Bad totp_policy → 400, and no orphan account is left behind.
	if rec := f.doAuthRecorder(http.MethodPost, "/api/v1/admin/users", `{"username":"erin","password":"erin-pass-1234","role":"member","totp_policy":"bogus"}`, nil); rec.code != http.StatusBadRequest {
		t.Errorf("bad totp_policy status %d want 400: %s", rec.code, rec.body)
	}
	for _, u := range f.webauth.ListUsers() {
		if u.Username == "erin" {
			t.Error("invalid totp_policy must not create an account")
		}
	}
}
