package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplatesDir_ReflectsGlobal(t *testing.T) {
	s, _, dir := newTestServer(t)
	// Off → machine-owned .gosidian/templates.
	if got := s.templatesDir(); !strings.HasSuffix(got, filepath.Join(".gosidian", "templates")) {
		t.Errorf("off: templatesDir = %q", got)
	}
	// On → <global>/templates.
	s.SetGlobal(true, "global", "global-private")
	want := filepath.Join(dir, "global", "templates")
	if got := s.templatesDir(); got != want {
		t.Errorf("on: templatesDir = %q, want %q", got, want)
	}
}

func TestScaffold_FromGlobal(t *testing.T) {
	s, v, dir := newTestServer(t)
	s.SetGlobal(true, "global", "global-private")

	tdir := filepath.Join(dir, "global", "templates", "demo")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tdir, "_template.toml"), []byte("name = \"demo\"\ndescription = \"d\"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tdir, "README.md"), []byte("# {{PROJECT}}\n\nfrom global template\n"), 0o644)

	res, _ := s.handleProjectScaffold(context.Background(), call(map[string]any{"project": "newproj", "template": "demo"}))
	body := resultText(t, res)
	if !strings.Contains(body, "newproj/README.md") {
		t.Fatalf("expected scaffolded README in result: %s", body)
	}
	note, err := v.Load("newproj/README.md")
	if err != nil {
		t.Fatalf("scaffolded note not found: %v", err)
	}
	if !strings.Contains(string(note.Content), "# newproj") {
		t.Errorf("PROJECT not substituted: %s", note.Content)
	}
}
