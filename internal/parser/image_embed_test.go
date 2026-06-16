package parser

import (
	"strings"
	"testing"
)

// bothResolver implements Resolver + ImageResolver for image-embed tests.
type bothResolver struct{ img map[string]string }

func (b bothResolver) Resolve(string) string        { return "" }
func (b bothResolver) ResolveImage(t string) string { return b.img[t] }

func TestRender_ImageEmbed(t *testing.T) {
	r := NewRenderer()
	res := bothResolver{img: map[string]string{"shot.webp": "/vault-files/attachments/shot.webp"}}
	out, err := r.Render([]byte("intro\n\n![[shot.webp]]\n"), res)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `src="/vault-files/attachments/shot.webp"`) {
		t.Errorf("image embed not rendered as <img>:\n%s", out)
	}
}

func TestRender_ImageEmbed_Alias(t *testing.T) {
	r := NewRenderer()
	res := bothResolver{img: map[string]string{"d.png": "/vault-files/d.png"}}
	out, _ := r.Render([]byte("![[d.png|Diagramma]]"), res)
	if !strings.Contains(out, `src="/vault-files/d.png"`) || !strings.Contains(out, `alt="Diagramma"`) {
		t.Errorf("alias/img not rendered:\n%s", out)
	}
}

func TestRender_ImageEmbed_Unresolved(t *testing.T) {
	r := NewRenderer()
	out, _ := r.Render([]byte("![[missing.webp]]"), bothResolver{img: map[string]string{}})
	if strings.Contains(out, "<img") {
		t.Errorf("unresolved embed should not render an <img>:\n%s", out)
	}
}

func TestRender_ImageEmbed_InertWithoutImageResolver(t *testing.T) {
	// A plain Resolver (no ImageResolver) must leave ![[...]] inert — backward
	// compatibility for the HTML handlers / MCP rendered body.
	r := NewRenderer()
	out, _ := r.Render([]byte("![[x.png]]"), ResolverFunc(func(string) string { return "" }))
	if strings.Contains(out, "<img") {
		t.Errorf("image embeds must be inert without an ImageResolver:\n%s", out)
	}
}
