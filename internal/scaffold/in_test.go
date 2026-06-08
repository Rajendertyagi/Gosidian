package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplatesIn_CustomDir(t *testing.T) {
	dir := t.TempDir()
	demo := filepath.Join(dir, "demo")
	if err := os.MkdirAll(demo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, MetaFilename), []byte("name = \"demo\"\ndescription = \"a demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demo, "README.md"), []byte("# {{PROJECT}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ListTemplatesIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "demo" {
		t.Fatalf("ListTemplatesIn = %+v", list)
	}

	tmpl, err := LoadTemplateIn(dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.FileCount != 1 || len(tmpl.Files) != 1 || tmpl.Files[0] != "README.md" {
		t.Errorf("LoadTemplateIn files = %+v", tmpl.Files)
	}
	data, err := tmpl.ReadFile("README.md")
	if err != nil || string(data) != "# {{PROJECT}}\n" {
		t.Errorf("ReadFile = %q err=%v", data, err)
	}

	// Missing dir → empty, no error.
	empty, err := ListTemplatesIn(filepath.Join(dir, "nope"))
	if err != nil || len(empty) != 0 {
		t.Errorf("missing dir should be empty: %v %v", empty, err)
	}
}
