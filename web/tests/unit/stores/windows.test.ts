import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useWindowsStore } from '@/stores/windows'

describe('windows store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('open() appends, focuses, and returns an id', () => {
    const s = useWindowsStore()
    const id = s.open({ type: 'note', key: 'note:a', props: { path: 'a' } })
    expect(s.windows).toHaveLength(1)
    expect(s.focusedId).toBe(id)
    expect(s.focused?.key).toBe('note:a')
  })

  it('open() de-dups by key and refocuses/restores instead of duplicating', () => {
    const s = useWindowsStore()
    const a = s.open({ type: 'note', key: 'note:a', props: { path: 'a' } })
    s.open({ type: 'note', key: 'note:b', props: { path: 'b' } })
    s.minimize(a)
    const again = s.open({ type: 'note', key: 'note:a', props: { path: 'a' } })
    expect(again).toBe(a)
    expect(s.windows).toHaveLength(2)
    expect(s.focusedId).toBe(a)
    expect(s._byId(a)?.minimized).toBe(false) // restored
  })

  it('open() inserts the new window immediately right of the focused one', () => {
    const s = useWindowsStore()
    const a = s.open({ type: 'note', key: 'note:a', props: {} })
    s.open({ type: 'note', key: 'note:b', props: {} })
    s.focus(a)
    s.open({ type: 'note', key: 'note:c', props: {} })
    expect(s.windows.map((w) => w.key)).toEqual(['note:a', 'note:c', 'note:b'])
  })

  it('cycleWidth steps s → m → full → s', () => {
    const s = useWindowsStore()
    const id = s.open({ type: 'note', key: 'note:a', props: {}, width: 's' })
    const w = () => s._byId(id)!.width
    expect(w()).toBe('s')
    s.cycleWidth(id)
    expect(w()).toBe('m')
    s.cycleWidth(id)
    expect(w()).toBe('full')
    s.cycleWidth(id)
    expect(w()).toBe('s')
  })

  it('minimize moves focus to the last visible window; restore re-focuses', () => {
    const s = useWindowsStore()
    const a = s.open({ type: 'note', key: 'note:a', props: {} })
    const b = s.open({ type: 'note', key: 'note:b', props: {} })
    s.focus(b)
    s.minimize(b)
    expect(s.minimizedList.map((w) => w.key)).toEqual(['note:b'])
    expect(s.focusedId).toBe(a) // last visible
    s.restore(b)
    expect(s.focusedId).toBe(b)
    expect(s.minimizedList).toHaveLength(0)
  })

  it('close() picks a sensible next focus and clears focus when empty', () => {
    const s = useWindowsStore()
    const a = s.open({ type: 'note', key: 'note:a', props: {} })
    const b = s.open({ type: 'note', key: 'note:b', props: {} })
    s.focus(a)
    s.close(a)
    expect(s.focusedId).toBe(b)
    s.close(b)
    expect(s.windows).toHaveLength(0)
    expect(s.focusedId).toBeNull()
  })

  it('focusAdjacent clamps at the ends and skips minimized windows', () => {
    const s = useWindowsStore()
    const a = s.open({ type: 'note', key: 'note:a', props: {} })
    const b = s.open({ type: 'note', key: 'note:b', props: {} })
    s.open({ type: 'note', key: 'note:c', props: {} })
    s.minimize(b)
    s.focus(a)
    s.focusAdjacent(1)
    expect(s.focused?.key).toBe('note:c') // b is minimized, skipped
    s.focusAdjacent(1)
    expect(s.focused?.key).toBe('note:c') // clamped at the right end
    s.focusAdjacent(-1)
    expect(s.focused?.key).toBe('note:a')
  })
})
