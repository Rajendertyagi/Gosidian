package v1

import (
	"net/http"
	"strings"
	"testing"
)

func TestInsightsPending_OwnerSeesPending(t *testing.T) {
	SelfImproveEnabled = true
	SelfImproveProject = "insights"
	t.Cleanup(func() { SelfImproveEnabled = false; SelfImproveProject = "insights" })

	f := newAdminFixture(t)
	f.seedNote(t, "insights/2026-06-08-x-abcd.md",
		"---\ntitle: x\ntags: [insights, type:insight, status:pending]\ntype: insight\nstatus: pending\n---\n\n# x\n")
	// A non-pending note in the same project must not be counted.
	f.seedNote(t, "insights/README.md",
		"---\ntitle: r\ntags: [insights, type:index]\ntype: index\n---\n\n# r\n")

	w := f.doAuthRecorder(http.MethodGet, "/api/v1/insights/pending", "", nil)
	if w.code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.code, w.body)
	}
	if !strings.Contains(w.body, `"enabled":true`) {
		t.Errorf("expected enabled true: %s", w.body)
	}
	if !strings.Contains(w.body, `"count":1`) {
		t.Errorf("expected count 1: %s", w.body)
	}
}

func TestInsightsPending_DisabledReturnsEmpty(t *testing.T) {
	SelfImproveEnabled = false
	f := newAdminFixture(t)
	w := f.doAuthRecorder(http.MethodGet, "/api/v1/insights/pending", "", nil)
	if w.code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.code, w.body)
	}
	if !strings.Contains(w.body, `"enabled":false`) || !strings.Contains(w.body, `"count":0`) {
		t.Errorf("disabled should return enabled:false count:0: %s", w.body)
	}
}

func TestInsightsPending_MemberForbidden(t *testing.T) {
	SelfImproveEnabled = true
	t.Cleanup(func() { SelfImproveEnabled = false })
	f := newAdminFixture(t)
	mem := f.memberToken(t)
	w := f.req(t, http.MethodGet, "/api/v1/insights/pending", "", mem)
	if w.code != http.StatusForbidden {
		t.Errorf("member should be forbidden, got %d: %s", w.code, w.body)
	}
}
