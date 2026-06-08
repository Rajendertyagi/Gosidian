package authz

import (
	"reflect"
	"testing"

	"github.com/gosidian/gosidian/internal/webauth"
)

// pub builds an isPublic predicate from a set of public project names.
func pub(names ...string) func(string) bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return func(p string) bool { return m[p] }
}

func TestCapabilities(t *testing.T) {
	cases := []struct {
		role                       webauth.Role
		write, admin, guest bool
	}{
		{webauth.RoleOwner, true, true, false},
		{webauth.RoleMember, true, false, false},
		{webauth.RoleGuest, false, false, true},
	}
	for _, c := range cases {
		p := Principal{Role: c.role}
		if p.CanWrite() != c.write {
			t.Errorf("%s CanWrite=%v want %v", c.role, p.CanWrite(), c.write)
		}
		if p.CanAdmin() != c.admin {
			t.Errorf("%s CanAdmin=%v want %v", c.role, p.CanAdmin(), c.admin)
		}
		if p.IsGuest() != c.guest {
			t.Errorf("%s IsGuest=%v want %v", c.role, p.IsGuest(), c.guest)
		}
	}
}

func TestProjectVisibility(t *testing.T) {
	all := []string{"alpha", "docs", "secret"}
	isPub := pub("docs")

	guest := Principal{Role: webauth.RoleGuest}
	if got := guest.VisibleProjects(all, isPub); !reflect.DeepEqual(got, []string{"docs"}) {
		t.Errorf("guest VisibleProjects=%v want [docs]", got)
	}
	if guest.CanAccessProject("secret", isPub) {
		t.Error("guest must NOT access a private project")
	}
	if !guest.CanAccessProject("docs", isPub) {
		t.Error("guest must access a public project")
	}

	for _, role := range []webauth.Role{webauth.RoleOwner, webauth.RoleMember} {
		p := Principal{Role: role}
		if got := p.VisibleProjects(all, isPub); !reflect.DeepEqual(got, all) {
			t.Errorf("%s VisibleProjects=%v want all", role, got)
		}
		if !p.CanAccessProject("secret", isPub) {
			t.Errorf("%s must access every project", role)
		}
	}
}

// An unrecognized/zero-value role must fail closed: no write, no admin,
// public-only reads, so a malformed Principal never widens access.
func TestUnknownRoleFailsClosed(t *testing.T) {
	p := Principal{Role: webauth.Role("")}
	if p.CanWrite() || p.CanAdmin() {
		t.Error("unknown role must not write or admin")
	}
	if p.CanAccessProject("secret", pub("docs")) {
		t.Error("unknown role must not access a private project")
	}
	if got := p.VisibleProjects([]string{"docs", "secret"}, pub("docs")); !reflect.DeepEqual(got, []string{"docs"}) {
		t.Errorf("unknown role VisibleProjects=%v want [docs]", got)
	}
}

// A nil isPublic predicate must fail closed: guests see nothing.
func TestGuestFailsClosedOnNilPredicate(t *testing.T) {
	guest := Principal{Role: webauth.RoleGuest}
	if guest.CanAccessProject("docs", nil) {
		t.Error("guest with nil isPublic must be denied")
	}
	if got := guest.VisibleProjects([]string{"a", "b"}, nil); len(got) != 0 {
		t.Errorf("guest with nil isPublic VisibleProjects=%v want empty", got)
	}
}
