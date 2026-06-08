package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
	"github.com/gosidian/gosidian/internal/projects"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func skillCount(t *testing.T, res *mcplib.CallToolResult) int {
	t.Helper()
	var p struct {
		Skills []json.RawMessage `json:"available_skills"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return len(p.Skills)
}

func mkSkill(t *testing.T, s *Server, ctx context.Context, path, title, proj string) {
	t.Helper()
	_, _ = s.handleCreate(ctx, call(map[string]any{
		"path":    path,
		"content": "---\ntitle: " + title + "\ntags: [" + proj + ", type:skill]\n---\n\n# " + title + "\n",
	}))
}

func TestBootstrap_GlobalMerge(t *testing.T) {
	s, _, _ := newTestServer(t)
	pstore, err := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.SetProjects(pstore)
	s.SetGlobal(true, "global", "global-private")
	ctx := context.Background() // admin token (auth disabled)

	mkSkill(t, s, ctx, "proj/skills/a.md", "Local Skill", "proj")
	mkSkill(t, s, ctx, "global/skills/shared.md", "Shared Skill", "global")
	mkSkill(t, s, ctx, "global/skills/dup.md", "Local Skill", "global") // collision → local wins
	mkSkill(t, s, ctx, "global-private/skills/secret.md", "Secret Skill", "global-private")
	mkSkill(t, s, ctx, "global/templates/web-app/skills/tmpl.md", "Template Skill", "global") // under templates/ → excluded

	if err := pstore.Set("proj", projects.Flags{UseGlobals: true}); err != nil {
		t.Fatal(err)
	}

	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "proj"}))
	var p struct {
		Skills []struct {
			Title  string `json:"title"`
			Source string `json:"source"`
		} `json:"available_skills"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bySource := map[string]string{}
	localCount := 0
	for _, sk := range p.Skills {
		bySource[sk.Title] = sk.Source
		if sk.Title == "Local Skill" {
			localCount++
		}
	}
	if bySource["Local Skill"] != "local" {
		t.Errorf("Local Skill source = %q, want local", bySource["Local Skill"])
	}
	if bySource["Shared Skill"] != "global" {
		t.Errorf("Shared Skill source = %q, want global", bySource["Shared Skill"])
	}
	if bySource["Secret Skill"] != "global-private" {
		t.Errorf("Secret Skill source = %q, want global-private", bySource["Secret Skill"])
	}
	if localCount != 1 {
		t.Errorf("collision: expected exactly 1 'Local Skill' (local wins), got %d", localCount)
	}
	if _, present := bySource["Template Skill"]; present {
		t.Error("template-definition skill under global/templates/ must not be merged")
	}
}

func TestBootstrap_GlobalDisabledOrOptOut(t *testing.T) {
	s, _, _ := newTestServer(t)
	pstore, _ := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	s.SetProjects(pstore)
	ctx := context.Background()
	mkSkill(t, s, ctx, "global/skills/shared.md", "Shared", "global")

	// Feature off → no merge even if the project opted in.
	s.SetGlobal(false, "global", "global-private")
	_ = pstore.Set("proj", projects.Flags{UseGlobals: true})
	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "proj"}))
	if c := skillCount(t, res); c != 0 {
		t.Errorf("disabled global: expected 0 skills, got %d", c)
	}

	// Enabled but project opted out → no merge.
	s.SetGlobal(true, "global", "global-private")
	_ = pstore.Set("proj", projects.Flags{})
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj"}))
	if c := skillCount(t, res); c != 0 {
		t.Errorf("opted-out project: expected 0 skills, got %d", c)
	}
}

func TestBootstrap_GlobalPrivateGatedByScope(t *testing.T) {
	s, _, _ := newTestServer(t)
	pstore, _ := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	s.SetProjects(pstore)
	s.SetGlobal(true, "global", "global-private")
	admin := context.Background()
	mkSkill(t, s, admin, "global/skills/pub.md", "Pub", "global")
	mkSkill(t, s, admin, "global-private/skills/priv.md", "Priv", "global-private")
	_ = pstore.Set("proj", projects.Flags{UseGlobals: true})

	// Scoped token (Project=proj) sees global-public but NOT global-private.
	scoped := &auth.Token{ID: "x", Name: "scoped", Project: "proj", Scopes: []string{auth.ScopeRead}}
	ctx := context.WithValue(context.Background(), tokenCtxKey, scoped)
	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "proj"}))
	body := resultText(t, res)
	if !strings.Contains(body, `"title":"Pub"`) {
		t.Errorf("scoped token should see global-public Pub: %s", body)
	}
	if strings.Contains(body, `"title":"Priv"`) {
		t.Errorf("scoped token must NOT see global-private Priv: %s", body)
	}
}
