# Editor

A note opens as a **plancia window** with a **View / Edit** toggle in
the window header. It defaults to *view* (rendered preview); flipping
to *edit* mounts the editor in the same window — no second window. The
editor itself is **CodeMirror 6**, loaded lazily so a window that's
only ever read never pulls the editor chunk. No rich-text, no WYSIWYG
gymnastics — the editor is intentionally thin because the vault is the
source of truth and many users edit notes from Obsidian, a terminal
editor, or another markdown tool on the filesystem.

## Modes

While in *edit*, a toolbar above the editor toggles four layout modes:

- **Editor** — full-width textarea, no preview
- **Split** — editor + preview side-by-side
- **Stacked** — editor on top, preview below (good on narrow displays)
- **Preview** — rendered note only; no editing affordance

Mode choice is persisted in local storage per browser.

## Live preview

The preview uses the same goldmark pipeline as the MCP rendering path.
Wiki-links `[[target]]` are resolved and linked; unresolved targets are
rendered with a `broken` style so authors notice typos immediately.

## Save / history

- **Save** writes the note atomically (temp file + rename) and updates
  the SQLite index synchronously.
- **History** (`/notes/<path>/history`) lists previous states captured
  by git sync when enabled — with a one-click `restore` action.

## Print / Save as PDF

In **view mode**, a **Print** button appears in the note's header — but
**only for Markdown notes** (`.html` notes are not printable; the
sandboxed iframe the browser clips to a single page is out of reach —
see IMP-053). Clicking it opens the browser's native print dialog, from
which you typically pick *Save as PDF*.

The print stylesheet (`@media print`) shows **only this note's rendered
article** and hides the rest of the plancia (topbar, sidebar, other
windows) plus the browser's own header/footer chrome, so a single,
clean note reaches the page. The button calls `printNote()`, which
tags the article with the `gosidian-print-target` class, fires
`window.print()`, and removes the class again on `afterprint`.

## Maximize / window controls

There is no dedicated "focus mode" anymore — to give a note the full
screen, use the plancia window's **maximize** control. See
[Overview → window controls](overview.md) for the full set of window
affordances (maximize, minimize, close, drag, resize).

## Command palette

`Cmd+K` / `Ctrl+K` opens a fuzzy finder across notes, projects, tags,
and built-in actions (go to graph, create note, …). Keyboard-first:
arrow keys to select, `Enter` to run, `Esc` to close. Recent
selections are remembered for quick re-access.

## Not supported in-editor

Deliberately excluded from the web editor because they belong
elsewhere:

- **Rich text formatting toolbar** — the markdown source is the
  source of truth; a toolbar would hide it
- **Collaborative editing** — out of scope; use git sync for
  multi-writer workflows
- **Drag-drop reordering in outline panel** — the outline is a read-
  only view of the heading structure

For heavy-duty editing (long-form writing, multi-cursor operations,
vim bindings) open the vault in Obsidian, VS Code, or your editor of
choice and edit on disk. The watcher picks up external changes and
reindexes in real time — see
[Vault format → Source of truth](../vault/format.md#source-of-truth).
