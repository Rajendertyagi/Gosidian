package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
)

// A multi-project token (the orchestrator-bus case) writes in all of its
// projects and nowhere else, and must name a project explicitly where a
// single-project token would be defaulted.
func TestMCP_MultiProjectTokenScoping(t *testing.T) {
	s, _, _ := newTestServer(t)
	admin := context.Background()

	// Seed a foreign project so scoped listings have something to hide.
	res, _ := s.handleCreate(admin, call(map[string]any{"path": "pc/secret.md", "content": "# C"}))
	resultText(t, res)

	orch := &auth.Token{
		ID:       "orch0001",
		Name:     "orchestrator",
		Projects: []string{"pa", "pb"},
		Scopes:   []string{auth.ScopeRead, auth.ScopeWrite},
	}
	ctx := ctxWithToken(orch)

	// Writes in both scoped projects succeed; outside is rejected.
	res, _ = s.handleCreate(ctx, call(map[string]any{"path": "pa/a.md", "content": "# A"}))
	resultText(t, res)
	res, _ = s.handleCreate(ctx, call(map[string]any{"path": "pb/b.md", "content": "# B"}))
	resultText(t, res)
	res, _ = s.handleCreate(ctx, call(map[string]any{"path": "pc/nope.md", "content": "# X"}))
	if msg := expectError(t, res); !strings.Contains(msg, "outside the token's project scope") {
		t.Fatalf("out-of-scope create error = %q", msg)
	}

	// Optional-project listings must not silently widen: the project
	// argument is required when the scope spans several projects.
	res, _ = s.handleListNotes(ctx, call(map[string]any{}))
	if msg := expectError(t, res); !strings.Contains(msg, "pass the project argument") {
		t.Fatalf("multi-project listing without project error = %q", msg)
	}
	res, _ = s.handleListNotes(ctx, call(map[string]any{"project": "pb"}))
	if txt := resultText(t, res); !strings.Contains(txt, "pb/b.md") || strings.Contains(txt, "pa/a.md") {
		t.Fatalf("scoped listing of pb = %s", txt)
	}
	res, _ = s.handleListNotes(ctx, call(map[string]any{"project": "pc"}))
	if msg := expectError(t, res); !strings.Contains(msg, "outside the token's scope") {
		t.Fatalf("out-of-scope listing error = %q", msg)
	}

	// Project catalogue shows only the token's projects.
	res, _ = s.handleListProjects(ctx, call(map[string]any{}))
	if txt := resultText(t, res); strings.Contains(txt, "pc") || !strings.Contains(txt, "pa") || !strings.Contains(txt, "pb") {
		t.Fatalf("scoped project list = %s", txt)
	}

	// resolveProject-based tools accept any project in scope, reject others.
	res, _ = s.handlePendingHandoffs(ctx, call(map[string]any{"project": "pa"}))
	resultText(t, res)
	res, _ = s.handlePendingHandoffs(ctx, call(map[string]any{"project": "pc"}))
	expectError(t, res)
	res, _ = s.handlePendingHandoffs(ctx, call(map[string]any{}))
	if msg := expectError(t, res); !strings.Contains(msg, "pass the project argument") {
		t.Fatalf("resolveProject without project error = %q", msg)
	}

	// Multi-project ≠ admin: project lifecycle stays off-limits.
	res, _ = s.handleCreateProject(ctx, call(map[string]any{"name": "pd"}))
	expectError(t, res)

	// The bus flow across projects: orchestrator hands off in pa and pb with
	// one token, claims land under its identity.
	res, _ = s.handleCreateHandoff(ctx, call(map[string]any{
		"project": "pa", "from_agent": "orchestrator", "to_agent": "agent-a", "summary": "do pa work",
	}))
	var created struct {
		Path      string `json:"path"`
		CreatedBy string `json:"created_by"`
	}
	decodeResult(t, res, &created)
	if created.CreatedBy != "orchestrator@orch0001" {
		t.Fatalf("created_by = %q", created.CreatedBy)
	}
	res, _ = s.handleClaimHandoff(ctx, call(map[string]any{"path": created.Path}))
	resultText(t, res)
}
