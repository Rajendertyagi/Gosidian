package initprompt

import (
	"strings"
	"testing"
)

func TestRenderAgentAnchor_Defaults(t *testing.T) {
	got, err := RenderAgentAnchor(ProfileClaude, AnchorInput{
		CanonicalPath: "plancia/agents/frontend-engineer.md",
		Slug:          "frontend-engineer",
		Description:   "Vue 3 lib",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != ".claude/agents/frontend-engineer.md" {
		t.Errorf("path = %q", got.Path)
	}
	if got.AnchorVersion != AnchorVersion {
		t.Errorf("anchor version = %d", got.AnchorVersion)
	}
	if got.MetaVersion == "" {
		t.Error("empty meta version")
	}
	for _, want := range []string{
		"name: frontend-engineer",
		"mcp__gosidian__memory_get",
		`memory_get({ path: "plancia/agents/frontend-engineer.md" })`,
		"gosidian:anchor",
	} {
		if !strings.Contains(got.Content, want) {
			t.Errorf("content missing %q\n---\n%s", want, got.Content)
		}
	}
	if strings.Contains(got.Content, "model:") {
		t.Error("unexpected model line when model empty")
	}
}

func TestRenderAgentAnchor_HarnessOverrides(t *testing.T) {
	got, err := RenderAgentAnchor(ProfileClaude, AnchorInput{
		CanonicalPath: "p/agents/x.md",
		Slug:          "x",
		Name:          "custom-name",
		Description:   "desc",
		Tools:         []string{"Read", "mcp__gosidian__memory_get"},
		Model:         "opus",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: custom-name", "model: opus", "tools: Read, mcp__gosidian__memory_get"} {
		if !strings.Contains(got.Content, want) {
			t.Errorf("content missing %q\n---\n%s", want, got.Content)
		}
	}
}

func TestRenderAgentAnchor_MetaVersionStableAndSensitive(t *testing.T) {
	base := AnchorInput{CanonicalPath: "p/agents/x.md", Slug: "x", Description: "d"}
	a, _ := RenderAgentAnchor(ProfileClaude, base)
	b, _ := RenderAgentAnchor(ProfileClaude, base)
	if a.MetaVersion != b.MetaVersion {
		t.Error("meta version not stable for identical input")
	}
	changed := base
	changed.Description = "different"
	c, _ := RenderAgentAnchor(ProfileClaude, changed)
	if a.MetaVersion == c.MetaVersion {
		t.Error("meta version did not change when description changed")
	}
}

func TestSupportsAnchorsAndUnsupportedProfile(t *testing.T) {
	if !SupportsAnchors(ProfileClaude) {
		t.Error("claude should support anchors")
	}
	if SupportsAnchors(ProfileGeneric) {
		t.Error("generic should NOT support anchors")
	}
	if _, err := RenderAgentAnchor(ProfileGeneric, AnchorInput{CanonicalPath: "p/agents/x.md", Slug: "x"}); err == nil {
		t.Error("expected error rendering anchor for unsupported profile")
	}
}
