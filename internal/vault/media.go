package vault

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gosidian/gosidian/internal/attach"
	"github.com/gosidian/gosidian/internal/parser"
)

// MediaRef is the resolved image payload of a media note: a markdown note whose
// frontmatter declares `type: image` and a `media:` pointer to an image
// attachment (ADR-013, Framing B). The note itself stays a plain .md — body is
// the searchable caption/transcript — so all the note machinery (index, FTS,
// tags, links, graph) is untouched; this struct is an overlay the read paths
// surface so the SPA never has to parse frontmatter.
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

// MediaRefForNote inspects a note's frontmatter and, when media notes are
// enabled AND the note declares `type: image`, returns the resolved image
// reference plus true. The second return reports whether the note is a media
// note at all (independent of whether the pointer resolves), so callers can set
// kind="image" even for a broken pointer.
//
// Resolution failures (missing `media:` key, path escape, non-image extension,
// file absent) are NOT errors: the ref is returned with Broken=true so the SPA
// renders a placeholder instead of the read failing. When media notes are
// disabled the note is treated as ordinary markdown (returns nil, false), which
// keeps the feature fully backward-compatible behind its flag.
func (v *Vault) MediaRefForNote(rel string, content []byte) (*MediaRef, bool) {
	if !v.mediaNotes {
		return nil, false
	}
	raw := parser.FrontmatterRawForPath(rel, content)
	if raw == "" {
		return nil, false
	}
	fields := parser.ParseFrontmatterFields(raw)
	kind, _ := fields["type"].(string)
	if !strings.EqualFold(strings.TrimSpace(kind), "image") {
		return nil, false
	}

	mediaPath, _ := fields["media"].(string)
	mediaPath = strings.TrimSpace(mediaPath)
	if mediaPath == "" {
		return &MediaRef{Broken: true}, true
	}

	// Sanitize the pointer (rejects ".." escapes) and validate it points at a
	// readable image inside the vault.
	clean, err := v.Rel(mediaPath)
	if err != nil {
		return &MediaRef{Path: mediaPath, Broken: true}, true
	}
	ref := &MediaRef{Path: clean, URL: "/vault-files/" + clean}
	mime, isImage, err := attach.ValidateExt(strings.ToLower(filepath.Ext(clean)))
	if err != nil || !isImage {
		ref.Broken = true
		return ref, true
	}
	fi, err := os.Stat(filepath.Join(v.Root, filepath.FromSlash(clean)))
	if err != nil || fi.IsDir() {
		ref.Broken = true
		return ref, true
	}
	ref.MIME = mime
	ref.Size = fi.Size()
	return ref, true
}
