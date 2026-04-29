package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// 1x1 transparent PNG, hex-encoded
var pngBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func multipartUpload(t *testing.T, name string, body []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(body); err != nil {
		t.Fatal(err)
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

func TestServer_AttachUploadAndServe(t *testing.T) {
	s, dir := setupServer(t)

	// POST /api/attach?project=<empty> with a tiny png
	body, contentType := multipartUpload(t, "pasted.png", pngBytes)
	r := httptest.NewRequest("POST", "/api/attach", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("attach status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Path     string `json:"path"`
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Path == "" {
		t.Errorf("missing path in response")
	}
	// File on disk
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(resp.Path))); err != nil {
		t.Errorf("file not saved: %v", err)
	}
	// Markdown points at /vault-files/
	if resp.Markdown[:18] != "![](/vault-files/a" {
		t.Errorf("unexpected markdown: %s", resp.Markdown)
	}

	// Now fetch via /vault-files/
	w2 := doReq(t, s, "GET", "/vault-files/"+resp.Path, "", false)
	if w2.Code != http.StatusOK {
		t.Errorf("vault-files fetch status = %d", w2.Code)
	}
	if !bytes.Equal(w2.Body.Bytes(), pngBytes) {
		t.Errorf("served bytes do not match original")
	}
}

func TestServer_AttachWithProject(t *testing.T) {
	s, dir := setupServer(t)
	if _, err := s.vault.CreateProject("Work"); err != nil {
		t.Fatal(err)
	}
	body, contentType := multipartUpload(t, "img.png", pngBytes)
	r := httptest.NewRequest("POST", "/api/attach?project=Work", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if filepath.Dir(resp.Path) != "Work/attachments" {
		t.Errorf("attachment in wrong dir: %s", resp.Path)
	}
	if _, err := os.Stat(filepath.Join(dir, "Work", "attachments")); err != nil {
		t.Errorf("project attachments dir not created: %v", err)
	}
}

func TestServer_AttachRejectsBadType(t *testing.T) {
	s, _ := setupServer(t)
	body, contentType := multipartUpload(t, "evil.exe", []byte("MZ"))
	r := httptest.NewRequest("POST", "/api/attach", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
}

// TestServer_UploadEndpoint covers the new POST /api/upload (v1.11) which
// is the editor-decoupled twin of /api/attach. Response shape is richer
// (mime, kind, size, original_filename, hash) and intentionally drops the
// `markdown` field — caller builds the embed at attach-time.
func TestServer_UploadEndpoint(t *testing.T) {
	s, dir := setupServer(t)
	if _, err := s.vault.CreateProject("Work"); err != nil {
		t.Fatal(err)
	}

	body, contentType := multipartUpload(t, "screenshot.png", pngBytes)
	r := httptest.NewRequest("POST", "/api/upload?project=Work&kind=auto", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Path             string `json:"path"`
		URL              string `json:"url"`
		MIME             string `json:"mime"`
		Kind             string `json:"kind"`
		Size             int    `json:"size"`
		OriginalFilename string `json:"original_filename"`
		Hash             string `json:"hash"`
		Markdown         string `json:"markdown"` // expected to be empty / absent
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Path == "" || resp.URL == "" || resp.Hash == "" {
		t.Errorf("missing core fields: %+v", resp)
	}
	if resp.Kind != "image" {
		t.Errorf("kind = %q, want image", resp.Kind)
	}
	if resp.MIME != "image/png" {
		t.Errorf("mime = %q, want image/png", resp.MIME)
	}
	if resp.Size != len(pngBytes) {
		t.Errorf("size = %d, want %d", resp.Size, len(pngBytes))
	}
	if resp.OriginalFilename != "screenshot.png" {
		t.Errorf("original_filename = %q", resp.OriginalFilename)
	}
	if resp.Markdown != "" {
		t.Errorf("markdown should be absent for /api/upload, got: %q", resp.Markdown)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(resp.Path))); err != nil {
		t.Errorf("file not saved: %v", err)
	}
}

func TestServer_UploadRequiresProject(t *testing.T) {
	s, _ := setupServer(t)
	body, contentType := multipartUpload(t, "x.png", pngBytes)
	r := httptest.NewRequest("POST", "/api/upload", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project, got %d", w.Code)
	}
}

func TestServer_UploadRejectsBadKind(t *testing.T) {
	s, _ := setupServer(t)
	body, contentType := multipartUpload(t, "x.png", pngBytes)
	r := httptest.NewRequest("POST", "/api/upload?project=Work&kind=nonsense", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad kind, got %d", w.Code)
	}
}

func TestServer_UploadRejectsMIMESpoof(t *testing.T) {
	s, _ := setupServer(t)
	if _, err := s.vault.CreateProject("Work"); err != nil {
		t.Fatal(err)
	}
	// Declare .png but send JS-like content
	body, contentType := multipartUpload(t, "fake.png", []byte("alert('xss')\nfunction f(){}"))
	r := httptest.NewRequest("POST", "/api/upload?project=Work", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for MIME spoof, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestServer_UploadRejectsBadExt(t *testing.T) {
	s, _ := setupServer(t)
	if _, err := s.vault.CreateProject("Work"); err != nil {
		t.Fatal(err)
	}
	body, contentType := multipartUpload(t, "evil.exe", []byte("MZ"))
	r := httptest.NewRequest("POST", "/api/upload?project=Work", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
}

func TestServer_VaultFileBlocksNonAttachment(t *testing.T) {
	s, _ := setupServer(t)
	// hello.md exists but is not under attachments/
	w := doReq(t, s, "GET", "/vault-files/hello.md", "", false)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-attachment, got %d", w.Code)
	}
}
