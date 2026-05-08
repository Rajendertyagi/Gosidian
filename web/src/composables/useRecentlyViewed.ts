/**
 * Recently viewed notes — client-side stack persisted in
 * localStorage. The home page surfaces it as a "Pick up where you
 * left off" widget; the command palette (Phase 3.2) uses it to
 * prioritise recents in the fuzzy ranking.
 *
 * Kept separate from the tree store because the entry never lives
 * server-side: it's purely a UX affordance keyed to the device.
 */
import { ref, watch } from 'vue'

export interface RecentEntry {
  path: string
  title: string
  ts: number
}

const STORAGE_KEY = 'gosidian.recent'
const MAX_ENTRIES = 10

function load(): RecentEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as RecentEntry[]
    return Array.isArray(parsed) ? parsed.filter((e) => e && typeof e.path === 'string') : []
  } catch {
    return []
  }
}

const entries = ref<RecentEntry[]>(load())
let initialised = false

function persist() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(entries.value))
  } catch {
    // Quota exceeded or storage disabled — silently drop. The
    // feature is best-effort UX, not load-bearing.
  }
}

export function useRecentlyViewed() {
  if (!initialised) {
    initialised = true
    watch(entries, persist, { deep: true })
  }

  function record(path: string, title: string) {
    const now = Date.now()
    const filtered = entries.value.filter((e) => e.path !== path)
    filtered.unshift({ path, title, ts: now })
    entries.value = filtered.slice(0, MAX_ENTRIES)
  }

  function remove(path: string) {
    entries.value = entries.value.filter((e) => e.path !== path)
  }

  function clear() {
    entries.value = []
  }

  return {
    entries,
    record,
    remove,
    clear,
  }
}
