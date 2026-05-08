package v1

import (
	"net/http"
	"strings"
	"time"

	"github.com/gosidian/gosidian/internal/audit"
	"github.com/gosidian/gosidian/internal/config"
)

// settingsView is the JSON shape the SPA reads. Sensitive fields
// (git push tokens, bcrypt-hashed user records, MCP token plaintexts)
// are NEVER surfaced — the SPA gets enough to render the settings
// page and the gitsync status badge, nothing more.
//
// The shape is intentionally a subset of internal/config.Config: we
// reserve the right to evolve config.toml independently and add a
// translation layer here when fields diverge. Today the mapping is
// 1:1 minus secrets.
type settingsView struct {
	Git   gitSettings   `json:"git"`
	Trash trashSettings `json:"trash"`
	I18n  i18nSettings  `json:"i18n"`
	MCP   mcpSettings   `json:"mcp"`
}

type gitSettings struct {
	Enabled     bool   `json:"enabled"`
	Remote      string `json:"remote"`
	Branch      string `json:"branch"`
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	DebounceMS  int64  `json:"debounce_ms"`
	Push        bool   `json:"push"`
	TokenEnv    string `json:"token_env"` // env var name only; never the value
}

type trashSettings struct {
	Enabled     bool  `json:"enabled"`
	RetentionMS int64 `json:"retention_ms"`
}

type i18nSettings struct {
	DefaultLang  string   `json:"default_lang"`
	EnabledLangs []string `json:"enabled_langs"`
}

type mcpSettings struct {
	WritePerMinute int   `json:"write_per_minute"`
	MaxNoteBytes   int64 `json:"max_note_bytes"`
}

// updateSettingsRequest covers PATCH-style updates. Pointer fields
// distinguish "not present" (no change) from "explicitly cleared".
// Only owners may PUT.
type updateSettingsRequest struct {
	Git *struct {
		Enabled     *bool   `json:"enabled,omitempty"`
		Remote      *string `json:"remote,omitempty"`
		Branch      *string `json:"branch,omitempty"`
		AuthorName  *string `json:"author_name,omitempty"`
		AuthorEmail *string `json:"author_email,omitempty"`
		DebounceMS  *int64  `json:"debounce_ms,omitempty"`
		Push        *bool   `json:"push,omitempty"`
		TokenEnv    *string `json:"token_env,omitempty"`
	} `json:"git,omitempty"`
	Trash *struct {
		Enabled     *bool  `json:"enabled,omitempty"`
		RetentionMS *int64 `json:"retention_ms,omitempty"`
	} `json:"trash,omitempty"`
	I18n *struct {
		DefaultLang  *string  `json:"default_lang,omitempty"`
		EnabledLangs []string `json:"enabled_langs,omitempty"`
	} `json:"i18n,omitempty"`
	MCP *struct {
		WritePerMinute *int   `json:"write_per_minute,omitempty"`
		MaxNoteBytes   *int64 `json:"max_note_bytes,omitempty"`
	} `json:"mcp,omitempty"`
}

func (r *Router) handleSettings(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.getSettings(w, req)
	case http.MethodPut:
		r.putSettings(w, req)
	default:
		WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
	}
}

// getSettings is readable by any authenticated user — the SPA needs
// it to render the settings page even for member roles. Sensitive
// fields are stripped above.
func (r *Router) getSettings(w http.ResponseWriter, _ *http.Request) {
	cfg, err := r.loadConfig()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, toSettingsView(cfg))
}

// putSettings is owner-only. It loads the current config, applies
// the patch, validates, and saves atomically. A failed validation
// leaves the on-disk file untouched.
func (r *Router) putSettings(w http.ResponseWriter, req *http.Request) {
	user := UserFromContext(req.Context())
	if user == nil {
		WriteError(w, http.StatusUnauthorized, CodeAuthTokenInvalid, "no user in context")
		return
	}
	if user.Role != "owner" {
		WriteError(w, http.StatusForbidden, CodeAuthOwnerOnly, "owner role required to update settings")
		return
	}
	if r.deps.ConfigPath == "" {
		WriteError(w, http.StatusServiceUnavailable, CodeServerUnavailable, "config path not wired")
		return
	}

	var body updateSettingsRequest
	if err := DecodeJSON(req, &body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, err.Error())
		return
	}

	cfg, err := r.loadConfig()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, err.Error())
		return
	}
	if errMsg := applySettingsPatch(cfg, &body); errMsg != "" {
		WriteError(w, http.StatusBadRequest, CodeValidationFormat, errMsg)
		return
	}

	if err := config.Save(r.deps.ConfigPath, cfg); err != nil {
		WriteError(w, http.StatusInternalServerError, CodeServerInternal, "save: "+err.Error())
		return
	}

	if r.deps.Audit != nil {
		_ = r.deps.Audit.Write(audit.Entry{
			Source: audit.SourceHTTP,
			Actor:  user.Username,
			UserID: user.ID,
			Action: "settings_update",
			Path:   r.deps.ConfigPath,
		})
	}

	WriteJSON(w, http.StatusOK, toSettingsView(cfg))
}

