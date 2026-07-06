package vault

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gosidian/gosidian/internal/attach"
	"github.com/gosidian/gosidian/internal/parser"
)

// MediaRef is the resolved payload of a media-style note: a markdown note
// whose frontmatter declares a payload kind (`type: image`, ADR-013, or
// `type: table`, ADR-016) and a `media:` pointer to the attachment carrying
// the bytes. The note itself stays a plain .md — body is the searchable
// caption/transcript — so all the note machinery (index, FTS, tags, links,
// graph) is untouched; this struct is an overlay the read paths surface so
// the SPA never has to parse frontmatter.
type MediaRef struct {
	Path   string `json:"path"`             // vault-relative attachment path (the resolved `media:` value)
	URL    string `json:"url"`              // serving URL, e.g. /vault-files/<path>
	MIME   string `json:"mime,omitempty"`   // image MIME type; empty when broken
	Size   int64  `json:"size,omitempty"`   // bytes; 0 when broken
	Broken bool   `json:"broken,omitempty"` // the pointer is missing/invalid/not a readable image
}

// ResolveAttachmentByName resolves an Obsidian image-embed target (`![[X]]`)
// to a vault-relative attachment path. A target carrying a slash is treated as
// a vault-relative path; a bare filename is searched for in the vault-root
// attachments/ dir and in each project's attachments/. Returns the path and
// true on the first hit. Used by the renderer's image-embed support.
func (v *Vault) ResolveAttachmentByName(name string) (string, bool) {
	name = strings.TrimSpace(filepath.ToSlash(name))
	if name == "" {
		return "", false
	}
	if strings.Contains(name, "/") {
		if rel, err := v.Rel(name); err == nil && v.Exists(rel) {
			return rel, true
		}
		return "", false
	}
	candidates := []string{"attachments/" + name}
	if projs, err := v.Projects(); err == nil {
		for _, p := range projs {
			candidates = append(candidates, p.Name+"/attachments/"+name)
		}
	}
	for _, c := range candidates {
		if v.Exists(c) {
			return c, true
		}
	}
	return "", false
}

// MediaRefForNote inspects a note's frontmatter and, when the matching feature
// flag is enabled AND the note declares a payload kind (`type: image` with
// media_notes on, ADR-013; `type: table` with table_notes on, ADR-016),
// returns the resolved payload reference plus the kind. An empty kind means
// the note is ordinary markdown. The kind is reported even for a broken
// pointer, so callers can still tag the note and the SPA renders a
// placeholder.
//
// Resolution failures (missing `media:` key, path escape, wrong extension for
// the kind, file absent) are NOT errors: the ref is returned with Broken=true
// so the read never fails. With the flag off the note is treated as ordinary
// markdown, which keeps each feature fully backward-compatible.
func (v *Vault) MediaRefForNote(rel string, content []byte) (*MediaRef, string) {
	raw := parser.FrontmatterRawForPath(rel, content)
	if raw == "" {
		return nil, ""
	}
	fields := parser.ParseFrontmatterFields(raw)
	declared, _ := fields["type"].(string)
	var kind string
	var extOK func(ext string) (mime string, ok bool)
	switch strings.ToLower(strings.TrimSpace(declared)) {
	case "image":
		if !v.mediaNotes {
			return nil, ""
		}
		kind = "image"
		extOK = func(ext string) (string, bool) {
			mime, isImage, err := attach.ValidateExt(ext)
			return mime, err == nil && isImage
		}
	case "table":
		if !v.tableNotes {
			return nil, ""
		}
		kind = "table"
		extOK = func(ext string) (string, bool) {
			mime, _, err := attach.ValidateExt(ext)
			return mime, err == nil && ext == ".csv"
		}
	default:
		return nil, ""
	}

	mediaPath, _ := fields["media"].(string)
	mediaPath = strings.TrimSpace(mediaPath)
	if mediaPath == "" {
		return &MediaRef{Broken: true}, kind
	}

	// Sanitize the pointer (rejects ".." escapes) and validate it points at a
	// readable attachment of the right type inside the vault.
	clean, err := v.Rel(mediaPath)
	if err != nil {
		return &MediaRef{Path: mediaPath, Broken: true}, kind
	}
	ref := &MediaRef{Path: clean, URL: "/vault-files/" + clean}
	mime, ok := extOK(strings.ToLower(filepath.Ext(clean)))
	if !ok {
		ref.Broken = true
		return ref, kind
	}
	fi, err := os.Stat(filepath.Join(v.Root, filepath.FromSlash(clean)))
	if err != nil || fi.IsDir() {
		ref.Broken = true
		return ref, kind
	}
	ref.MIME = mime
	ref.Size = fi.Size()
	return ref, kind
}
