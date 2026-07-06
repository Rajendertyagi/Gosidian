package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// bigNoteBody builds a note larger than the memory_get soft cap, with real
// headings so the truncated response carries a useful outline.
func bigNoteBody() string {
	var b strings.Builder
	b.WriteString("---\ntitle: big\ntags: [proj]\n---\n\n# Big\n\n")
	filler := strings.Repeat("lorem ipsum dolor sit amet ", 40) + "\n\n"
	for i := 0; i < 100; i++ {
		b.WriteString("## Section ")
		b.WriteByte(byte('A' + i%26))
		b.WriteString("\n\n")
		b.WriteString(filler)
	}
	return b.String()
}

func getNote(t *testing.T, s *Server, ctx context.Context, args map[string]any) noteContent {
	t.Helper()
	res, err := s.handleGet(ctx, call(args))
	if err != nil {
		t.Fatal(err)
	}
	var nc noteContent
	if err := json.Unmarshal([]byte(resultText(t, res)), &nc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return nc
}

func TestGet_OversizeGuard(t *testing.T) {
	s, _, _ := newTestServer(t)
	ctx := context.Background()
	content := bigNoteBody()
	if len(content) <= getBodySoftCap {
		t.Fatalf("fixture too small: %d bytes", len(content))
	}
	if res, _ := s.handleCreate(ctx, call(map[string]any{"path": "proj/big.md", "content": content})); res.IsError {
		t.Fatalf("seed: %s", expectError(t, res))
	}

	// Default: truncated with outline + hint + full-note etag.
	nc := getNote(t, s, ctx, map[string]any{"path": "proj/big.md"})
	if !nc.Truncated {
		t.Fatal("oversize body should be truncated by default")
	}
	if len(nc.Content) > getTruncChunk {
		t.Errorf("chunk = %d bytes, want <= %d", len(nc.Content), getTruncChunk)
	}
	if nc.Size != int64(len(content)) {
		t.Errorf("size = %d, want %d", nc.Size, len(content))
	}
	if len(nc.Headings) == 0 || nc.Hint == "" || nc.Frontmatter == "" {
		t.Errorf("truncated response should carry outline+hint+frontmatter: %+v", nc.Hint)
	}
	// Outline cap: 101 headings in the fixture → capped to 80, total reported.
	if len(nc.Headings) != getTruncMaxHeadings || nc.OutlineTotal != 101 {
		t.Errorf("outline cap: len=%d total=%d, want %d/101", len(nc.Headings), nc.OutlineTotal, getTruncMaxHeadings)
	}
	if !strings.Contains(nc.Hint, "outline capped") {
		t.Errorf("hint should mention the outline cap: %s", nc.Hint)
	}
	if nc.ETag == "" {
		t.Error("etag must be present (stamps the full note)")
	}

	// raw:true bypasses the guard.
	nc = getNote(t, s, ctx, map[string]any{"path": "proj/big.md", "raw": true})
	if nc.Truncated || nc.Content != content {
		t.Error("raw:true should return the full body untruncated")
	}

	// Small note: untouched, no guard fields.
	if res, _ := s.handleCreate(ctx, call(map[string]any{"path": "proj/small.md", "content": "---\ntitle: s\n---\n\nshort"})); res.IsError {
		t.Fatalf("seed small: %s", expectError(t, res))
	}
	nc = getNote(t, s, ctx, map[string]any{"path": "proj/small.md"})
	if nc.Truncated || nc.Hint != "" {
		t.Error("small note must not be truncated")
	}

	// Explicit max_bytes truncates even below the default threshold.
	nc = getNote(t, s, ctx, map[string]any{"path": "proj/big.md", "max_bytes": 1000})
	if !nc.Truncated || len(nc.Content) > 1000 {
		t.Errorf("max_bytes: truncated=%v len=%d", nc.Truncated, len(nc.Content))
	}
}
