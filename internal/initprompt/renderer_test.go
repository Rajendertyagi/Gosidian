package initprompt

import (
	"strings"
	"testing"
	"time"
)

func TestRender_AllProfilesAllModes(t *testing.T) {
	cases := []struct {
		name    string
		profile Profile
		mode    Mode
	}{
		{"claude augment", ProfileClaude, ModeAugment},
		{"claude fromscratch", ProfileClaude, ModeFromScratch},
		{"cursor augment", ProfileCursor, ModeAugment},
		{"cursor fromscratch", ProfileCursor, ModeFromScratch},
		{"codex augment", ProfileCodex, ModeAugment},
		{"codex fromscratch", ProfileCodex, ModeFromScratch},
		{"aider augment", ProfileAider, ModeAugment},
		{"aider fromscratch", ProfileAider, ModeFromScratch},
		{"generic augment", ProfileGeneric, ModeAugment},
		{"generic fromscratch", ProfileGeneric, ModeFromScratch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Render("myproj", tc.profile, tc.mode, Hints{}, false)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}
			if res.Mode != tc.mode {
				t.Errorf("mode = %q, want %q", res.Mode, tc.mode)
			}
			if !res.NeedsScaffold {
				t.Error("NeedsScaffold should be true when projectExists=false")
			}
			if len(res.Prompt) < 500 {
				t.Errorf("Prompt too short: %d chars (want ≥ 500)", len(res.Prompt))
			}
			if len(res.GosidianBlock) < 1500 {
				t.Errorf("GosidianBlock too short: %d chars (want ≥ 1500)", len(res.GosidianBlock))
			}
			if len(res.SuggestedQuestions) == 0 {
				t.Error("SuggestedQuestions should not be empty")
			}
			// PROJECT and TODAY must always be resolved server-side.
			if strings.Contains(res.GosidianBlock, "{{PROJECT}}") {
				t.Error("GosidianBlock contains unresolved {{PROJECT}}")
			}
			if strings.Contains(res.GosidianBlock, "{{TODAY}}") {
				t.Error("GosidianBlock contains unresolved {{TODAY}}")
			}
			if !strings.Contains(res.GosidianBlock, "myproj") {
				t.Error("GosidianBlock should mention the project name")
			}
			today := time.Now().UTC().Format("2006-01-02")
			if !strings.Contains(res.GosidianBlock, today) {
				t.Errorf("GosidianBlock should mention today (%s)", today)
			}
		})
	}
}

func TestRender_AugmentPromptAnchors(t *testing.T) {
	res, err := Render("p", ProfileClaude, ModeAugment, Hints{}, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	wantAnchors := []string{
		"Determina filename",
		"Merge",
		"preservando tutto il contenuto esistente",
	}
	for _, a := range wantAnchors {
		if !strings.Contains(res.Prompt, a) {
			t.Errorf("augment prompt missing anchor %q", a)
		}
	}
	if strings.Contains(res.Prompt, "Scan cwd") {
		t.Error("augment prompt should not contain 'Scan cwd' anchor (that's from-scratch territory)")
	}
}

func TestRender_FromScratchPromptAnchors(t *testing.T) {
	res, err := Render("p", ProfileClaude, ModeFromScratch, Hints{}, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	wantAnchors := []string{
		"Scan cwd",
		"Compila",
	}
	for _, a := range wantAnchors {
		if !strings.Contains(res.Prompt, a) {
			t.Errorf("from-scratch prompt missing anchor %q", a)
		}
	}
}

func TestRender_ProjectExistsSkipsScaffold(t *testing.T) {
	res, err := Render("p", ProfileGeneric, ModeAugment, Hints{}, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if res.NeedsScaffold {
		t.Error("NeedsScaffold should be false when projectExists=true")
	}
}

func TestRender_HintsResolveBlockPlaceholders(t *testing.T) {
	hints := Hints{
		Language:     "italiano",
		CodeLanguage: "inglese",
		ProjectType:  "applicazione web",
		Stack:        "Next.js + Prisma",
		HotFiles:     "src/app.tsx, prisma/schema.prisma",
	}
	res, err := Render("p", ProfileClaude, ModeAugment, hints, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	for _, ph := range []string{
		"{{LANGUAGE}}", "{{CODE_LANGUAGE}}", "{{PROJECT_TYPE}}",
		"{{STACK}}", "{{HOT_FILES}}",
	} {
		if strings.Contains(res.GosidianBlock, ph) {
			t.Errorf("block contains unresolved %s despite hint", ph)
		}
	}
	for _, want := range []string{"italiano", "inglese", "applicazione web", "Next.js + Prisma"} {
		if !strings.Contains(res.GosidianBlock, want) {
			t.Errorf("block missing hint value %q", want)
		}
	}
}

func TestRender_NoHintsKeepsPlaceholdersIntact(t *testing.T) {
	res, err := Render("p", ProfileClaude, ModeAugment, Hints{}, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	for _, ph := range []string{
		"{{LANGUAGE}}", "{{CODE_LANGUAGE}}", "{{PROJECT_TYPE}}",
		"{{STACK}}", "{{HOT_FILES}}",
	} {
		if !strings.Contains(res.GosidianBlock, ph) {
			t.Errorf("block missing placeholder %s (should remain unresolved without hint)", ph)
		}
	}
}

func TestRender_RejectsUnknownProfile(t *testing.T) {
	_, err := Render("p", Profile("bogus"), ModeAugment, Hints{}, true)
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if !strings.Contains(err.Error(), "unknown agent profile") {
		t.Errorf("error should mention unknown profile, got: %v", err)
	}
}

func TestRender_RejectsUnknownMode(t *testing.T) {
	_, err := Render("p", ProfileClaude, Mode("weird"), Hints{}, true)
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestRender_RejectsEmptyProject(t *testing.T) {
	_, err := Render("  ", ProfileClaude, ModeAugment, Hints{}, true)
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestRender_AgentNameFallsBackToProfileDisplayName(t *testing.T) {
	res, err := Render("p", ProfileClaude, ModeAugment, Hints{}, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(res.GosidianBlock, "Claude Code") {
		t.Error("block should contain default AGENT_NAME = 'Claude Code' for claude profile")
	}
}

func TestRender_AgentNameHintOverridesDefault(t *testing.T) {
	hints := Hints{AgentName: "Custom IDE"}
	res, err := Render("p", ProfileGeneric, ModeAugment, hints, true)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(res.GosidianBlock, "Custom IDE") {
		t.Error("block should honour AgentName hint")
	}
}

func TestProfiles_ReturnsAllRegistered(t *testing.T) {
	got := Profiles()
	if len(got) != 5 {
		t.Errorf("Profiles() returned %d entries, want 5", len(got))
	}
	for _, p := range got {
		if !IsKnownProfile(p) {
			t.Errorf("Profiles() returned unknown profile %q", p)
		}
	}
}
