package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/initprompt"
)

// seedBootstrapVault populates the test server with a small but realistic set
// of notes under the "gosidian" project so handleBootstrap has something to
// aggregate. Returns the project name used.
func seedBootstrapVault(t *testing.T, s *Server) string {
	t.Helper()
	ctx := context.Background()
	entries := []struct{ path, content string }{
		{"gosidian/hot.md", "---\ntitle: hot\ntype: index\n---\n\n# Hot\nwelcome"},
		{"gosidian/README.md", "---\ntitle: readme\ntype: index\n---\n\n# Readme"},
		{"gosidian/plans/active.md", "---\ntitle: active\ntype: plan\nstatus: in-progress\ndescription: a live task\nupdated: 2026-04-22\ntags: [type:plan, status:in-progress]\n---\n\n# Active"},
		{"gosidian/plans/old.md", "---\ntitle: old\ntype: plan\nstatus: done\ntags: [type:plan, status:done]\n---\n\n# Old"},
		{"gosidian/skills/build.md", "---\ntitle: build\ntype: skill\ndescription: how to build\ntags: [type:skill]\n---\n\n# build\n\n## Trigger phrase\n\n- rebuild the thing"},
		{"gosidian/agents/backend.md", "---\ntitle: backend-agent\ntype: agent\ntags: [type:agent]\n---\n\n# backend"},
		{"other/foreign.md", "---\ntitle: foreign\ntags: [type:plan, status:in-progress]\n---\n\nnot mine"},
	}
	for _, e := range entries {
		res, err := s.handleCreate(ctx, call(map[string]any{"path": e.path, "content": e.content}))
		if err != nil || (res != nil && res.IsError) {
			t.Fatalf("seed %q: err=%v res=%+v", e.path, err, res)
		}
	}
	return "gosidian"
}

