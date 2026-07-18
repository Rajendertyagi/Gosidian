package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// With the per-path lock held across load→checkIfMatch→write, if_match is a
// true compare-and-swap: among N concurrent updates carrying the same etag,
// exactly one wins and the others get an etag-mismatch error.
func TestMCP_UpdateIfMatchIsCompareAndSwap(t *testing.T) {
	s, v, _ := newTestServer(t)
	ctx := context.Background()

	res, _ := s.handleCreate(ctx, call(map[string]any{"path": "p/note.md", "content": "seed"}))
	resultText(t, res)
	note, err := v.Load("p/note.md")
	if err != nil {
		t.Fatal(err)
	}
	etag := note.ETag()

	const n = 8
	var wg sync.WaitGroup
	success := make([]bool, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Distinct lengths → every outcome has a distinct etag/size.
			content := strings.Repeat("x", 10+i)
			r, _ := s.handleUpdate(ctx, call(map[string]any{
				"path": "p/note.md", "content": content, "if_match": etag,
			}))
			success[i] = r != nil && !r.IsError
		}(i)
	}
	wg.Wait()

	winners := 0
	winner := -1
	for i, ok := range success {
		if ok {
			winners++
			winner = i
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly 1 winning update, got %d", winners)
	}
	final, err := v.Load("p/note.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Repeat("x", 10+winner); string(final.Content) != want {
		t.Fatalf("final content is not the winner's: got %d bytes, want %d", len(final.Content), len(want))
	}
}

// Concurrent memory_ask calls must mint distinct OQ ids (the load→next-id→
// write sequence runs under the path lock).
func TestMCP_AskConcurrentDistinctOQIDs(t *testing.T) {
	s, _, _ := newTestServer(t)
	ctx := context.Background()

	const n = 6
	var wg sync.WaitGroup
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, _ := s.handleAsk(ctx, call(map[string]any{
				"project":  "p",
				"question": "question " + strings.Repeat("?", i+1),
			}))
			if r == nil || r.IsError {
				return
			}
			var out struct {
				OQID string `json:"oq_id"`
			}
			for _, c := range r.Content {
				if tc, ok := c.(mcplib.TextContent); ok {
					_ = json.Unmarshal([]byte(tc.Text), &out)
				}
			}
			ids[i] = out.OQID
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for i, id := range ids {
		if id == "" {
			t.Fatalf("ask %d failed or returned no oq_id", i)
		}
		if seen[id] {
			t.Fatalf("duplicate OQ id %q minted by concurrent asks", id)
		}
		seen[id] = true
	}
}
