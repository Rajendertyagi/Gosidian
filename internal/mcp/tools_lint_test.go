package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

// TestMCP_Lint_RulesEchoReflectsRequest guards BUG-013: the response `rules`
// field must report the rules actually run, not a hardcoded default. When an
// explicit (e.g. opt-in) rule is requested it must be echoed; with no rules the
// default set is echoed.
func TestMCP_Lint_RulesEchoReflectsRequest(t *testing.T) {
	s, _, _ := newTestServer(t)
	ctx := context.Background()

	// One note so the run has something to scan.
	if _, err := s.handleCreate(ctx, call(map[string]any{
		"path":    "proj/a.md",
		"content": "---\ntitle: a\ntags: [proj]\n---\n\n# a\n",
	})); err != nil {
		t.Fatal(err)
	}

	type lintResp struct {
		Rules []string `json:"rules"`
	}
	parse := func(t *testing.T, body string) lintResp {
		t.Helper()
		var got lintResp
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatalf("parse: %v body=%s", err, body)
		}
		return got
	}

	// Explicit opt-in rule → echo must reflect exactly what was requested.
	res, err := s.handleLint(ctx, call(map[string]any{
		"project": "proj",
		"rules":   []any{"unlinked-mentions"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := parse(t, resultText(t, res))
	if len(got.Rules) != 1 || got.Rules[0] != "unlinked-mentions" {
		t.Fatalf("rules echo = %v, want [unlinked-mentions]", got.Rules)
	}

	// No rules → echo the default set (starts with broken-wikilink, excludes
	// the opt-in unlinked-mentions).
	res, err = s.handleLint(ctx, call(map[string]any{"project": "proj"}))
	if err != nil {
		t.Fatal(err)
	}
	got = parse(t, resultText(t, res))
	if len(got.Rules) == 0 || got.Rules[0] != "broken-wikilink" {
		t.Fatalf("default rules echo = %v, want default set starting broken-wikilink", got.Rules)
	}
	for _, r := range got.Rules {
		if r == "unlinked-mentions" {
			t.Fatalf("default echo must not include opt-in unlinked-mentions: %v", got.Rules)
		}
	}
}
