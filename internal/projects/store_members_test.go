package projects

import (
	"path/filepath"
	"testing"
)

func TestProjectMembers_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := s.MemberLevel("Work", "u1"); ok {
		t.Fatal("expected no membership initially")
	}
	if err := s.SetMember("Work", "u1", LevelWrite); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMember("Work", "u2", LevelRead); err != nil {
		t.Fatal(err)
	}
	if lvl, ok := s.MemberLevel("Work", "u1"); !ok || lvl != LevelWrite {
		t.Errorf("u1 level = %q,%v want write,true", lvl, ok)
	}

	// Update level in place (no duplicate).
	if err := s.SetMember("Work", "u1", LevelRead); err != nil {
		t.Fatal(err)
	}
	if got := s.MembersOf("Work"); len(got) != 2 {
		t.Errorf("members = %d want 2 (update must not duplicate)", len(got))
	}
	if lvl, _ := s.MemberLevel("Work", "u1"); lvl != LevelRead {
		t.Errorf("u1 level after update = %q want read", lvl)
	}

	// Invalid level rejected.
	if err := s.SetMember("Work", "u3", "admin"); err == nil {
		t.Error("expected error on invalid level")
	}

	// Persistence: reopen and re-read.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if lvl, ok := s2.MemberLevel("Work", "u2"); !ok || lvl != LevelRead {
		t.Errorf("persisted u2 = %q,%v want read,true", lvl, ok)
	}
}

func TestProjectMembers_RemoveAndCascade(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "projects.json"))
	_ = s.SetMember("A", "u1", LevelWrite)
	_ = s.SetMember("A", "u2", LevelRead)
	_ = s.SetMember("B", "u1", LevelRead)

	if err := s.RemoveMember("A", "u1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.MemberLevel("A", "u1"); ok {
		t.Error("u1 still member of A after RemoveMember")
	}
	if _, ok := s.MemberLevel("A", "u2"); !ok {
		t.Error("u2 must remain member of A")
	}

	// RemoveUserEverywhere drops u1 from B too.
	if err := s.RemoveUserEverywhere("u1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.MemberLevel("B", "u1"); ok {
		t.Error("u1 still member of B after RemoveUserEverywhere")
	}
}

func TestProjectMembers_DeleteAndRename(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "projects.json"))
	_ = s.SetMember("Old", "u1", LevelWrite)

	// Delete drops members even when the project has no Flags entry.
	if err := s.Delete("Old"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.MemberLevel("Old", "u1"); ok {
		t.Error("members survived project Delete")
	}

	// Rename moves members.
	_ = s.SetMember("Src", "u9", LevelRead)
	if err := s.Rename("Src", "Dst"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.MemberLevel("Src", "u9"); ok {
		t.Error("members left behind at old name after Rename")
	}
	if lvl, ok := s.MemberLevel("Dst", "u9"); !ok || lvl != LevelRead {
		t.Errorf("members not moved to new name: %q,%v", lvl, ok)
	}
}

func TestMemberScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	s, _ := Open(path)
	if s.MemberScope() != MemberScopeAll {
		t.Errorf("default scope = %q want all", s.MemberScope())
	}
	if err := s.SetMemberScope(MemberScopeMembers); err != nil {
		t.Fatal(err)
	}
	if s.MemberScope() != MemberScopeMembers {
		t.Errorf("scope = %q want members", s.MemberScope())
	}
	// Unknown normalizes to all.
	_ = s.SetMemberScope("bogus")
	if s.MemberScope() != MemberScopeAll {
		t.Errorf("bogus scope = %q want all", s.MemberScope())
	}
	// Persistence of the enabled mode.
	_ = s.SetMemberScope(MemberScopeMembers)
	s2, _ := Open(path)
	if s2.MemberScope() != MemberScopeMembers {
		t.Errorf("persisted scope = %q want members", s2.MemberScope())
	}
}
