package mcp

import "testing"

func TestAnchorSlug(t *testing.T) {
	cases := map[string]string{
		"plancia/agents/frontend-engineer.md": "frontend-engineer",
		"x.md":                                "x",
		"a/b/c.md":                            "c",
	}
	for in, want := range cases {
		if got := anchorSlug(in); got != want {
			t.Errorf("anchorSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnchorInputFromNote(t *testing.T) {
	content := []byte("---\n" +
		"title: X\n" +
		"type: agent\n" +
		"description: routing hint\n" +
		"harness:\n" +
		"  tools: [Read, mcp__gosidian__memory_get]\n" +
		"  model: sonnet\n" +
		"tags: [type:agent]\n" +
		"---\n\nbody\n")
	in := anchorInputFromNote("p/agents/x.md", content)
	if in.Slug != "x" || in.Name != "x" {
		t.Errorf("slug/name = %q/%q", in.Slug, in.Name)
	}
	if in.Description != "routing hint" {
		t.Errorf("description = %q", in.Description)
	}
	if in.Model != "sonnet" {
		t.Errorf("model = %q", in.Model)
	}
	if len(in.Tools) != 2 {
		t.Errorf("tools = %v", in.Tools)
	}
}
