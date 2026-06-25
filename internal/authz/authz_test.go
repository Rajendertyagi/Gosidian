package authz

import (
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

// lcfg builds a legacy-mode AccessConfig (membership not enforced) from an
// isPublic predicate — the pre-feature behavior most of these tests assert.
func lcfg(isPub func(string) bool) AccessConfig { return AccessConfig{IsPublic: isPub} }

func TestCapabilities(t *testing.T) {
	cases := []struct {
		role                webauth.Role
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

// Legacy mode (member_scope unset): guest sees only public; owner/member see all.
func TestProjectVisibility(t *testing.T) {
	isPub := pub("docs")

	guest := Principal{Role: webauth.RoleGuest}
	if guest.CanAccessProject("secret", lcfg(isPub)) {
		t.Error("guest must NOT access a private project")
	}
	if !guest.CanAccessProject("docs", lcfg(isPub)) {
		t.Error("guest must access a public project")
	}

	for _, role := range []webauth.Role{webauth.RoleOwner, webauth.RoleMember} {
		p := Principal{Role: role}
		if !p.CanAccessProject("secret", lcfg(isPub)) {
			t.Errorf("%s must access every project in legacy mode", role)
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
	if p.CanAccessProject("secret", lcfg(pub("docs"))) {
		t.Error("unknown role must not access a private project")
	}
	if !p.CanAccessProject("docs", lcfg(pub("docs"))) {
		t.Error("unknown role may still read a public project")
	}
}

// A nil isPublic predicate must fail closed: guests see nothing.
func TestGuestFailsClosedOnNilPredicate(t *testing.T) {
	guest := Principal{Role: webauth.RoleGuest}
	if guest.CanAccessProject("docs", AccessConfig{}) {
		t.Error("guest with nil IsPublic must be denied")
	}
}
