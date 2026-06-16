package v1

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/authz"
	"github.com/gosidian/gosidian/internal/index"
	"github.com/gosidian/gosidian/internal/vault"
)

const inlineTestPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func TestInlineImages(t *testing.T) {
	dir := t.TempDir()
	vroot := filepath.Join(dir, "vault")
	if err := os.MkdirAll(filepath.Join(vroot, "attachments"), 0o755); err != nil {
		t.Fatal(err)
	}
	png, _ := base64.StdEncoding.DecodeString(inlineTestPNG)
	if err := os.WriteFile(filepath.Join(vroot, "attachments", "x.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := index.Open(filepath.Join(dir, "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { idx.Close() })

	// Construct the Router directly (NewRouter would register routes and panic
	// on the nil Auth dep); inlineImages only needs Vault + Index.
	r := &Router{deps: &Deps{Vault: vault.New(vroot), Index: idx}}
	p := authz.Principal{}

	md := "![[x.png]]\n\nlink [[note]] and ![alt](/vault-files/attachments/x.png)\n"
	out := r.inlineImages(md, "markdown", p)
	if n := strings.Count(out, "data:image/png;base64,"); n != 2 {
		t.Errorf("markdown: want 2 inlined images, got %d:\n%s", n, out)
	}
	if strings.Contains(out, "![[x.png]]") || strings.Contains(out, "/vault-files/") {
		t.Errorf("markdown image references not replaced:\n%s", out)
	}
	if !strings.Contains(out, "[[note]]") {
		t.Errorf("non-image wikilink must stay intact:\n%s", out)
	}

	html := `<p><img src="/vault-files/attachments/x.png" alt="a"></p>`
	outH := r.inlineImages(html, "html", p)
	if !strings.Contains(outH, "data:image/png;base64,") || strings.Contains(outH, "/vault-files/") {
		t.Errorf("html image not inlined:\n%s", outH)
	}

	// Unresolvable reference is left untouched (no broken data: URI).
	if got := r.inlineImages("![[nope.png]]", "markdown", p); got != "![[nope.png]]" {
		t.Errorf("unresolvable embed should be untouched, got %q", got)
	}
}
