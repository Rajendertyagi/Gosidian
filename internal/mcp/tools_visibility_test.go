package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
	"github.com/gosidian/gosidian/internal/projects"
)

func TestMCP_HiddenProjectFiltersListProjects(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead})
	_, _ = s.vault.CreateProject("Visible")
	_, _ = s.vault.CreateProject("Hidden")

	pstore, err := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pstore.Set("Hidden", projects.Flags{HiddenFromMCP: true}); err != nil {
		t.Fatal(err)
	}
	s.SetProjects(pstore)

	res, _ := s.handleListProjects(ctx, call(map[string]any{}))
	body := resultText(t, res)
	if !strings.Contains(body, "Visible") {
		t.Errorf("missing Visible: %s", body)
	}
	if strings.Contains(body, "Hidden") {
		t.Errorf("Hidden leaked: %s", body)
	}
}

func TestMCP_HiddenProjectExplicit403OnListNotes(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead})
	_, _ = s.vault.CreateProject("Hidden")

	pstore, _ := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	_ = pstore.Set("Hidden", projects.Flags{HiddenFromMCP: true})
	s.SetProjects(pstore)

	res, _ := s.handleListNotes(ctx, call(map[string]any{"project": "Hidden"}))
	msg := expectError(t, res)
	if !strings.Contains(msg, "hidden") {
		t.Errorf("expected hidden error, got %q", msg)
	}
}

func TestMCP_HiddenProjectFiltersSearch(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	admin := context.WithValue(context.Background(), tokenCtxKey, auth.AdminToken())
	_, _ = s.vault.CreateProject("Visible")
	_, _ = s.vault.CreateProject("Hidden")
	_, _ = s.handleCreate(admin, call(map[string]any{"path": "Visible/v.md", "content": "find me"}))
	_, _ = s.handleCreate(admin, call(map[string]any{"path": "Hidden/h.md", "content": "find me"}))

	pstore, _ := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	_ = pstore.Set("Hidden", projects.Flags{HiddenFromMCP: true})
	s.SetProjects(pstore)

	// vault-wide search: hidden project silently filtered
	res, _ := s.handleSearch(ctx, call(map[string]any{"query": "find"}))
	body := resultText(t, res)
	if !strings.Contains(body, "Visible/v.md") {
		t.Errorf("missing Visible hit: %s", body)
	}
	if strings.Contains(body, "Hidden/h.md") {
		t.Errorf("Hidden hit leaked: %s", body)
	}

	// explicit projects=[Hidden] → 403
	res, _ = s.handleSearch(ctx, call(map[string]any{"query": "find", "projects": []any{"Hidden"}}))
	msg := expectError(t, res)
	if !strings.Contains(msg, "hidden") {
		t.Errorf("expected hidden error, got %q", msg)
	}
}
