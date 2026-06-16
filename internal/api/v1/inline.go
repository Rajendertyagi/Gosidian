package v1

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gosidian/gosidian/internal/attach"
	"github.com/gosidian/gosidian/internal/authz"
)

var (
	// `![[target]]` / `![[target|alias]]` Obsidian image embed.
	mdEmbedRe = regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	// `![alt](/vault-files/...)` standard markdown image at a vault URL.
	mdVaultImgRe = regexp.MustCompile(`(!\[[^\]]*\]\()(/vault-files/[^)\s]+)(\))`)
	// `<img ... src="/vault-files/...">` in an HTML note.
	htmlVaultImgRe = regexp.MustCompile(`(<img\b[^>]*?\bsrc\s*=\s*)(["'])(/vault-files/[^"']+)(["'])`)
)

// inlineImages rewrites a note's image references into data: URIs so a
// downloaded file is self-contained, WITHOUT touching the stored note (which
// keeps the lightweight reference for MCP reads and storage — token savings).
// Markdown `![[X]]` and `![](/vault-files/Y)` become `![](data:...)`; HTML
// `<img src="/vault-files/Z">` becomes `src="data:..."`. Unresolvable refs are
// left untouched.
func (r *Router) inlineImages(content, format string, p authz.Principal) string {
	if r.deps.Vault == nil {
		return content
	}
	if format == "html" {
		return htmlVaultImgRe.ReplaceAllStringFunc(content, func(m string) string {
			sub := htmlVaultImgRe.FindStringSubmatch(m)
			if d := r.vaultURLToDataURI(sub[3]); d != "" {
				return sub[1] + sub[2] + d + sub[4]
			}
			return m
		})
	}
	// Markdown: resolve ![[embed]] first (needs vault/index resolution), then
	// plain ![](/vault-files/...) URLs.
	res := previewResolver{r: r, p: p}
	content = mdEmbedRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := mdEmbedRe.FindStringSubmatch(m)
		target, alias := splitEmbed(sub[1])
		if u := res.ResolveImage(target); u != "" {
			if d := r.vaultURLToDataURI(u); d != "" {
				return "![" + alias + "](" + d + ")"
			}
		}
		return m
	})
	content = mdVaultImgRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := mdVaultImgRe.FindStringSubmatch(m)
		if d := r.vaultURLToDataURI(sub[2]); d != "" {
			return sub[1] + d + sub[3]
		}
		return m
	})
	return content
}

// vaultURLToDataURI reads the attachment behind a /vault-files/<rel> URL and
// returns its data: URI, or "" on any failure.
func (r *Router) vaultURLToDataURI(url string) string {
	rel := strings.TrimPrefix(url, "/vault-files/")
	clean, err := r.deps.Vault.Rel(rel)
	if err != nil {
		return ""
	}
	abs, err := r.deps.Vault.Abs(clean)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	return attach.DataURI(data, filepath.Ext(clean))
}

func splitEmbed(inner string) (target, alias string) {
	parts := strings.SplitN(inner, "|", 2)
	target = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		alias = strings.TrimSpace(parts[1])
	}
	return target, alias
}
