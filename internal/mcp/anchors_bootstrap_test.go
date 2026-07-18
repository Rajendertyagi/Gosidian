package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/projects"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func mkAgent(t *testing.T, s *Server, ctx context.Context, path, title, proj string) {
	t.Helper()
	_, _ = s.handleCreate(ctx, call(map[string]any{
		"path":    path,
		"content": "---\ntitle: " + title + "\ndescription: routing hint\ntags: [" + proj + ", type:agent]\n---\n\n# " + title + "\n",
	}))
}

type anchorsBlock struct {
	Profile   string `json:"profile"`
	TargetDir string `json:"target_dir"`
	Items     []struct {
		Path        string `json:"path"`
		Content     string `json:"content"`
		MetaVersion string `json:"meta_version"`
		Canonical   string `json:"canonical"`
		Unchanged   bool   `json:"unchanged"`
	} `json:"items"`
	Reconcile string `json:"reconcile"`
}

func TestBootstrap_AgentAnchors_Gating(t *testing.T) {
	s, _, _ := newTestServer(t)
	pstore, err := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.SetProjects(pstore)
	ctx := context.Background()

	mkAgent(t, s, ctx, "proj/agents/frontend-engineer.md", "Frontend Engineer", "proj")

	parse := func(res *mcplib.CallToolResult) (anchorsBlock, bool) {
		t.Helper()
		var p struct {
			Anchors *anchorsBlock `json:"anchors"`
		}
		if err := json.Unmarshal([]byte(resultText(t, res)), &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if p.Anchors == nil {
			return anchorsBlock{}, false
		}
		return *p.Anchors, true
	}

	// 1. master OFF (default) + flag on → no anchors (backward-compat).
	if err := pstore.Set("proj", projects.Flags{UseAnchors: true}); err != nil {
		t.Fatal(err)
	}
	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude"}))
	if _, ok := parse(res); ok {
		t.Error("master off: expected no anchors")
	}

	s.SetAgentAnchors(true)

	// 2. master on + flag on + claude → anchors present and correct.
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude"}))
	ab, ok := parse(res)
	if !ok {
		t.Fatal("expected anchors payload")
	}
	if ab.Profile != "claude" || ab.TargetDir != ".claude/agents" {
		t.Errorf("profile/target_dir = %q/%q", ab.Profile, ab.TargetDir)
	}
	if len(ab.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(ab.Items))
	}
	it := ab.Items[0]
	if it.Path != ".claude/agents/frontend-engineer.md" {
		t.Errorf("item path = %q", it.Path)
	}
	if it.Canonical != "proj/agents/frontend-engineer.md" {
		t.Errorf("canonical = %q", it.Canonical)
	}
	if it.MetaVersion == "" {
		t.Error("empty meta version")
	}
	if !strings.Contains(it.Content, `memory_get({ path: "proj/agents/frontend-engineer.md" })`) {
		t.Errorf("content missing canonical pull:\n%s", it.Content)
	}
	if ab.Reconcile == "" {
		t.Error("empty reconcile directive")
	}

	// 3. master on + flag OFF → no anchors.
	if err := pstore.Set("proj", projects.Flags{}); err != nil {
		t.Fatal(err)
	}
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude"}))
	if _, ok := parse(res); ok {
		t.Error("flag off: expected no anchors")
	}

	// 4. master on + flag on + generic profile (no subagent support) → no anchors.
	if err := pstore.Set("proj", projects.Flags{UseAnchors: true}); err != nil {
		t.Fatal(err)
	}
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "generic"}))
	if _, ok := parse(res); ok {
		t.Error("generic profile: expected no anchors")
	}

	// 5. default profile (no param) resolves to claude → anchors present.
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj"}))
	if _, ok := parse(res); !ok {
		t.Error("default profile claude: expected anchors")
	}

	// 6. anchors enabled but empty desired set → block present (items:[] +
	// target_dir so stale local anchors can still be cleaned) WITHOUT the
	// ~500-char reconcile directive (token economy, plan 20260706).
	if err := pstore.Set("empty", projects.Flags{UseAnchors: true}); err != nil {
		t.Fatal(err)
	}
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "empty", "profile": "claude"}))
	ab, ok = parse(res)
	if !ok {
		t.Fatal("empty set: anchors block should still be present")
	}
	if len(ab.Items) != 0 || ab.Reconcile != "" || ab.TargetDir == "" {
		t.Errorf("empty set: want items=0, no reconcile, target_dir set; got %+v", ab)
	}

	// 7. harness.materialize:false → the note stays canonical (listed in
	// available_agents) but is excluded from the anchors desired set; a
	// previously-materialised anchor becomes an orphan the reconcile flow
	// removes (IMP-070).
	_, _ = s.handleCreate(ctx, call(map[string]any{
		"path":    "proj/agents/orchestrator.md",
		"content": "---\ntitle: Orchestrator\ndescription: coordination role, not spawnable\ntags: [proj, type:agent]\ntype: agent\nharness:\n  materialize: false\n---\n\n# Orchestrator\n",
	}))
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude"}))
	ab, ok = parse(res)
	if !ok {
		t.Fatal("materialize opt-out: anchors block should be present")
	}
	if len(ab.Items) != 1 || ab.Items[0].Canonical != "proj/agents/frontend-engineer.md" {
		t.Errorf("materialize opt-out: want only frontend-engineer in items, got %+v", ab.Items)
	}
}

