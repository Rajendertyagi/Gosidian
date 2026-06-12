package index

import "testing"

func TestIndex_HTMLNote(t *testing.T) {
	idx := openTest(t)
	// Markdown note the html note links to.
	upsert(t, idx, "proj/parser.md", "Parser", "# Parser")
	// HTML note: frontmatter title + a wikilink, plus a <script> whose content
	// must NOT pollute the FTS body.
	htmlBody := "<!--\n---\ntitle: Dash\ntags: [proj]\n---\n-->\n" +
		"<html><body><h1>Dashboard widget</h1><p>uses [[proj/parser]]</p>" +
		"<script>var secret='zzzsecretzzz'</script></body></html>"
	upsert(t, idx, "proj/dash.html", "dash", htmlBody)

	// Title comes from the HTML frontmatter, overriding the filename.
	n, _ := idx.Note("proj/dash.html")
	if n == nil || n.Title != "Dash" {
		t.Fatalf("html note = %+v, want title Dash", n)
	}
	// Outlink from the html note resolves to the markdown target.
	outs, _ := idx.Outlinks("proj/dash.html")
	if len(outs) != 1 || outs[0].TargetPath != "proj/parser.md" {
		t.Errorf("html outlinks = %+v, want →proj/parser.md", outs)
	}
	// Backlink: the markdown note sees the html note linking to it.
	bl, _ := idx.Backlinks("proj/parser.md")
	if len(bl) != 1 || bl[0].Path != "proj/dash.html" {
		t.Errorf("backlinks = %+v, want [proj/dash.html]", bl)
	}
	// FTS indexes the visible text, not the <script> body.
	if hits, _ := idx.Search("Dashboard", 10); len(hits) != 1 || hits[0].Path != "proj/dash.html" {
		t.Errorf("search Dashboard = %+v, want [proj/dash.html]", hits)
	}
	if hits, _ := idx.Search("zzzsecretzzz", 10); len(hits) != 0 {
		t.Errorf("script content leaked into FTS: %+v", hits)
	}

	// A wikilink to the html note by path-without-extension resolves to .html.
	upsert(t, idx, "proj/ref.md", "Ref", "see [[proj/dash]]")
	outs, _ = idx.Outlinks("proj/ref.md")
	if len(outs) != 1 || outs[0].TargetPath != "proj/dash.html" {
		t.Errorf("ref→dash resolution = %+v, want proj/dash.html", outs)
	}
}
