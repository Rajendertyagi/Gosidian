package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToken_ProjectListAndScope(t *testing.T) {
	admin := &Token{}
	if !admin.IsAdmin() || admin.ProjectList() != nil || !admin.AllowsPath("any/note.md") || !admin.AllowsProject("x") {
		t.Fatal("empty-scope token must behave as admin")
	}
	if admin.ScopeLabel() != "(admin)" {
		t.Fatalf("admin ScopeLabel = %q", admin.ScopeLabel())
	}

	legacy := &Token{Project: "p"}
	if legacy.IsAdmin() {
		t.Fatal("legacy single-project token must not be admin")
	}
	if got := legacy.ProjectList(); len(got) != 1 || got[0] != "p" {
		t.Fatalf("legacy ProjectList = %v", got)
	}
	if !legacy.AllowsPath("p/note.md") || !legacy.AllowsPath("p") || legacy.AllowsPath("q/note.md") || legacy.AllowsPath("pp/note.md") {
		t.Fatal("legacy AllowsPath scope broken")
	}

	multi := &Token{Projects: []string{"pa", "pb"}}
	if multi.IsAdmin() {
		t.Fatal("multi-project token must not be admin")
	}
	for _, ok := range []string{"pa/x.md", "pb/sub/y.md", "pa", "pb"} {
		if !multi.AllowsPath(ok) {
			t.Fatalf("multi AllowsPath(%q) = false, want true", ok)
		}
	}
	for _, no := range []string{"pc/z.md", "pab/z.md", ""} {
		if multi.AllowsPath(no) {
			t.Fatalf("multi AllowsPath(%q) = true, want false", no)
		}
	}
	if !multi.AllowsProject("pb") || multi.AllowsProject("pc") {
		t.Fatal("multi AllowsProject broken")
	}
	if multi.ScopeLabel() != "pa,pb" {
		t.Fatalf("multi ScopeLabel = %q", multi.ScopeLabel())
	}
}

func TestStore_CreateMultiProjectRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Dirty input: duplicates and blanks are normalized away.
	plain, tok, err := s.Create("orchestrator", []string{"pa", " pb ", "pa", ""}, []string{ScopeRead, ScopeWrite}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if tok.Project != "pa" {
		t.Fatalf("legacy Project field = %q, want first project", tok.Project)
	}
	if got := tok.ProjectList(); len(got) != 2 || got[0] != "pa" || got[1] != "pb" {
		t.Fatalf("ProjectList = %v", got)
	}

	// Reload from disk: the multi scope survives and Validate returns it.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.Validate(plain)
	if err != nil {
		t.Fatal(err)
	}
	if list := got.ProjectList(); len(list) != 2 || list[1] != "pb" {
		t.Fatalf("reloaded ProjectList = %v", list)
	}

	// Single-project creation keeps the legacy shape (no projects array).
	_, single, err := s.Create("solo", []string{"pa"}, []string{ScopeRead}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if single.Project != "pa" || single.Projects != nil {
		t.Fatalf("single-project token = %+v, want legacy shape", single)
	}

	// Invalid project names are rejected.
	if _, _, err := s.Create("bad", []string{"a/b"}, []string{ScopeRead}, 0, ""); err == nil {
		t.Fatal("Create with path-like project name must fail")
	}
}

func TestStore_LegacySingleProjectFileStillWorks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	legacy := `{"tokens":[{"id":"deadbeef","name":"old","hash":"abc","created_at":"2026-01-01T00:00:00Z","project":"solo","scopes":["read"]}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	toks := s.List()
	if len(toks) != 1 {
		t.Fatalf("List = %d tokens", len(toks))
	}
	if got := toks[0].ProjectList(); len(got) != 1 || got[0] != "solo" {
		t.Fatalf("legacy file ProjectList = %v", got)
	}
	if toks[0].IsAdmin() {
		t.Fatal("legacy scoped token must not be admin")
	}
}
