package index

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/parser"
)

// TestFrontmatterDetectionConsistency is the cross-subsystem guard requested by
// the self-improve insight that motivated parser.FrontmatterRawForPath: a note
// whose frontmatter the indexer reads must yield the SAME title and tags when
// parsed through the shared primitive that the linter and MCP tools use. If a
// future change generalizes a note format in one detection path but not the
// other, this fails loudly instead of letting the two silently disagree (the
// BUG-012 class: a well-formed .html note that indexes fine yet fails lint).
func TestFrontmatterDetectionConsistency(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "markdown",
			path: "proj/note.md",
			body: "---\ntitle: Shared Title\ntags: [proj, type:doc]\n---\n\nbody [[proj/other]]\n",
		},
		{
			name: "html",
			path: "proj/note.html",
			body: "<!--\n---\ntitle: Shared Title\ntags: [proj, type:doc]\n---\n-->\n<html><body>x [[proj/other]]</body></html>\n",
		},
	}

	covered := map[string]bool{}
	for _, tc := range cases {
		covered[strings.ToLower(filepath.Ext(tc.path))] = true
		t.Run(tc.name, func(t *testing.T) {
			_, idxTags, idxTitle, _ := extractForPath(tc.path, tc.body)

			fm := parser.ParseFrontmatterFields(parser.FrontmatterRawForPath(tc.path, []byte(tc.body)))
			fmTitle, _ := fm["title"].(string)
			fmTags, _ := fm["tags"].([]string)

			if fmTitle == "" || len(fmTags) == 0 {
				t.Fatalf("primitive read no frontmatter for %s note (title=%q tags=%v) — detection paths disagree", tc.name, fmTitle, fmTags)
			}
			if idxTitle != fmTitle {
				t.Errorf("title drift: indexer=%q primitive=%q", idxTitle, fmTitle)
			}
			// The indexer may surface extra body #tags (markdown), but every
			// frontmatter tag the primitive sees must also reach the index.
			idx := map[string]bool{}
			for _, tg := range idxTags {
				idx[tg] = true
			}
			for _, tg := range fmTags {
				if !idx[tg] {
					t.Errorf("frontmatter tag %q seen by primitive but missing from indexer tags %v", tg, idxTags)
				}
			}
		})
	}

	// Structural enforcement of the insight's "enumerate every consumer" rule:
	// every note extension the indexer dispatches on must have a consistency
	// fixture above, so a newly-supported note kind cannot be added to one
	// detection path and forgotten in the other without this test failing.
	for _, e := range noteExts {
		if !covered[e] {
			t.Errorf("note extension %q has no frontmatter-consistency fixture; add one so the indexer and parser.FrontmatterRawForPath cannot drift on it", e)
		}
	}
}
