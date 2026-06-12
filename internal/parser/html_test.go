package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractHTML(t *testing.T) {
	body := []byte("<!--\n" +
		"---\n" +
		"title: Dashboard\n" +
		"tags: [proj, type:doc]\n" +
		"---\n" +
		"-->\n" +
		"<html>\n" +
		"<head><style>.x { color: #fff; }</style></head>\n" +
		"<body>\n" +
		"<h1>Dashboard</h1>\n" +
		"<p>See [[proj/parser]] and [[proj/index|the index]].</p>\n" +
		"<script>const u = \"[[not-a-link]]\"; console.log(\"#nothashtag\");</script>\n" +
		"</body>\n" +
		"</html>")

	links, tags, title, text := ExtractHTML(body)

	if title != "Dashboard" {
		t.Errorf("title = %q, want Dashboard", title)
	}
	if want := []string{"proj", "type:doc"}; !reflect.DeepEqual(tags, want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
	// Two wikilinks from the body; the one inside <script> is stripped.
	if len(links) != 2 {
		t.Fatalf("links = %+v, want 2", links)
	}
	if links[0].Target != "proj/parser" {
		t.Errorf("link[0] = %+v, want proj/parser", links[0])
	}
	if links[1].Target != "proj/index" || links[1].Alias != "the index" {
		t.Errorf("link[1] = %+v, want proj/index|the index", links[1])
	}
	// FTS text keeps visible prose, drops script/style content.
	if !strings.Contains(text, "Dashboard") || !strings.Contains(text, "See") {
		t.Errorf("text missing visible prose: %q", text)
	}
	for _, leak := range []string{"console.log", "color", "#fff", "not-a-link"} {
		if strings.Contains(text, leak) {
			t.Errorf("text leaked %q: %q", leak, text)
		}
	}
	// Tags must not be harvested from the body's '#' tokens.
	for _, tg := range tags {
		if tg == "nothashtag" || tg == "fff" {
			t.Errorf("tag harvested from HTML body: %v", tags)
		}
	}
}

func TestExtractHTML_NoFrontmatter(t *testing.T) {
	body := []byte("<html><body><p>hello &amp; bye [[other]]</p></body></html>")
	links, tags, title, text := ExtractHTML(body)
	if title != "" || len(tags) != 0 {
		t.Errorf("expected empty title/tags, got %q / %v", title, tags)
	}
	if len(links) != 1 || links[0].Target != "other" {
		t.Errorf("links = %+v, want [other]", links)
	}
	// Entities are decoded in the FTS text.
	if !strings.Contains(text, "hello & bye") {
		t.Errorf("text = %q, want decoded '&'", text)
	}
}

func TestExtractHTMLFrontmatterRaw(t *testing.T) {
	body := []byte("<!--\n---\ntitle: w\ntags: [proj, type:doc]\n---\n-->\n<html><body>x</body></html>\n")
	raw := ExtractHTMLFrontmatterRaw(body)
	if !strings.Contains(raw, "title: w") || !strings.Contains(raw, "tags: [proj, type:doc]") {
		t.Fatalf("raw frontmatter = %q, want the inner YAML", raw)
	}
	// The inner YAML must feed the shared frontmatter machinery cleanly.
	fm := ParseFrontmatterFields(raw)
	if fm["title"] != "w" {
		t.Errorf("ParseFrontmatterFields(raw) title = %v, want w", fm["title"])
	}

	// No leading comment-wrapped block → empty (not the markdown form).
	if got := ExtractHTMLFrontmatterRaw([]byte("<html><body>x</body></html>")); got != "" {
		t.Errorf("expected empty for note without HTML frontmatter, got %q", got)
	}
}
