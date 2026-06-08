import { describe, expect, it } from 'vitest'
import { tokenForWindow, parseToken, planciaKey, specForToken } from '@/composables/usePlanciaSync'

describe('plancia token round-trip', () => {
  it('encodes a note path as note:<encoded> and decodes back', () => {
    const tok = tokenForWindow({ type: 'note', props: { path: 'gosidian/hot.md' } })
    expect(tok).toBe('note:' + encodeURIComponent('gosidian/hot.md'))
    expect(parseToken(tok)).toEqual({ type: 'note', arg: 'gosidian/hot.md' })
  })

  it('round-trips paths with special chars (colon, comma, space, unicode)', () => {
    for (const path of ['a/b,c.md', 'weird: note.md', 'spazî/notà èè.md', 'x:y:z.md']) {
      const tok = tokenForWindow({ type: 'edit', props: { path } })
      // The delimiter is the first ':'; the encoded arg carries no raw ':'.
      expect(tok.indexOf(':')).toBe('edit'.length)
      expect(parseToken(tok)).toEqual({ type: 'edit', arg: path })
    }
  })

  it('singletons serialise to the bare type', () => {
    expect(tokenForWindow({ type: 'settings', props: {} })).toBe('settings')
    expect(tokenForWindow({ type: 'search', props: {} })).toBe('search')
    expect(parseToken('settings')).toEqual({ type: 'settings', arg: null })
  })

  it('graph focus uses the focus prop; bare graph is the global graph', () => {
    expect(tokenForWindow({ type: 'graph', props: {} })).toBe('graph')
    const ego = tokenForWindow({ type: 'graph', props: { focus: 'p/n.md', depth: 1 } })
    expect(ego).toBe('graph:' + encodeURIComponent('p/n.md'))
    expect(parseToken(ego)).toEqual({ type: 'graph', arg: 'p/n.md' })
  })

  it('parseToken rejects empty input', () => {
    expect(parseToken('')).toBeNull()
    expect(parseToken('   ')).toBeNull()
  })

  it('planciaKey matches the decoded de-dup convention', () => {
    expect(planciaKey('note', 'a/b.md')).toBe('note:a/b.md')
    expect(planciaKey('settings')).toBe('settings')
  })

  it('specForToken rebuilds props and an initial title', () => {
    expect(specForToken('note', 'a/b.md')).toEqual({ title: 'b', props: { path: 'a/b.md' } })
    expect(specForToken('graph', 'a/b.md')).toEqual({ title: '↳ b', props: { focus: 'a/b.md', depth: 1 } })
    expect(specForToken('settings', null)).toEqual({ title: 'Settings', props: {} })
  })
})