func TestBootstrap_AgentAnchors_KnownMetasDelta(t *testing.T) {
	s, _, _ := newTestServer(t)
	pstore, err := projects.Open(filepath.Join(t.TempDir(), "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.SetProjects(pstore)
	s.SetAgentAnchors(true)
	ctx := context.Background()
	if err := pstore.Set("proj", projects.Flags{UseAnchors: true}); err != nil {
		t.Fatal(err)
	}
	mkAgent(t, s, ctx, "proj/agents/alpha.md", "Alpha", "proj")
	mkAgent(t, s, ctx, "proj/agents/beta.md", "Beta", "proj")

	parse := func(res *mcplib.CallToolResult) anchorsBlock {
		t.Helper()
		var p struct {
			Anchors *anchorsBlock `json:"anchors"`
		}
		if err := json.Unmarshal([]byte(resultText(t, res)), &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if p.Anchors == nil {
			t.Fatal("expected anchors payload")
		}
		return *p.Anchors
	}

	// First bootstrap: full content, collect the metas.
	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude"}))
	first := parse(res)
	if len(first.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(first.Items))
	}
	metas := map[string]any{}
	for _, it := range first.Items {
		if it.Content == "" || it.Unchanged {
			t.Errorf("first bootstrap: item %s should carry content", it.Canonical)
		}
		metas[it.Canonical] = it.MetaVersion
	}

	// Repeat with all metas known → every item unchanged, no content, but the
	// reconcile directive still ships (the "leave it" rule lives there).
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude", "known_anchor_metas": metas}))
	second := parse(res)
	if len(second.Items) != 2 {
		t.Fatalf("delta items = %d, want 2", len(second.Items))
	}
	for _, it := range second.Items {
		if !it.Unchanged || it.Content != "" {
			t.Errorf("known meta: item %s should be unchanged with no content, got unchanged=%v content=%dB",
				it.Canonical, it.Unchanged, len(it.Content))
		}
		if it.MetaVersion == "" {
			t.Errorf("known meta: item %s must still carry meta_version", it.Canonical)
		}
	}
	if second.Reconcile == "" {
		t.Error("delta bootstrap: reconcile directive should still be present")
	}

	// Stale meta for beta → beta full, alpha unchanged (mixed delta).
	metas["proj/agents/beta.md"] = "0000deadbeef"
	res, _ = s.handleBootstrap(ctx, call(map[string]any{"project": "proj", "profile": "claude", "known_anchor_metas": metas}))
	third := parse(res)
	for _, it := range third.Items {
		wantUnchanged := it.Canonical == "proj/agents/alpha.md"
		if it.Unchanged != wantUnchanged || (it.Content == "") != wantUnchanged {
			t.Errorf("mixed delta: item %s unchanged=%v content=%dB", it.Canonical, it.Unchanged, len(it.Content))
		}
	}
}