func (r *Router) loadConfig() (*config.Config, error) {
	if r.deps.ConfigPath == "" {
		return config.Default(), nil
	}
	return config.Load(r.deps.ConfigPath)
}

func toSettingsView(c *config.Config) settingsView {
	return settingsView{
		Git: gitSettings{
			Enabled:     c.Git.Enabled,
			Remote:      c.Git.Remote,
			Branch:      c.Git.Branch,
			AuthorName:  c.Git.AuthorName,
			AuthorEmail: c.Git.AuthorEmail,
			DebounceMS:  c.Git.Debounce.Milliseconds(),
			Push:        c.Git.Push,
			TokenEnv:    c.Git.TokenEnv,
		},
		Trash: trashSettings{
			Enabled:     c.Trash.Enabled,
			RetentionMS: c.Trash.Retention.Milliseconds(),
		},
		I18n: i18nSettings{
			DefaultLang:  c.I18n.DefaultLang,
			EnabledLangs: append([]string(nil), c.I18n.EnabledLangs...),
		},
		MCP: mcpSettings{
			WritePerMinute: c.MCP.WritePerMinute,
			MaxNoteBytes:   c.MCP.MaxNoteBytes,
		},
	}
}

// applySettingsPatch merges the request body into cfg, validating
// each field as it lands. Returns a non-empty error string on the
// first failure; cfg state may be partially mutated on early-exit
// but the caller hasn't called Save yet so disk is untouched.
func applySettingsPatch(cfg *config.Config, body *updateSettingsRequest) string {
	if body == nil {
		return ""
	}
	if body.Git != nil {
		g := body.Git
		if g.Enabled != nil {
			cfg.Git.Enabled = *g.Enabled
		}
		if g.Remote != nil {
			cfg.Git.Remote = strings.TrimSpace(*g.Remote)
		}
		if g.Branch != nil {
			cfg.Git.Branch = strings.TrimSpace(*g.Branch)
		}
		if g.AuthorName != nil {
			cfg.Git.AuthorName = strings.TrimSpace(*g.AuthorName)
		}
		if g.AuthorEmail != nil {
			cfg.Git.AuthorEmail = strings.TrimSpace(*g.AuthorEmail)
		}
		if g.DebounceMS != nil {
			if *g.DebounceMS < 1000 {
				return "git.debounce_ms must be at least 1000"
			}
			cfg.Git.Debounce = time.Duration(*g.DebounceMS) * time.Millisecond
		}
		if g.Push != nil {
			cfg.Git.Push = *g.Push
		}
		if g.TokenEnv != nil {
			cfg.Git.TokenEnv = strings.TrimSpace(*g.TokenEnv)
		}
		if cfg.Git.Enabled && cfg.Git.Push && cfg.Git.Remote == "" {
			return "git push enabled but remote is empty"
		}
	}
	if body.Trash != nil {
		if body.Trash.Enabled != nil {
			cfg.Trash.Enabled = *body.Trash.Enabled
		}
		if body.Trash.RetentionMS != nil {
			if *body.Trash.RetentionMS < 0 {
				return "trash.retention_ms cannot be negative"
			}
			cfg.Trash.Retention = time.Duration(*body.Trash.RetentionMS) * time.Millisecond
		}
	}
	if body.I18n != nil {
		if body.I18n.DefaultLang != nil {
			d := strings.TrimSpace(*body.I18n.DefaultLang)
			if d == "" {
				return "i18n.default_lang cannot be empty"
			}
			cfg.I18n.DefaultLang = d
		}
		if body.I18n.EnabledLangs != nil {
			cfg.I18n.EnabledLangs = append([]string(nil), body.I18n.EnabledLangs...)
		}
	}
	if body.MCP != nil {
		if body.MCP.WritePerMinute != nil {
			if *body.MCP.WritePerMinute < 0 {
				return "mcp.write_per_minute cannot be negative"
			}
			cfg.MCP.WritePerMinute = *body.MCP.WritePerMinute
		}
		if body.MCP.MaxNoteBytes != nil {
			if *body.MCP.MaxNoteBytes < 0 {
				return "mcp.max_note_bytes cannot be negative"
			}
			cfg.MCP.MaxNoteBytes = *body.MCP.MaxNoteBytes
		}
	}
	return ""
}
