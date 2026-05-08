// Package projects persists per-project flags (skip git sync, hidden from
// MCP) in <vault>/.gosidian/projects.json. The store is concurrent-safe and
// reloads transparently when the underlying file's mtime changes, so flags
// written from the CLI or another process become effective without a
// restart (mirrors the pattern of internal/auth.Store).
//
// Default behaviour: a project without an entry yields zero-value Flags
// (false/false), preserving the current "everything in, everything visible"
// invariant for projects that pre-existed this feature.
package projects

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Flags are the configurable per-project knobs. JSON keys use snake_case
// for human-edited file ergonomics.
type Flags struct {
	SkipGitSync   bool `json:"skip_git_sync,omitempty"`
	HiddenFromMCP bool `json:"hidden_from_mcp,omitempty"`
}

// Entry is a (name, flags) pair returned by All().
type Entry struct {
	Name string
	Flags
}

// Store is a concurrent-safe per-project flags store backed by a JSON file.
// Like auth.Store, it re-reads the file when its mtime changes so out-of-band
// edits become effective without a restart.
type Store struct {
	path  string
	mu    sync.RWMutex
	data  map[string]Flags
	mtime time.Time
}

type storeFile struct {
	Projects map[string]Flags `json:"projects"`
}

// Open loads the store from the given file path. A missing file is not an
// error — it returns an empty store, and the file is created lazily on the
// first Set/Delete/Rename.
func Open(path string) (*Store, error) {
	s := &Store{path: path, data: map[string]Flags{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path returns the on-disk path the store reads/writes.
func (s *Store) Path() string { return s.path }

// load replaces the in-memory snapshot with what's on disk. Caller must hold
// s.mu in write mode or be in an initialization context.
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.data = map[string]Flags{}
			s.mtime = time.Time{}
			return nil
		}
		return err
	}
	var sf storeFile
	if len(data) > 0 {
		if err := json.Unmarshal(data, &sf); err != nil {
			return fmt.Errorf("parse projects file: %w", err)
		}
	}
	if sf.Projects == nil {
		sf.Projects = map[string]Flags{}
	}
	s.data = sf.Projects
	if st, err := os.Stat(s.path); err == nil {
		s.mtime = st.ModTime()
	}
	return nil
}

// reloadIfStale re-reads the file when its mtime (or existence) diverges from
// the last-loaded snapshot. Caller must hold s.mu in write mode.
func (s *Store) reloadIfStale() {
	st, err := os.Stat(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !s.mtime.IsZero() || len(s.data) > 0 {
				s.data = map[string]Flags{}
				s.mtime = time.Time{}
			}
		}
		return
	}
	if st.ModTime().Equal(s.mtime) {
		return
	}
	_ = s.load()
}

// save writes the current in-memory snapshot atomically (write+rename, 0o600).
// Caller must hold s.mu in write mode.
func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(storeFile{Projects: s.data}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if st, err := os.Stat(s.path); err == nil {
		s.mtime = st.ModTime()
	}
	return nil
}

// Get returns the flags for a project. Unknown projects yield zero-value
// flags (backward-compatible default).
func (s *Store) Get(name string) Flags {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	return s.data[name]
}

// Set persists the flags for a project. If both fields are zero the entry is
// removed instead, keeping projects.json minimal.
func (s *Store) Set(name string, f Flags) error {
	if name == "" || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid project name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	if f == (Flags{}) {
		delete(s.data, name)
	} else {
		s.data[name] = f
	}
	return s.save()
}

// Delete removes any entry for the project. No-op if absent.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	if _, ok := s.data[name]; !ok {
		return nil
	}
	delete(s.data, name)
	return s.save()
}

// Rename atomically moves an entry from oldName to newName. No-op if oldName
// has no entry. If newName already has an entry it's overwritten.
func (s *Store) Rename(oldName, newName string) error {
	if newName == "" || strings.ContainsAny(newName, "/\\") {
		return fmt.Errorf("invalid project name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	f, ok := s.data[oldName]
	if !ok {
		return nil
	}
	delete(s.data, oldName)
	s.data[newName] = f
	return s.save()
}

// All returns every entry, sorted by Name. Stable ordering is convenient for
// UI rendering and deterministic tests.
func (s *Store) All() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	out := make([]Entry, 0, len(s.data))
	for n, f := range s.data {
		out = append(out, Entry{Name: n, Flags: f})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SkipNamesForGit returns the set of project names with SkipGitSync=true,
// sorted. Used by gitsync to render the managed block of .gitignore.
func (s *Store) SkipNamesForGit() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadIfStale()
	out := make([]string, 0)
	for n, f := range s.data {
		if f.SkipGitSync {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
