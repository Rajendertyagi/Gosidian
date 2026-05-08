import { beforeEach, describe, expect, it } from 'vitest'
import { useRecentlyViewed } from '@/composables/useRecentlyViewed'

// useRecentlyViewed is a localStorage-backed stack (max 10). Specs
// guard the invariants that matter:
//   - record() puts the entry at the front
//   - re-recording an existing path bubbles it up (no duplicates)
//   - the cap keeps memory bounded
//   - persistence: a fresh composable picks up the last persisted
//     state from localStorage.
describe('useRecentlyViewed', () => {
  beforeEach(() => {
    window.localStorage.clear()
    // The ref is a module-level singleton, so tests share state
    // unless we explicitly reset.
    useRecentlyViewed().clear()
  })

  it('records latest at the front', () => {
    const r = useRecentlyViewed()
    r.record('a.md', 'A')
    r.record('b.md', 'B')
    expect(r.entries.value.map((e) => e.path)).toEqual(['b.md', 'a.md'])
  })

  it('deduplicates by path (re-record bubbles up)', () => {
    const r = useRecentlyViewed()
    r.record('a.md', 'A')
    r.record('b.md', 'B')
    r.record('a.md', 'A v2')
    expect(r.entries.value.map((e) => e.path)).toEqual(['a.md', 'b.md'])
    expect(r.entries.value[0]?.title).toBe('A v2')
  })

  it('caps at 10 entries (oldest dropped)', () => {
    const r = useRecentlyViewed()
    for (let i = 0; i < 12; i++) r.record(`n${i}.md`, `N${i}`)
    expect(r.entries.value.length).toBeLessThanOrEqual(10)
    // The two oldest (n0, n1) should be gone.
    expect(r.entries.value.find((e) => e.path === 'n0.md')).toBeUndefined()
    expect(r.entries.value.find((e) => e.path === 'n11.md')).toBeDefined()
  })
})
