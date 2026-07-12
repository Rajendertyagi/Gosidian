package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
)

// rpcMessage drives the real JSON-RPC dispatch (the same path an SSE client
// hits), so the ToolFilter is exercised exactly as in production.
func rpcMessage(t *testing.T, s *Server, ctx context.Context, method, params string) string {
	t.Helper()
	msg := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
	res := s.impl.HandleMessage(ctx, json.RawMessage(msg))
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func listToolNames(t *testing.T, s *Server, ctx context.Context) map[string]struct{} {
	t.Helper()
	out := rpcMessage(t, s, ctx, "tools/list", `{}`)
	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse tools/list: %v — %s", err, out)
	}
	names := make(map[string]struct{}, len(resp.Result.Tools))
	for _, tl := range resp.Result.Tools {
		names[tl.Name] = struct{}{}
	}
	return names
}

func coreCtx(tok *auth.Token) context.Context {
	return context.WithValue(context.Background(), tokenCtxKey, tok)
}

func TestToolProfile_CoreFiltersListAndCall(t *testing.T) {
	s, _, _ := newTestServer(t)
	worker := &auth.Token{ID: "w1", Name: "worker", Scopes: []string{auth.ScopeRead, auth.ScopeWrite}, ToolProfile: auth.ToolProfileCore}

	// Full surface (auth disabled → admin → full profile).
	full := listToolNames(t, s, context.Background())
	if len(full) <= len(coreToolSet) {
		t.Fatalf("full profile should see the whole catalogue, got %d tools", len(full))
	}

	// Core token: exactly the static core set (media/table flags off,
	// no self-improve opt-in on this token).
	core := listToolNames(t, s, coreCtx(worker))
	if len(core) != len(coreToolSet) {
		t.Errorf("core list = %d tools, want %d: %v", len(core), len(coreToolSet), core)
	}
	for name := range coreToolSet {
		if _, ok := core[name]; !ok {
			t.Errorf("core set missing %q in tools/list", name)
		}
	}
	if _, ok := core["memory_audit_tail"]; ok {
		t.Error("full-only tool leaked into the core list")
	}

	// tools/call: hidden tool blocked, core tool served.
	if out := rpcMessage(t, s, coreCtx(worker), "tools/call", `{"name":"memory_audit_tail","arguments":{}}`); !strings.Contains(out, "not found") {
		t.Errorf("call on filtered-out tool should be rejected, got: %s", out)
	}
	if out := rpcMessage(t, s, coreCtx(worker), "tools/call", `{"name":"memory_list_projects","arguments":{}}`); strings.Contains(out, "not found") {
		t.Errorf("core tool should be callable: %s", out)
	}
	// The same hidden tool works on the full profile.
	if out := rpcMessage(t, s, context.Background(), "tools/call", `{"name":"memory_audit_tail","arguments":{}}`); strings.Contains(out, "not found") {
		t.Errorf("full profile should reach memory_audit_tail: %s", out)
	}
}

func TestToolProfile_DynamicAdmissions(t *testing.T) {
	s, _, _ := newTestServer(t)
	worker := &auth.Token{ID: "w2", Name: "worker", Scopes: []string{auth.ScopeRead, auth.ScopeWrite}, ToolProfile: auth.ToolProfileCore}

	// ADR-018: the legacy upload matrix stays hidden from core even with the
	// flags on — memory_ingest is the single door and routes internally.
	s.vault.SetTableNotes(true)
	s.vault.SetMediaNotes(true)
	names := listToolNames(t, s, coreCtx(worker))
	for _, legacy := range []string{
		"memory_create_table_note", "memory_create_media_note",
		"memory_upload_attachment", "memory_upload_resource",
	} {
		if _, ok := names[legacy]; ok {
			t.Errorf("%s should stay full-profile-only under ADR-018", legacy)
		}
	}
	if _, ok := names["memory_ingest"]; !ok {
		t.Error("memory_ingest should be in the core surface")
	}

	// Self-improve opt-in admits memory_self_improve.
	if _, ok := names["memory_self_improve"]; ok {
		t.Error("self_improve should be hidden without opt-in")
	}
	worker.SelfImproveOptIn = true
	names = listToolNames(t, s, coreCtx(worker))
	if _, ok := names["memory_self_improve"]; !ok {
		t.Error("self_improve should appear for an opted-in core token")
	}
}