func TestMCP_Bootstrap_HappyPath(t *testing.T) {
	s, _, _ := newTestServer(t)
	proj := seedBootstrapVault(t, s)

	res, err := s.handleBootstrap(context.Background(), call(map[string]any{"project": proj}))
	if err != nil {
		t.Fatal(err)
	}
	body := resultText(t, res)

	var p struct {
		Project           string                 `json:"project"`
		HotMD             bootstrapFile          `json:"hot_md"`
		Readme            bootstrapFile          `json:"readme"`
		AgentMD           bootstrapFile          `json:"agent_md"`
		ActivePlans       []noteRef              `json:"active_plans"`
		AvailableSkills   []noteRef              `json:"available_skills"`
		AvailableAgents   []noteRef              `json:"available_agents"`
		RecentNotes       []recentNoteResponse   `json:"recent_notes"`
		Stats             bootstrapStats         `json:"stats"`
		Missing           []string               `json:"missing"`
		DirectivesVersion int                    `json:"directives_version"`
		DirectivesBlock   string                 `json:"directives_block"`
		StubVersion       int                    `json:"stub_version"`
		Capabilities      *bootstrapCapabilities `json:"capabilities"`
	}
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("parse: %v body=%s", err, body)
	}

	if p.Project != proj {
		t.Errorf("project = %q, want %q", p.Project, proj)
	}
	if !p.HotMD.Present || !p.Readme.Present {
		t.Errorf("hot/readme should be present: %+v %+v", p.HotMD, p.Readme)
	}
	if p.AgentMD.Present {
		t.Errorf("agent instruction file not seeded, should be absent")
	}
	if p.HotMD.ETag == "" {
		t.Errorf("hot_md etag should be non-empty")
	}

	// Active plans: only gosidian/plans/active.md (gosidian/plans/old.md is done;
	// other/foreign.md is outside project).
	if len(p.ActivePlans) != 1 || p.ActivePlans[0].Path != "gosidian/plans/active.md" {
		t.Errorf("active_plans = %+v, want only gosidian/plans/active.md", p.ActivePlans)
	}
	if len(p.AvailableSkills) != 1 || p.AvailableSkills[0].Path != "gosidian/skills/build.md" {
		t.Errorf("available_skills = %+v", p.AvailableSkills)
	}
	if len(p.AvailableAgents) != 1 {
		t.Errorf("available_agents = %+v", p.AvailableAgents)
	}
	if len(p.RecentNotes) == 0 || len(p.RecentNotes) > 5 {
		t.Errorf("recent_notes len = %d", len(p.RecentNotes))
	}
	if p.Stats.NotesCount < 6 {
		t.Errorf("expected at least 6 project notes, got %d", p.Stats.NotesCount)
	}
	if len(p.Stats.TopTags) == 0 {
		t.Errorf("top_tags should not be empty")
	}
	// IMP-050: the instruction file is expected to live in the agent's working
	// dir (stub model), so its vault-absence is flagged expected_external rather
	// than reported in `missing`. hot.md + README.md are seeded → missing empty.
	if !p.AgentMD.ExpectedExternal {
		t.Errorf("agent_md.expected_external should be true when no vault instruction file: %+v", p.AgentMD)
	}
	for _, m := range p.Missing {
		if m == "AGENTS.md" {
			t.Errorf("AGENTS.md must not appear in missing (IMP-050): %v", p.Missing)
		}
	}
	if len(p.Missing) != 0 {
		t.Errorf("missing should be empty (hot.md + README.md seeded), got %v", p.Missing)
	}

	if p.DirectivesVersion != initprompt.DirectivesVersion {
		t.Errorf("directives_version = %d, want %d", p.DirectivesVersion, initprompt.DirectivesVersion)
	}
	if p.StubVersion != initprompt.StubVersion {
		t.Errorf("stub_version = %d, want %d", p.StubVersion, initprompt.StubVersion)
	}
	// directives_block must be served and carry its own version marker + the
	// project name (ADR-010: directives delivered via bootstrap, not embedded).
	if p.DirectivesBlock == "" {
		t.Error("directives_block should be served, got empty")
	}
	if !strings.Contains(p.DirectivesBlock, "gosidian:directives") {
		t.Error("directives_block missing its version marker")
	}
	if !strings.Contains(p.DirectivesBlock, proj) {
		t.Error("directives_block should be rendered for the project")
	}

	// capabilities: always present, mirrors live config (flags off in the test
	// server) and carries the attachment surface incl. the /upload hint.
	if p.Capabilities == nil {
		t.Fatal("capabilities block should always be present")
	}
	if p.Capabilities.HTMLNotes || p.Capabilities.MediaNotes {
		t.Errorf("test server has html/media notes off, got %+v", p.Capabilities)
	}
	if p.Capabilities.Attachments.MaxMiB != 10 {
		t.Errorf("attachments.max_mib = %d, want 10", p.Capabilities.Attachments.MaxMiB)
	}
	if len(p.Capabilities.Attachments.Extensions) == 0 {
		t.Error("attachments.extensions should not be empty")
	}
	if !strings.Contains(p.Capabilities.Attachments.UploadEndpointHint, "/upload") {
		t.Error("attachments.upload_endpoint_hint should mention /upload")
	}
}

func TestMCP_Bootstrap_EmptyProject(t *testing.T) {
	s, _, _ := newTestServer(t)
	// No seed — project "void" has zero notes.

	res, err := s.handleBootstrap(context.Background(), call(map[string]any{"project": "void"}))
	if err != nil {
		t.Fatal(err)
	}
	body := resultText(t, res)

	var p struct {
		Missing []string       `json:"missing"`
		Stats   bootstrapStats `json:"stats"`
	}
	_ = json.Unmarshal([]byte(body), &p)
	if p.Stats.NotesCount != 0 {
		t.Errorf("expected 0 notes, got %d", p.Stats.NotesCount)
	}
	// IMP-050: AGENTS.md no longer counts as missing vault scaffold — only
	// hot.md + README.md remain.
	if len(p.Missing) != 2 {
		t.Errorf("expected 2 missing convention files (hot.md, README.md), got %v", p.Missing)
	}
}

func TestMCP_Bootstrap_RequiresProject(t *testing.T) {
	s, _, _ := newTestServer(t)
	res, _ := s.handleBootstrap(context.Background(), call(map[string]any{}))
	msg := expectError(t, res)
	if msg == "" || msg == "unauthorized" {
		t.Errorf("expected project-required error, got %q", msg)
	}
}
