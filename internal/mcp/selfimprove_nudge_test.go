package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestNudgeTracker_Cadence(t *testing.T) {
	tr := newNudgeTracker()
	base := time.Unix(0, 0)
	// everyN=3, cap=2, cooldown=0 → fires on the 3rd and 6th call, then
	// the per-session cap stops it.
	var got []bool
	for i := 0; i < 7; i++ {
		got = append(got, tr.tick("k", 3, 2, 0, base))
	}
	want := []bool{false, false, true, false, false, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tick %d = %v, want %v (all=%v)", i+1, got[i], want[i], got)
		}
	}
}

func TestNudgeTracker_Cooldown(t *testing.T) {
	tr := newNudgeTracker()
	base := time.Unix(1000, 0)
	if !tr.tick("k", 1, 5, 10*time.Minute, base) {
		t.Fatal("first tick should fire")
	}
	if tr.tick("k", 1, 5, 10*time.Minute, base.Add(time.Minute)) {
		t.Error("should be throttled within cooldown")
	}
	if !tr.tick("k", 1, 5, 10*time.Minute, base.Add(11*time.Minute)) {
		t.Error("should fire after cooldown elapses")
	}
}

func TestNudgeTracker_DisabledWhenZero(t *testing.T) {
	tr := newNudgeTracker()
	if tr.tick("k", 0, 1, 0, time.Unix(0, 0)) {
		t.Error("everyN=0 must never fire")
	}
	if tr.tick("k", 5, 0, 0, time.Unix(0, 0)) {
		t.Error("maxPerSession=0 must never fire")
	}
}

func TestNudgeTracker_PerSessionIsolation(t *testing.T) {
	tr := newNudgeTracker()
	now := time.Unix(0, 0)
	// Two distinct sessions each reach their own threshold independently.
	tr.tick("a", 2, 1, 0, now)
	if !tr.tick("a", 2, 1, 0, now) {
		t.Error("session a should fire on its 2nd call")
	}
	if tr.tick("b", 2, 1, 0, now) {
		t.Error("session b should not fire on its 1st call")
	}
}

func nudgeNext(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return mcplib.NewToolResultText("ok"), nil
}

func TestNudge_MiddlewareAppendsForOptedIn(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	s.SetSelfImproveNudge(1, 5, 0) // every call, no cooldown
	wrapped := s.selfImproveNudgeMiddleware(nudgeNext)
	res, _ := wrapped(optInCtx(true), call(map[string]any{}))
	if !strings.Contains(resultText(t, res), "self-improvement") {
		t.Error("expected nudge appended for opted-in token")
	}
}

func TestNudge_MiddlewareSilentForNonOptedIn(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	s.SetSelfImproveNudge(1, 5, 0)
	wrapped := s.selfImproveNudgeMiddleware(nudgeNext)
	res, _ := wrapped(optInCtx(false), call(map[string]any{}))
	if strings.Contains(resultText(t, res), "self-improvement") {
		t.Error("must not nudge a non-opted-in token")
	}
}

func TestNudge_MiddlewareSilentWhenDisabled(t *testing.T) {
	s, _, _ := newTestServer(t)
	// enabled defaults false
	s.SetSelfImproveNudge(1, 5, 0)
	wrapped := s.selfImproveNudgeMiddleware(nudgeNext)
	res, _ := wrapped(optInCtx(true), call(map[string]any{}))
	if strings.Contains(resultText(t, res), "self-improvement") {
		t.Error("must not nudge when the loop is disabled")
	}
}

func TestNudge_MiddlewareSkipsSelfImproveCall(t *testing.T) {
	s, _, _ := newTestServer(t)
	s.SetSelfImprove(true, "insights")
	s.SetSelfImproveNudge(1, 5, 0)
	wrapped := s.selfImproveNudgeMiddleware(nudgeNext)
	req := call(map[string]any{})
	req.Params.Name = "memory_self_improve"
	res, _ := wrapped(optInCtx(true), req)
	if strings.Contains(resultText(t, res), "self-improvement") {
		t.Error("must not nudge on the memory_self_improve call itself")
	}
}
