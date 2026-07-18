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

func TestRenderAgentAnchor_ToolsAll(t *testing.T) {
	got, err := RenderAgentAnchor(ProfileClaude, AnchorInput{
		CanonicalPath: "p/agents/x.md",
		Slug:          "x",
		Description:   "d",
		ToolsAll:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Content, "tools:") {
		t.Errorf("tools line should be omitted for ToolsAll:\n%s", got.Content)
	}
	// The frontmatter must stay well-formed: description directly followed
	// by the closing fence when both tools and model are absent.
	if !strings.Contains(got.Content, "description: \"d\"\n---\n") {
		t.Errorf("frontmatter malformed:\n%s", got.Content)
	}

	deflt, _ := RenderAgentAnchor(ProfileClaude, AnchorInput{CanonicalPath: "p/agents/x.md", Slug: "x", Description: "d"})
	enum, _ := RenderAgentAnchor(ProfileClaude, AnchorInput{CanonicalPath: "p/agents/x.md", Slug: "x", Description: "d", Tools: []string{"Read"}})
	if got.MetaVersion == deflt.MetaVersion || got.MetaVersion == enum.MetaVersion || deflt.MetaVersion == enum.MetaVersion {
		t.Errorf("meta versions must differ across default/enumerated/all: %q %q %q",
			deflt.MetaVersion, enum.MetaVersion, got.MetaVersion)
	}
}

// TestRenderAgentAnchor_MetaVersionGolden pins the meta_version of a
// default-tools anchor. A change here means EVERY existing anchor file in
// every project gets rewritten on its first post-upgrade bootstrap (the
// reconcile flow rewrites on meta mismatch) — change the pinned value only
// with that consequence in mind.
func TestRenderAgentAnchor_MetaVersionGolden(t *testing.T) {
	got, err := RenderAgentAnchor(ProfileClaude, AnchorInput{CanonicalPath: "p/agents/x.md", Slug: "x", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if got.MetaVersion != "f115e5118bbf" {
		t.Errorf("meta_version = %q, want f115e5118bbf (mass-rewrite guard)", got.MetaVersion)
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
