package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGlobalCheck_DetectsReferences(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetGlobal(true, "global", "global-private")
	ctx := context.Background()

	// A global-private skill, referenced by two different projects.
	mkSkill(t, s, ctx, "global-private/skills/secret.md", "Secret", "global-private")
	_, _ = s.handleCreate(ctx, call(map[string]any{
		"path": "x/note.md", "content": "---\ntitle: n\ntags: [x]\n---\n\nuses [[global-private/skills/secret]]\n",
	}))
	_, _ = s.handleCreate(ctx, call(map[string]any{
		"path": "y/note.md", "content": "---\ntitle: n\ntags: [y]\n---\n\nalso [[global-private/skills/secret]]\n",
	}))

	res, _ := s.handleGlobalCheck(ctx, call(map[string]any{"project": "x"}))
	var r struct {
		Referenced []struct {
			Path       string   `json:"path"`
			AlsoUsedBy []string `json:"also_used_by"`
		} `json:"referenced"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Referenced) != 1 {
		t.Fatalf("expected 1 referenced note, got %+v", r.Referenced)
	}
	if r.Referenced[0].Path != "global-private/skills/secret.md" {
		t.Errorf("referenced path = %q", r.Referenced[0].Path)
	}
	foundY := false
	for _, p := range r.Referenced[0].AlsoUsedBy {
		if p == "x" {
			t.Error("the checked project must be excluded from also_used_by")
		}
		if p == "y" {
			foundY = true
		}
	}
	if !foundY {
		t.Errorf("expected y in also_used_by, got %+v", r.Referenced[0].AlsoUsedBy)
	}
}

func TestGlobalCheck_OffReturnsEmpty(t *testing.T) {
	s, _, _ := newTestServer(t)
	// Feature off (default).
	res, _ := s.handleGlobalCheck(context.Background(), call(map[string]any{"project": "x"}))
	if !strings.Contains(resultText(t, res), `"referenced":[]`) {
		t.Errorf("off: expected empty referenced: %s", resultText(t, res))
	}
}
