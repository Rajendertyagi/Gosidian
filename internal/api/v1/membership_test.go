package v1

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/projects"
	"github.com/gosidian/gosidian/internal/webauth"
)

// memberUser creates a member account and mints a SPA bearer for it.
func (f *notesFixture) memberUser(t *testing.T, name string) (*webauth.User, string) {
	t.Helper()
	u, err := f.webauth.AddUser(name, name+"-pass-1234", webauth.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	bearer, _, err := f.spaTokens.Create(u.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	return u, bearer
}

func TestProjectMembership_Enforcement(t *testing.T) {
	f := newNotesFixture(t)
	f.seedNote(t, "Alpha/a.md", "x")
	f.seedNote(t, "Beta/b.md", "x")

	m1, bearer := f.memberUser(t, "m1")
	hdr := map[string]string{"Authorization": "Bearer " + bearer}

	// Legacy mode (default): the member sees both projects.
	if rec := f.request(http.MethodGet, "/api/v1/projects", "", hdr); !strings.Contains(rec.Body.String(), "Alpha") || !strings.Contains(rec.Body.String(), "Beta") {
		t.Fatalf("legacy: member must see all projects: %s", rec.Body.String())
	}

	// Enable per-project membership enforcement.
	if err := f.projects.SetMemberScope(projects.MemberScopeMembers); err != nil {
		t.Fatal(err)
	}

	// A member with no memberships now sees nothing, and private notes 404 (no leak).
	if rec := f.request(http.MethodGet, "/api/v1/projects", "", hdr); strings.Contains(rec.Body.String(), "Alpha") || strings.Contains(rec.Body.String(), "Beta") {
		t.Errorf("enforced: non-member must see no projects: %s", rec.Body.String())
	}
	if rec := f.request(http.MethodGet, "/api/v1/notes/Beta/b.md", "", hdr); rec.Code != http.StatusNotFound {
		t.Errorf("enforced: non-member note read = %d want 404", rec.Code)
	}

	// Grant read on Alpha: sees Alpha (not Beta), reads it, but cannot write it.
	if err := f.projects.SetMember("Alpha", m1.ID, projects.LevelRead); err != nil {
		t.Fatal(err)
	}
	if rec := f.request(http.MethodGet, "/api/v1/projects", "", hdr); !strings.Contains(rec.Body.String(), "Alpha") || strings.Contains(rec.Body.String(), "Beta") {
		t.Errorf("enforced: read-member must see only Alpha: %s", rec.Body.String())
	}
	if rec := f.request(http.MethodGet, "/api/v1/notes/Alpha/a.md", "", hdr); rec.Code != http.StatusOK {
		t.Errorf("read-member Alpha read = %d want 200", rec.Code)
	}
	if rec := f.request(http.MethodPut, "/api/v1/notes/Alpha/a.md", `{"content":"y"}`, hdr); rec.Code != http.StatusForbidden {
		t.Errorf("read-member Alpha write = %d want 403 (%s)", rec.Code, rec.Body.String())
	}

	// Upgrade to write: now the member may write Alpha, still not Beta.
	if err := f.projects.SetMember("Alpha", m1.ID, projects.LevelWrite); err != nil {
		t.Fatal(err)
	}
	if rec := f.request(http.MethodPut, "/api/v1/notes/Alpha/a.md", `{"content":"y"}`, hdr); rec.Code != http.StatusOK {
		t.Errorf("write-member Alpha write = %d want 200 (%s)", rec.Code, rec.Body.String())
	}
	if rec := f.request(http.MethodPut, "/api/v1/notes/Beta/b.md", `{"content":"z"}`, hdr); rec.Code != http.StatusForbidden {
		t.Errorf("write-member Beta write = %d want 403", rec.Code)
	}
}

func TestProjectMembers_CRUD(t *testing.T) {
	f := newNotesFixture(t)
	f.seedNote(t, "Shared/n.md", "x")
	m1, bearer := f.memberUser(t, "alice")
	hdr := map[string]string{"Authorization": "Bearer " + bearer}

	// Owner adds alice as a write member.
	if rec := f.doAuthRecorder(http.MethodPut, "/api/v1/projects/Shared/members", `{"user_id":"`+m1.ID+`","level":"write"}`, nil); rec.code != http.StatusCreated {
		t.Fatalf("add member = %d %s", rec.code, rec.body)
	}
	if rec := f.doAuthRecorder(http.MethodGet, "/api/v1/projects/Shared/members", "", nil); !strings.Contains(rec.body, "alice") || !strings.Contains(rec.body, "write") {
		t.Errorf("member list missing alice: %s", rec.body)
	}

	// Non-owner cannot manage members.
	if rec := f.request(http.MethodGet, "/api/v1/projects/Shared/members", "", hdr); rec.Code != http.StatusForbidden {
		t.Errorf("member-mgmt as non-owner = %d want 403", rec.Code)
	}

	// Invalid level / unknown user rejected.
	if rec := f.doAuthRecorder(http.MethodPut, "/api/v1/projects/Shared/members", `{"user_id":"`+m1.ID+`","level":"admin"}`, nil); rec.code != http.StatusBadRequest {
		t.Errorf("invalid level = %d want 400", rec.code)
	}
	if rec := f.doAuthRecorder(http.MethodPut, "/api/v1/projects/Shared/members", `{"user_id":"ghost","level":"read"}`, nil); rec.code != http.StatusNotFound {
		t.Errorf("unknown user = %d want 404", rec.code)
	}

	// Remove.
	if rec := f.doAuthRecorder(http.MethodDelete, "/api/v1/projects/Shared/members/"+m1.ID, "", nil); rec.code != http.StatusNoContent {
		t.Errorf("remove member = %d want 204 (%s)", rec.code, rec.body)
	}
	if _, ok := f.projects.MemberLevel("Shared", m1.ID); ok {
		t.Error("membership not removed")
	}
}

func TestProjectMembership_DisableCascade(t *testing.T) {
	f := newNotesFixture(t)
	f.seedNote(t, "P/n.md", "x")
	m1, _ := f.memberUser(t, "bob")
	if err := f.projects.SetMember("P", m1.ID, projects.LevelWrite); err != nil {
		t.Fatal(err)
	}
	if rec := f.doAuthRecorder(http.MethodDelete, "/api/v1/admin/users/"+m1.ID, "", nil); rec.code != http.StatusNoContent {
		t.Fatalf("disable = %d %s", rec.code, rec.body)
	}
	if _, ok := f.projects.MemberLevel("P", m1.ID); ok {
		t.Error("membership not stripped on user disable")
	}
}

func TestProjectMembership_CreatorAutoMembership(t *testing.T) {
	f := newNotesFixture(t)
	if err := f.projects.SetMemberScope(projects.MemberScopeMembers); err != nil {
		t.Fatal(err)
	}
	_, bearer := f.memberUser(t, "carol")
	hdr := map[string]string{"Authorization": "Bearer " + bearer}

	// A member creates a project and must retain write access to it.
	if rec := f.request(http.MethodPost, "/api/v1/projects", `{"name":"Carol"}`, hdr); rec.Code != http.StatusCreated {
		t.Fatalf("create project = %d (%s)", rec.Code, rec.Body.String())
	}
	if rec := f.request(http.MethodPost, "/api/v1/notes", `{"path":"Carol/x.md","content":"c"}`, hdr); rec.Code != http.StatusCreated {
		t.Errorf("creator write to own project = %d want 201 (%s)", rec.Code, rec.Body.String())
	}
}
