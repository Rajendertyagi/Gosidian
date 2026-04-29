package mcp

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosidian/gosidian/internal/attach"
	"github.com/gosidian/gosidian/internal/audit"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerAttachmentTools adds the four attachment-related tools.
// Called from registerTools().
func (s *Server) registerAttachmentTools() {
	s.impl.AddTool(mcp.NewTool("memory_upload_attachment",
		mcp.WithDescription("Upload a file attachment to the vault. Provide EITHER base64 data OR a source_path (server-side filesystem path). source_path avoids passing large files through the context — the server reads the file directly. The path must be inside the vault or an allowed upload root (GOSIDIAN_MCP_ALLOWED_UPLOAD_ROOTS). Supported types: png, jpg, jpeg, gif, webp, svg, pdf, csv, json, txt, zip, docx, xlsx. Max 10 MiB."),
		mcp.WithString("data", mcp.Description("Base64-encoded file content. Required unless source_path is provided.")),
		mcp.WithString("source_path", mcp.Description("Absolute filesystem path to the file — RESOLVED ON THE SERVER, not the client. Use this only when gosidian and the agent share the same filesystem (local install, Docker volume mount, co-located deploy). For remote setups (SSH tunnel, separate host) use 'data' instead — source_path will fail with a hint pointing you back here. Must be inside the vault or an allowed upload root (GOSIDIAN_MCP_ALLOWED_UPLOAD_ROOTS).")),
		mcp.WithString("filename", mcp.Description("Original filename for extension validation and link text. Required with data, optional with source_path (defaults to basename).")),
		mcp.WithString("project", mcp.Description("Optional project to store the attachment in. Empty stores at vault root.")),
	), s.handleUploadAttachment)

	s.impl.AddTool(mcp.NewTool("memory_list_attachments",
		mcp.WithDescription("List attachments in the vault, optionally filtered by project. Returns path, size, MIME type, and modification time for each attachment."),
		mcp.WithString("project", mcp.Description("Optional project name to scope the listing.")),
	), s.handleListAttachments)

	s.impl.AddTool(mcp.NewTool("memory_delete_attachment",
		mcp.WithDescription("Delete an attachment file from the vault. The file is permanently removed. Does not update notes that reference it."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Vault-relative attachment path (e.g. 'project/attachments/abc123.png').")),
	), s.handleDeleteAttachment)

	s.impl.AddTool(mcp.NewTool("memory_attachment_info",
		mcp.WithDescription("Get metadata about an attachment, including which notes reference it. Use this to check if an attachment is orphaned before deleting."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Vault-relative attachment path.")),
	), s.handleAttachmentInfo)

	s.impl.AddTool(mcp.NewTool("memory_upload_resource",
		mcp.WithDescription("Upload a file resource to the vault, decoupled from any specific note. Use this when an agent needs to stage multiple files (\"upload all, verify, then attach\" pattern) — the response carries the resource handle (path, url, hash, mime, kind, size) but NO embed markdown, so the caller chooses when/where to insert it. Storage is identical to memory_upload_attachment (<project>/attachments/<hash>.<ext>); the difference is purely API ergonomic. Provide EITHER base64 data OR source_path. Magic-bytes verification rejects MIME spoofs (e.g. JS payload declared as .png). Same 10 MiB cap. Supported extensions: png/jpg/jpeg/gif/webp/svg + pdf/csv/json/txt/zip/docx/xlsx."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Vault project to store the resource in. Required (resources without a project context are rare and intentionally not auto-routed).")),
		mcp.WithString("data", mcp.Description("Base64-encoded file content. Required unless source_path is provided.")),
		mcp.WithString("source_path", mcp.Description("Absolute filesystem path to the file — RESOLVED ON THE SERVER, not the client. Use this only when gosidian and the agent share the same filesystem (local install, Docker volume mount, co-located deploy). For remote setups (SSH tunnel, separate host) use 'data' instead — source_path will fail with a hint pointing you back here. Must be inside the vault or an allowed upload root (GOSIDIAN_MCP_ALLOWED_UPLOAD_ROOTS).")),
		mcp.WithString("filename", mcp.Description("Original filename for extension validation. Required with data, optional with source_path (defaults to basename).")),
		mcp.WithString("kind", mcp.Description("Hint for caller branching: image | document | auto (default). Echoed back in the response — server resolves auto from the validated MIME family.")),
	), s.handleUploadResource)
}

func (s *Server) handleUploadAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourcePath := req.GetString("source_path", "")
	dataB64 := req.GetString("data", "")
	filename := req.GetString("filename", "")
	project := req.GetString("project", "")

	if sourcePath == "" && dataB64 == "" {
		return mcp.NewToolResultError("provide either data (base64) or source_path"), nil
	}

	var res *attach.Result
	var dataSize int64

	if sourcePath != "" {
		// Server-side filesystem read — no base64 through context.
		if filename == "" {
			filename = filepath.Base(sourcePath)
		}
		// Auth check: verify write scope on the target attachment path.
		ext := strings.ToLower(filepath.Ext(filename))
		testRel := attach.RelPath(project, "test"+ext)
		if _, errRes := s.authorizeWrite(ctx, testRel); errRes != nil {
			return errRes, nil
		}
		var err error
		res, err = attach.StoreFromPath(s.vault, sourcePath, filename, project, s.effectiveUploadRoots())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		// For audit: stat the stored file to get size.
		if abs, absErr := s.vault.Abs(res.Path); absErr == nil {
			if fi, stErr := os.Stat(abs); stErr == nil {
				dataSize = fi.Size()
			}
		}
	} else {
		// Base64 path — original flow.
		if filename == "" {
			return mcp.NewToolResultError("filename is required when using data (base64)"), nil
		}
		data, err := base64.StdEncoding.DecodeString(dataB64)
		if err != nil {
			return mcp.NewToolResultError("invalid base64: " + err.Error()), nil
		}

		ext := strings.ToLower(filepath.Ext(filename))
		testRel := attach.RelPath(project, "test"+ext)
		tok, errRes := s.authorizeWrite(ctx, testRel)
		if errRes != nil {
			return errRes, nil
		}
		if errRes := s.checkWriteLimits(tok, len(data)); errRes != nil {
			return errRes, nil
		}

		res, err = attach.Store(s.vault, data, filename, project)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dataSize = int64(len(data))
	}

	s.auditWrite(ctx, audit.ActionUploadAttachment, res.Path, "", dataSize)
	return mcp.NewToolResultJSON(map[string]any{
		"path":     res.Path,
		"markdown": res.Markdown,
	})
}

// handleUploadResource is the editor-decoupled twin of
// handleUploadAttachment. Same storage, same validation, same audit
// action — but the response intentionally drops the `markdown` field so
// the caller (typically an agent doing 2-step "upload all, verify,
// then attach") decides when and where to insert the embed.
func (s *Server) handleUploadResource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if strings.TrimSpace(project) == "" {
		return mcp.NewToolResultError("project must not be empty"), nil
	}

	sourcePath := req.GetString("source_path", "")
	dataB64 := req.GetString("data", "")
	filename := req.GetString("filename", "")
	kindHint := strings.ToLower(strings.TrimSpace(req.GetString("kind", "auto")))
	if kindHint == "" {
		kindHint = "auto"
	}
	if kindHint != "image" && kindHint != "document" && kindHint != "auto" {
		return mcp.NewToolResultError("kind must be one of: image, document, auto"), nil
	}

	if sourcePath == "" && dataB64 == "" {
		return mcp.NewToolResultError("provide either data (base64) or source_path"), nil
	}

	var res *attach.Result
	var dataSize int64

	if sourcePath != "" {
		if filename == "" {
			filename = filepath.Base(sourcePath)
		}
		ext := strings.ToLower(filepath.Ext(filename))
		testRel := attach.RelPath(project, "test"+ext)
		if _, errRes := s.authorizeWrite(ctx, testRel); errRes != nil {
			return errRes, nil
		}
		res, err = attach.StoreFromPath(s.vault, sourcePath, filename, project, s.effectiveUploadRoots())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if abs, absErr := s.vault.Abs(res.Path); absErr == nil {
			if fi, stErr := os.Stat(abs); stErr == nil {
				dataSize = fi.Size()
			}
		}
	} else {
		if filename == "" {
			return mcp.NewToolResultError("filename is required when using data (base64)"), nil
		}
		data, err := base64.StdEncoding.DecodeString(dataB64)
		if err != nil {
			return mcp.NewToolResultError("invalid base64: " + err.Error()), nil
		}
		ext := strings.ToLower(filepath.Ext(filename))
		testRel := attach.RelPath(project, "test"+ext)
		tok, errRes := s.authorizeWrite(ctx, testRel)
		if errRes != nil {
			return errRes, nil
		}
		if errRes := s.checkWriteLimits(tok, len(data)); errRes != nil {
			return errRes, nil
		}
		res, err = attach.Store(s.vault, data, filename, project)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dataSize = int64(len(data))
	}

	s.auditWrite(ctx, audit.ActionUploadAttachment, res.Path, "", dataSize)

	// Resolve kind and MIME from the validated extension (always reliable
	// since attach.Store rejects unknown extensions before reaching here).
	ext := strings.ToLower(filepath.Ext(res.Filename))
	mime := "application/octet-stream"
	resolvedKind := "document"
	if info, ok := attach.AllowedExt[ext]; ok {
		mime = info.MIME
		if info.IsImage {
			resolvedKind = "image"
		}
	}

	return mcp.NewToolResultJSON(map[string]any{
		"path":              res.Path,
		"url":               "/vault-files/" + res.Path,
		"mime":              mime,
		"kind":              resolvedKind,
		"size":              dataSize,
		"original_filename": filename,
		"hash":              strings.TrimSuffix(res.Filename, ext),
	})
}

// allowedExtSet builds a set of allowed extensions for vault.ListAttachments.
func allowedExtSet() map[string]bool {
	m := make(map[string]bool, len(attach.AllowedExt))
	for ext := range attach.AllowedExt {
		m[ext] = true
	}
	return m
}

func (s *Server) handleListAttachments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tok, errRes := s.authorizeRead(ctx)
	if errRes != nil {
		return errRes, nil
	}
	project := req.GetString("project", "")
	if tok.ProjectFilter() != "" {
		if project != "" && project != tok.ProjectFilter() {
			return mcp.NewToolResultErrorf("project %q is outside the token's scope %q", project, tok.ProjectFilter()), nil
		}
		project = tok.ProjectFilter()
	}

	infos, err := s.vault.ListAttachments(project, allowedExtSet())
	if err != nil {
		return mcp.NewToolResultErrorFromErr("list attachments failed", err), nil
	}

	type attachEntry struct {
		Path  string `json:"path"`
		Size  int64  `json:"size"`
		MIME  string `json:"mime"`
		Mtime int64  `json:"mtime"`
	}
	out := make([]attachEntry, 0, len(infos))
	for _, info := range infos {
		if !tok.AllowsPath(info.Path) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(info.Path))
		mime := "application/octet-stream"
		if ei, ok := attach.AllowedExt[ext]; ok {
			mime = ei.MIME
		}
		out = append(out, attachEntry{
			Path:  info.Path,
			Size:  info.Size,
			MIME:  mime,
			Mtime: info.ModTime.Unix(),
		})
	}
	return mcp.NewToolResultJSON(map[string]any{"attachments": out})
}

