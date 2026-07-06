package mcp

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/auth"
)

// 1x1 transparent PNG, base64 — a real image so attach's magic-byte check passes.
const onePxPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func TestCreateMediaNote_Gating(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	// media_notes is off by default → the tool refuses.
	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "data": onePxPNG, "filename": "dot.png", "caption": "x",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "media notes are disabled") {
		t.Errorf("expected disabled error, got %q", msg)
	}
}

func TestCreateMediaNote_Success(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)

	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "data": onePxPNG, "filename": "dot.png",
		"title": "My Diagram", "caption": "Una immagine di prova.",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}

	// Path derived from the title slug.
	rel := "pics/my-diagram.md"
	note, err := s.vault.Load(rel)
	if err != nil {
		t.Fatalf("note not created at %s: %v", rel, err)
	}
	body := string(note.Content)
	if !strings.Contains(body, "type: image") || !strings.Contains(body, "media: pics/attachments/") {
		t.Errorf("frontmatter missing media fields:\n%s", body)
	}
	if !strings.Contains(body, "Una immagine di prova.") {
		t.Error("caption not written to body")
	}
	// The note resolves as a (non-broken) image media note end-to-end.
	ref, kind := s.vault.MediaRefForNote(note.Path, note.Content)
	if kind != "image" || ref.Broken || ref.MIME != "image/png" {
		t.Errorf("media ref not resolved: kind=%q ref=%+v", kind, ref)
	}
}

func TestCreateMediaNote_RejectsNonImage(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)
	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "data": "aGVsbG8=", "filename": "notes.txt", "caption": "x",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "images only") {
		t.Errorf("expected images-only rejection, got %q", msg)
	}
}

func TestCreateMediaNote_EmptyCaptionWarns(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)
	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "data": onePxPNG, "filename": "dot.png",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}
	if !strings.Contains(resultText(t, res), "empty caption") {
		t.Error("expected an empty-caption warning in the result")
	}
}

func TestCreateMediaNote_BridgeFilename(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)
	bridge := t.TempDir()
	s.SetBridgeDir(bridge)

	// Stage a real PNG in the bridge dir (as an agent would, via a shared mount).
	png, _ := base64.StdEncoding.DecodeString(onePxPNG)
	staged := filepath.Join(bridge, "shot.png")
	if err := os.WriteFile(staged, png, 0o644); err != nil {
		t.Fatal(err)
	}

	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "bridge_filename": "shot.png", "caption": "staged via bridge",
	}))
	if res.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, res))
	}

	note, err := s.vault.Load("pics/shot.md")
	if err != nil {
		t.Fatalf("note not created at pics/shot.md: %v", err)
	}
	if ref, kind := s.vault.MediaRefForNote(note.Path, note.Content); kind != "image" || ref.Broken || ref.MIME != "image/png" {
		t.Errorf("media ref not resolved: kind=%q ref=%+v", kind, ref)
	}
	// The staged file must be consumed after a successful upload.
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Errorf("staged bridge file should have been consumed, stat err=%v", err)
	}
}

func TestCreateMediaNote_BridgeNotConfigured(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)
	// bridge_filename with no bridge dir configured → clear error, no note.
	res, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "bridge_filename": "x.png", "caption": "c",
	}))
	if msg := expectError(t, res); !strings.Contains(msg, "no bridge dir") {
		t.Errorf("expected bridge-not-configured error, got %q", msg)
	}
}

func TestSlugifyFilename(t *testing.T) {
	cases := map[string]string{
		"My Diagram":      "my-diagram",
		"  spaced  out  ": "spaced-out",
		"a/b\\c":          "a-b-c",
		"!!!":             "image",
		"Café 2024":       "caf-2024",
	}
	for in, want := range cases {
		if got := slugifyFilename(in); got != want {
			t.Errorf("slugifyFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
