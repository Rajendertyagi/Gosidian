package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosidian/gosidian/internal/attach"
	"github.com/gosidian/gosidian/internal/auth"
	"github.com/gosidian/gosidian/internal/index"
	"github.com/gosidian/gosidian/internal/vault"
)

// serverWithToken builds an MCP server and returns it plus a valid bearer token
// plaintext, for exercising the HTTP upload endpoint directly.
func serverWithToken(t *testing.T, project string, scopes []string) (*Server, string) {
	t.Helper()
	idx, err := index.Open(filepath.Join(t.TempDir(), "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { idx.Close() })
	store, err := auth.Open(filepath.Join(t.TempDir(), "tokens.json"))
	if err != nil {
		t.Fatal(err)
	}
	plaintext, _, err := store.Create("test", splitProjects(project), scopes, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	return New(vault.New(t.TempDir()), idx, store), plaintext
}

func multipartPNG(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	png, _ := base64.StdEncoding.DecodeString(onePxPNG)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "shot.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(png); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestHTTPUpload_Bearer(t *testing.T) {
	s, token := serverWithToken(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	body, contentType := multipartPNG(t)

	req := httptest.NewRequest(http.MethodPost, "/mcp/upload?project=pics", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.handleHTTPUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["kind"] != "image" || resp["mime"] != "image/png" {
		t.Errorf("resp = %v", resp)
	}
	path, _ := resp["path"].(string)
	if !strings.HasPrefix(path, "pics/attachments/") {
		t.Errorf("path = %q", path)
	}
	if !s.vault.Exists(path) {
		t.Errorf("attachment not stored at %q", path)
	}
}

func TestHTTPUpload_RejectsNoToken(t *testing.T) {
	s, _ := serverWithToken(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	body, contentType := multipartPNG(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	s.handleHTTPUpload(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401; body = %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPUpload_RejectsReadOnly(t *testing.T) {
	s, token := serverWithToken(t, "", []string{auth.ScopeRead})
	body, contentType := multipartPNG(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.handleHTTPUpload(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (no write scope)", rec.Code)
	}
}

func TestCreateMediaNote_Attachment(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)

	// Pre-upload an image (as POST /mcp/upload would), then reference it.
	png, _ := base64.StdEncoding.DecodeString(onePxPNG)
	res, err := attach.Store(s.vault, png, "diagram.png", "pics")
	if err != nil {
		t.Fatal(err)
	}

	r, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "attachment": res.Path, "title": "Ref", "caption": "referenced, not re-uploaded",
	}))
	if r.IsError {
		t.Fatalf("unexpected error: %s", expectError(t, r))
	}
	note, err := s.vault.Load("pics/ref.md")
	if err != nil {
		t.Fatalf("note not created: %v", err)
	}
	if !strings.Contains(string(note.Content), "media: "+res.Path) {
		t.Errorf("note does not reference the attachment:\n%s", note.Content)
	}
	if ref, kind := s.vault.MediaRefForNote(note.Path, note.Content); kind != "image" || ref.Broken {
		t.Errorf("media ref not resolved: kind=%q ref=%+v", kind, ref)
	}
}

func TestCreateMediaNote_AttachmentMissing(t *testing.T) {
	s, ctx := newScopedServer(t, "", []string{auth.ScopeRead, auth.ScopeWrite})
	s.vault.SetMediaNotes(true)
	r, _ := s.handleCreateMediaNote(ctx, call(map[string]any{
		"project": "pics", "attachment": "pics/attachments/nope.png", "caption": "x",
	}))
	if msg := expectError(t, r); !strings.Contains(msg, "not found") {
		t.Errorf("expected not-found error, got %q", msg)
	}
}
