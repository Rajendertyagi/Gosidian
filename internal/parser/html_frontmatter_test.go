package parser

import "testing"

// An .html note may carry frontmatter either in the canonical HTML-comment form
// (ADR-011) or, tolerantly, as a bare markdown `--- YAML ---` block at the top.
// Both must yield the same title/tags and be detectable as frontmatter.
func TestExtractHTML_FrontmatterForms(t *testing.T) {
	comment := []byte("<!--\n---\ntitle: Guida\ntags: [rc, type:doc]\n---\n-->\n<html><body>ciao</body></html>")
	markdown := []byte("---\ntitle: Guida\ntags: [rc, type:doc]\n---\n<html><body>ciao</body></html>")

	for name, body := range map[string][]byte{"comment": comment, "markdown": markdown} {
		_, tags, title, _ := ExtractHTML(body)
		if title != "Guida" {
			t.Errorf("%s: title = %q, want Guida", name, title)
		}
		if len(tags) != 2 || tags[0] != "rc" || tags[1] != "type:doc" {
			t.Errorf("%s: tags = %v", name, tags)
		}
		if ExtractHTMLFrontmatterRaw(body) == "" {
			t.Errorf("%s: frontmatter not detected", name)
		}
	}
}
