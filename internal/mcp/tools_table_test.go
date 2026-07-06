package mcp

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
)

const sampleCSV = "user,action,ts\nalice,login,1\nbob,logout,2\n"

func csvB64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestCreateTableNote_Gating(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	// table_notes is off by default → the tool refuses.
	res, _ := s.handleCreateTableNote(ctx, call(map[string]any{
		"project": "audit", "data": csvB64(sampleCSV), "filename": "log.csv", "caption": "x",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "table notes are disabled") {
		t.Errorf("expected disabled error, got %q", msg)
	}
}

func TestCreateTableNote_Success(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetTableNotes(true)

	res, _ := s.handleCreateTableNote(ctx, call(map[string]any{
		"project": "audit", "data": csvB64(sampleCSV), "filename": "log.csv",
		"title": "Access Log July", "caption": "Accessi al portale, export luglio.",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}
	out := resultText(t, res)
	if !strings.Contains(out, `"kind":"table"`) || !strings.Contains(out, `"rows":2`) {
		t.Errorf("result missing kind/rows: %s", out)
	}

	// Path derived from the title slug.
	rel := "audit/access-log-july.md"
	note, err := s.vault.Load(rel)
	if err != nil {
		t.Fatalf("note not created at %s: %v", rel, err)
	}
	body := string(note.Content)
	if !strings.Contains(body, "type: table") || !strings.Contains(body, "media: audit/attachments/") {
		t.Errorf("frontmatter missing table fields:\n%s", body)
	}
	// The FTS surface: caption + auto-inlined headers + row count.
	if !strings.Contains(body, "Accessi al portale") ||
		!strings.Contains(body, "Columns: user, action, ts") ||
		!strings.Contains(body, "Rows: 2") {
		t.Errorf("body missing caption/columns/rows summary:\n%s", body)
	}
	// The note resolves as a (non-broken) table note end-to-end.
	ref, kind := s.vault.MediaRefForNote(note.Path, note.Content)
	if kind != "table" || ref.Broken || ref.MIME != "text/csv" {
		t.Errorf("table ref not resolved: kind=%q ref=%+v", kind, ref)
	}
}

func TestCreateTableNote_RejectsNonCSV(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetTableNotes(true)
	res, _ := s.handleCreateTableNote(ctx, call(map[string]any{
		"project": "audit", "data": csvB64("hello"), "filename": "notes.txt", "caption": "x",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, ".csv only") {
		t.Errorf("expected csv-only rejection, got %q", msg)
	}
}

func TestCreateTableNote_SemicolonDelimiter(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetTableNotes(true)
	res, _ := s.handleCreateTableNote(ctx, call(map[string]any{
		"project": "audit", "data": csvB64("user;action;ts\nalice;login;1\n"), "filename": "eu.csv",
		"caption": "export EU-locale",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}
	note, err := s.vault.Load("audit/eu.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(note.Content), "Columns: user, action, ts") {
		t.Errorf("semicolon delimiter not sniffed:\n%s", note.Content)
	}
}

func TestCreateTableNote_EmptyCaptionWarns(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetTableNotes(true)
	res, _ := s.handleCreateTableNote(ctx, call(map[string]any{
		"project": "audit", "data": csvB64(sampleCSV), "filename": "log.csv",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}
	if out := resultText(t, res); !strings.Contains(out, "warning") {
		t.Errorf("expected empty-caption warning, got %s", out)
	}
}

func TestCSVSummary(t *testing.T) {
	cols, rows, err := csvSummary([]byte("a,b\n\"x,1\",2\ny,3\n"))
	if err != nil || rows != 2 || len(cols) != 2 || cols[0] != "a" {
		t.Errorf("quoted csv: cols=%v rows=%d err=%v", cols, rows, err)
	}
	if _, _, err := csvSummary(nil); err == nil {
		t.Error("empty input should be an error (no header row)")
	}
	cols, rows, err = csvSummary([]byte("a\tb\n1\t2\n"))
	if err != nil || rows != 1 || len(cols) != 2 {
		t.Errorf("tab csv: cols=%v rows=%d err=%v", cols, rows, err)
	}
}