func (s *Server) handleDeleteAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rel, err := s.vault.Rel(path)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid path", err), nil
	}
	// Only allow deleting files inside attachments/ directories.
	if !strings.Contains("/"+rel, "/attachments/") {
		return mcp.NewToolResultError("path is not inside an attachments/ directory"), nil
	}
	if _, errRes := s.authorizeWrite(ctx, rel); errRes != nil {
		return errRes, nil
	}
	if !s.vault.Exists(rel) {
		return mcp.NewToolResultErrorf("attachment %q not found", rel), nil
	}
	if err := s.vault.Delete(rel); err != nil {
		return mcp.NewToolResultErrorFromErr("delete failed", err), nil
	}
	s.auditWrite(ctx, audit.ActionDeleteAttachment, rel, "", 0)
	return mcp.NewToolResultJSON(map[string]any{"deleted": true, "path": rel})
}

func (s *Server) handleAttachmentInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tok, errRes := s.authorizeRead(ctx)
	if errRes != nil {
		return errRes, nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	rel, err := s.vault.Rel(path)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("invalid path", err), nil
	}
	if !strings.Contains("/"+rel, "/attachments/") {
		return mcp.NewToolResultError("path is not inside an attachments/ directory"), nil
	}
	if !tok.AllowsPath(rel) {
		return mcp.NewToolResultErrorf("path %q is outside the token's scope", rel), nil
	}

	abs, err := s.vault.Abs(rel)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("resolve path", err), nil
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return mcp.NewToolResultErrorf("attachment %q not found", rel), nil
	}

	ext := strings.ToLower(filepath.Ext(rel))
	mime := "application/octet-stream"
	if ei, ok := attach.AllowedExt[ext]; ok {
		mime = ei.MIME
	}

	// Find notes that reference this attachment by searching the index for the
	// hashed filename (the unique part of the path).
	basename := filepath.Base(rel)
	var referencedBy []string
	if hits, err := s.index.Search(basename, 100); err == nil {
		for _, h := range hits {
			if !tok.AllowsPath(h.Path) {
				continue
			}
			// Verify the note actually contains a reference to this attachment
			// (not just a coincidental substring match on the hash).
			if note, loadErr := s.vault.Load(h.Path); loadErr == nil {
				if strings.Contains(string(note.Content), basename) {
					referencedBy = append(referencedBy, h.Path)
				}
			}
		}
	}

	return mcp.NewToolResultJSON(map[string]any{
		"path":          rel,
		"size":          fi.Size(),
		"mime":          mime,
		"mtime":         fi.ModTime().Unix(),
		"referenced_by": referencedBy,
	})
}
