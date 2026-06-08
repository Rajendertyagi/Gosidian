/**
 * Window-manager store — niri-style scrollable tiling "plancia".
 *
 * Ported from products-dc (`admin/stores/windows.ts`) and kept type-agnostic:
 * the store does not know what a window contains — that is decided by the
 * type→component `registry` passed to WindowManager. Reusable for any
 * "aggregate". The store stays pure: no router here (URL sync lives in the
 * usePlanciaSync composable so the store is unit-testable in isolation).
 */
import { defineStore } from 'pinia'

export type WindowWidth = 's' | 'm' | 'full'

/**
 * Header tag (reusable). Communicates "what this window is associated with".
 * When `open` is set the tag is clickable and opens that window (via the
 * injected `openWindow`) — provided its type is registered in the host
 * plancia, otherwise the tag stays informational (non-clickable).
 */
export interface WindowTag {
  label: string
  /** Tone key for the colour (usually the module type). Resolved via
   *  windowModuleTone with a neutral fallback. */
  tone?: string
  /** Window to open on click. Absent → purely informational tag. */
  open?: OpenSpec
}

export interface WindowInstance {
  id: string
  type: string
  /** Logical de-dup key (e.g. "note:gosidian/hot.md", "settings"). */
  key: string
  title: string
  props: Record<string, unknown>
  /** Header tags (associations), updated by the content via `tags` emit. */
  tags: WindowTag[]
  width: WindowWidth
  minimized: boolean
  dirty: boolean
}

export interface OpenSpec {
  type: string
  key?: string
  title?: string
  props?: Record<string, unknown>
  tags?: WindowTag[]
  width?: WindowWidth
  /** If false, opens (right of focus) WITHOUT moving focus. Default true. */
  focus?: boolean
}

interface WindowsState {
  windows: WindowInstance[]
  focusedId: string | null
  seq: number
  /** Bumped on every content-signalled data change (`changed` emit); dependent
   *  views can watch it to refresh. */
  dataVersion: number
}

/** Width cycle order (the ⤢ header button). Smallest → medium → full (req #4). */
const WIDTH_CYCLE: WindowWidth[] = ['s', 'm', 'full']

export const useWindowsStore = defineStore('windows', {
  state: (): WindowsState => ({ windows: [], focusedId: null, seq: 0, dataVersion: 0 }),
  getters: {
    visible: (s): WindowInstance[] => s.windows.filter((w) => !w.minimized),
    minimizedList: (s): WindowInstance[] => s.windows.filter((w) => w.minimized),
    focused: (s): WindowInstance | null =>
      s.windows.find((w) => w.id === s.focusedId) ?? null,
  },
  actions: {
    _byId(id: string): WindowInstance | undefined {
      return this.windows.find((w) => w.id === id)
    },
    /**
     * Open (or, if the `key` already exists, focus/restore) a window.
     * The new window is inserted immediately to the right of the focused one.
     */
    open(spec: OpenSpec): string {
      const wantFocus = spec.focus !== false
      const key = spec.key ?? `${spec.type}:auto:${this.seq + 1}`
      const existing = this.windows.find((w) => w.key === key)
      if (existing) {
        existing.minimized = false
        if (wantFocus) this.focusedId = existing.id
        return existing.id
      }
      this.seq += 1
      const id = `win-${this.seq}`
      const win: WindowInstance = {
        id,
        type: spec.type,
        key,
        title: spec.title ?? '',
        props: spec.props ?? {},
        tags: spec.tags ?? [],
        width: spec.width ?? 'm',
        minimized: false,
        dirty: false,
      }
      const fi = this.windows.findIndex((w) => w.id === this.focusedId)
      if (fi >= 0) this.windows.splice(fi + 1, 0, win)
      else this.windows.push(win)
      if (wantFocus) this.focusedId = id
      return id
    },
    close(id: string): void {
      const i = this.windows.findIndex((w) => w.id === id)
      if (i < 0) return
      this.windows.splice(i, 1)
      if (this.focusedId === id) {
        const next = this.windows[i] ?? this.windows[i - 1] ?? null
        this.focusedId = next ? next.id : null
      }
    },
    minimize(id: string): void {
      const w = this._byId(id)
      if (!w) return
      w.minimized = true
      if (this.focusedId === id) {
        const vis = this.windows.filter((x) => !x.minimized)
        const last = vis[vis.length - 1]
        this.focusedId = last ? last.id : null
      }
    },
    restore(id: string): void {
      const w = this._byId(id)
      if (!w) return
      w.minimized = false
      this.focusedId = id
    },
    focus(id: string): void {
      if (this._byId(id)) this.focusedId = id
    },
    /** Move focus to the adjacent visible window (-1 left, +1 right). */
    focusAdjacent(dir: -1 | 1): void {
      const vis = this.windows.filter((w) => !w.minimized)
      if (!vis.length) return
      const ci = vis.findIndex((w) => w.id === this.focusedId)
      const base = ci < 0 ? 0 : ci
      const ni = Math.min(Math.max(base + dir, 0), vis.length - 1)
      const target = vis[ni]
      if (target) this.focusedId = target.id
    },
    cycleWidth(id: string): void {
      const w = this._byId(id)
      if (!w) return
      const idx = WIDTH_CYCLE.indexOf(w.width)
      w.width = WIDTH_CYCLE[(idx + 1) % WIDTH_CYCLE.length] ?? 'm'
    },
    setTitle(id: string, title: string): void {
      const w = this._byId(id)
      if (w) w.title = title
    },
    setDirty(id: string, dirty: boolean): void {
      const w = this._byId(id)
      if (w) w.dirty = dirty
    },
    setTags(id: string, tags: WindowTag[]): void {
      const w = this._byId(id)
      if (w) w.tags = tags ?? []
    },
    /** Assign a stable key/props to a window (e.g. after a "new" note has been
     *  saved and obtained its path) → correct de-dup and URL sync. */
    identify(id: string, key: string, props: Record<string, unknown>): void {
      const w = this._byId(id)
      if (!w) return
      w.key = key
      Object.assign(w.props, props)
    },
    /** Signal that data changed (dependent views reload). */
    touch(): void {
      this.dataVersion += 1
    },
    reset(): void {
      this.windows = []
      this.focusedId = null
    },
  },
})
