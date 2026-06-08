package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/gosidian/gosidian/internal/auth"
	"github.com/gosidian/gosidian/internal/server/events"
)

// optInCtx returns a context carrying a write token, as the SSE pipeline
// would inject for a real dogfood token. optIn toggles the per-token
// self-improve opt-in flag.
func optInCtx(optIn bool) context.Context {
	tok := &auth.Token{
		ID:               "deadbeef",
		Name:             "dogfood",
		Scopes:           []string{auth.ScopeRead, auth.ScopeWrite},
		SelfImproveOptIn: optIn,
	}
	return context.WithValue(context.Background(), tokenCtxKey, tok)
}

func TestSelfImprove_DisabledRejects(t *testing.T) {
	s, _, _ := newTestServer(t)
	// enabled defaults to false → every call rejected.
	res, _ := s.handleSelfImprove(optInCtx(true), call(map[string]any{
		"category": "bug", "title": "x", "friction": "y", "confidence": "high",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "disabled") {
		t.Errorf("expected disabled error, got %q", msg)
	}
}

func TestSelfImprove_NotOptedInRejects(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	res, _ := s.handleSelfImprove(optInCtx(false), call(map[string]any{
		"category": "bug", "title": "x", "friction": "y", "confidence": "high",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "not opted in") {
		t.Errorf("expected opt-in error, got %q", msg)
	}
}

func TestSelfImprove_RecordsInsight(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	ctx := optInCtx(true)

	res, _ := s.handleSelfImprove(ctx, call(map[string]any{
		"category":    "performance",
		"title":       "Graph slow on large vault",
		"friction":    "Rendering the graph took noticeably long with many notes.",
		"confidence":  "medium",
		"suggestion":  "Consider server-side degree filtering by default.",
		"agent_label": "opus-4.8",
	}))
	body := resultText(t, res)

	var r struct {
		Path     string `json:"path"`
		Category string `json:"category"`
		Status   string `json:"status"`
		ETag     string `json:"etag"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, body)
	}
	if r.Category != "performance" || r.Status != "pending" {
		t.Errorf("unexpected result: %+v", r)
	}
	if !strings.HasPrefix(r.Path, "insights/") || !strings.HasSuffix(r.Path, ".md") {
		t.Errorf("path should be under insights/: %s", r.Path)
	}
	if r.ETag == "" {
		t.Error("expected etag in response")
	}

	get, _ := s.handleGet(ctx, call(map[string]any{"path": r.Path}))
	note := resultText(t, get)
	for _, want := range []string{
		"type: insight",
		"status: pending",
		"category: performance",
		"confidence: medium",
		"tags: [insights, type:insight, status:pending]",
		"agent_label:",
		"## Friction",
		"## Suggestion",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("insight note missing %q\n---\n%s", want, note)
		}
	}
	// Privacy: source_token is the hashed id prefix, never the plaintext.
	if !strings.Contains(note, "source_token: deadbeef") {
		t.Errorf("source_token not recorded as hashed id:\n%s", note)
	}
}

func TestSelfImprove_DefaultsProjectToInsights(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "") // empty project → built-in default
	res, _ := s.handleSelfImprove(optInCtx(true), call(map[string]any{
		"category": "meta", "title": "t", "friction": "f", "confidence": "low",
	}))
	body := resultText(t, res)
	if !strings.Contains(body, `"path":"insights/`) {
		t.Errorf("expected default insights project, got %s", body)
	}
}

func TestSelfImprove_ValidatesEnums(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	ctx := optInCtx(true)

	res, _ := s.handleSelfImprove(ctx, call(map[string]any{
		"category": "nope", "title": "x", "friction": "y", "confidence": "high",
	}))
	if !strings.Contains(expectError(t, res), "category") {
		t.Error("expected category error")
	}
	res, _ = s.handleSelfImprove(ctx, call(map[string]any{
		"category": "bug", "title": "x", "friction": "y", "confidence": "certain",
	}))
	if !strings.Contains(expectError(t, res), "confidence") {
		t.Error("expected confidence error")
	}
	res, _ = s.handleSelfImprove(ctx, call(map[string]any{
		"category": "bug", "title": "  ", "friction": "y", "confidence": "high",
	}))
	if !strings.Contains(expectError(t, res), "title") {
		t.Error("expected title error")
	}
}

func TestSelfImprove_BootstrapSurfacesPending(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	ctx := optInCtx(true)

	_, _ = s.handleSelfImprove(ctx, call(map[string]any{
		"category": "bug", "title": "broken thing", "friction": "y", "confidence": "high",
	}))

	// Bootstrapping ANY project surfaces the owner's pending insights.
	res, _ := s.handleBootstrap(ctx, call(map[string]any{"project": "gosidian"}))
	body := resultText(t, res)
	var p struct {
		Pending struct {
			Count int `json:"count"`
			Notes []struct {
				Path string `json:"path"`
			} `json:"notes"`
		} `json:"pending_insights"`
	}
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, body)
	}
	if p.Pending.Count != 1 {
		t.Fatalf("expected 1 pending insight, got %d\nbody=%s", p.Pending.Count, body)
	}
	if len(p.Pending.Notes) != 1 || !strings.HasPrefix(p.Pending.Notes[0].Path, "insights/") {
		t.Errorf("unexpected pending note: %+v", p.Pending.Notes)
	}
}

func TestSelfImprove_BootstrapOmitsWhenDisabled(t *testing.T) {
	s, _, _ := newTestServer(t)
	// Loop disabled (default) → field absent, zero behaviour change.
	res, _ := s.handleBootstrap(optInCtx(true), call(map[string]any{"project": "gosidian"}))
	if strings.Contains(resultText(t, res), "pending_insights") {
		t.Error("pending_insights must be absent when the loop is disabled")
	}
}

func TestSelfImprove_PublishesInsightEvent(t *testing.T) {
	s, _, _ := newTestServer(t)
	hub := events.New(events.HubOptions{Logger: slog.Default()})
	s.SetEvents(hub)
	s.SetSelfImprove(true, "insights")
	sub := hub.Subscribe(events.TopicInsight)

	_, _ = s.handleSelfImprove(optInCtx(true), call(map[string]any{
		"category": "bug", "title": "x", "friction": "y", "confidence": "high",
	}))

	select {
	case ev := <-sub.Ch:
		if ev.Topic != events.TopicInsight {
			t.Errorf("unexpected topic: %v", ev.Topic)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no insight event published")
	}
}
