package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPromoteAgent(t *testing.T) {
	s, _, _ := newTestServer(t)
	ctx := context.Background()

	foreign := "---\n" +
		"name: rc-database\n" +
		"description: DB ops for rc\n" +
		"tools: Read, Bash, mcp__gosidian__memory_get\n" +
		"---\n\n" +
		"You are the rc database engineer. Be careful with migrations.\n"

	res, _ := s.handlePromoteAgent(ctx, call(map[string]any{
		"project": "rc", "slug": "rc-database", "content": foreign, "profile": "claude",
	}))
	var p struct {
		Path   string `json:"path"`
		Anchor *struct {
			Path      string `json:"path"`
			Content   string `json:"content"`
			Canonical string `json:"canonical"`
		} `json:"anchor"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Path != "rc/agents/rc-database.md" {
		t.Errorf("path = %q", p.Path)
	}

	note, err := s.vault.Load("rc/agents/rc-database.md")
	if err != nil {
		t.Fatalf("canonical note not created: %v", err)
	}
	body := string(note.Content)
	for _, want := range []string{
		"type: agent",
		"tags: [rc, type:agent]",
		"description: DB ops for rc",
		"You are the rc database engineer",
		"harness:",
		"tools: [Read, Bash, mcp__gosidian__memory_get]",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("canonical note missing %q:\n%s", want, body)
		}
	}

	if p.Anchor == nil {
		t.Fatal("expected anchor in response")
	}
	if p.Anchor.Path != ".claude/agents/rc-database.md" {
		t.Errorf("anchor path = %q", p.Anchor.Path)
	}
	if p.Anchor.Canonical != "rc/agents/rc-database.md" {
		t.Errorf("anchor canonical = %q", p.Anchor.Canonical)
	}
	if !strings.Contains(p.Anchor.Content, `memory_get({ path: "rc/agents/rc-database.md" })`) {
		t.Errorf("anchor missing canonical pull:\n%s", p.Anchor.Content)
	}

	// Idempotency guard: promoting again over an existing canonical note errors.
	res2, _ := s.handlePromoteAgent(ctx, call(map[string]any{
		"project": "rc", "slug": "rc-database", "content": foreign,
	}))
	if !res2.IsError {
		t.Error("expected error promoting an already-existing canonical agent")
	}
}
