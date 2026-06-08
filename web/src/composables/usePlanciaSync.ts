/**
 * Plancia persistence — bidirectional sync between the window store and the
 * URL (`?w=&f=`), with a localStorage fallback that restores the last plancia
 * when the URL carries no window state.
 *
 * Rewritten from products-dc's `usePlanciaUrlSync` (which keyed windows by
 * numeric id) for gosidian, where windows are keyed by **note path (string)**
 * or are **singletons** (settings, search, graph…). A token is either the bare
 * type (singleton) or `type:<encodeURIComponent(arg)>`; parsing splits on the
 * FIRST `:` so an encoded path (no raw `:`) round-trips cleanly.
 *
 * Returns `hydrate`; the host (AppShell) calls it in onMounted. Owns the two
 * watchers (store→URL+localStorage debounced; URL→store on back/forward).
 */
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWindowsStore } from '@/stores/windows'

const LS_KEY = 'gosidian.plancia'

/** Per-type rules: which prop becomes the URL arg, and how to rebuild props. */
interface TypeSpec {
  arg: (props: Record<string, unknown>) => string | null
  props: (arg: string | null) => Record<string, unknown>
  title: (arg: string | null) => string
}

const str = (v: unknown): string | null => (typeof v === 'string' && v ? v : null)
const base = (p: string): string => (p.split('/').pop() ?? p).replace(/\.md$/, '')

const PATH_TYPE = (): TypeSpec => ({
  arg: (p) => str(p.path),
  props: (a) => (a ? { path: a } : {}),
  title: (a) => (a ? base(a) : ''),
})

const TYPE_SPECS: Record<string, TypeSpec> = {
  note: PATH_TYPE(),
  edit: PATH_TYPE(),
  history: PATH_TYPE(),
  graph: {
    arg: (p) => str(p.focus),
    props: (a) => (a ? { focus: a, depth: 1 } : {}),
    title: (a) => (a ? `↳ ${base(a)}` : 'Graph'),
  },
  tags: {
    arg: (p) => str(p.tag),
    props: (a) => (a ? { tag: a } : {}),
    title: (a) => (a ? `#${a}` : 'Tags'),
  },
  admin: {
    arg: (p) => str(p.section),
    props: (a) => (a ? { section: a } : {}),
    title: (a) => (a ? `Admin · ${a}` : 'Admin'),
  },
}

const SINGLETON_TITLE: Record<string, string> = {
  search: 'Search',
  projects: 'Projects',
  settings: 'Settings',
  trash: 'Trash',
}
const specFor = (type: string): TypeSpec =>
  TYPE_SPECS[type] ?? {
    arg: () => null,
    props: () => ({}),
    title: () => SINGLETON_TITLE[type] ?? type,
  }

/** Stable de-dup key (decoded). Use this when opening windows from the app so
 *  a hydrated window de-dups against a later open of the same target. */
export function planciaKey(type: string, arg?: string | null): string {
  return arg ? `${type}:${arg}` : type
}

/** URL token for a window: bare type (singleton) or `type:<encodeURIComponent(arg)>`.
 *  Pure — exported for testing. */
export function tokenForWindow(w: { type: string; props: Record<string, unknown> }): string {
  const a = specFor(w.type).arg(w.props)
  return a ? `${w.type}:${encodeURIComponent(a)}` : w.type
}

/** Parse a URL token back to {type, arg}; splits on the FIRST `:`. Pure. */
export function parseToken(tok: string): { type: string; arg: string | null } | null {
  const t = tok.trim()
  if (!t) return null
  const i = t.indexOf(':')
  if (i < 0) return { type: t, arg: null }
  let arg: string
  try {
    arg = decodeURIComponent(t.slice(i + 1))
  } catch {
    arg = t.slice(i + 1)
  }
  return { type: t.slice(0, i), arg: arg || null }
}

/** Open spec for a parsed token (initial title/props from the type spec). */
export function specForToken(type: string, arg: string | null): {
  title: string
  props: Record<string, unknown>
} {
  const spec = specFor(type)
  return { title: spec.title(arg), props: spec.props(arg) }
}

export function usePlanciaSync() {
  const store = useWindowsStore()
  const route = useRoute()
  const router = useRouter()

  function openToken(tok: string, wantFocus = true): void {
    const p = parseToken(tok)
    if (!p) return
    // Legacy /notes/:path/edit deep-link → open the note window in edit mode;
    // de-dups with the read window and the URL normalises to `note:<path>`.
    if (p.type === 'edit') {
      store.open({
        type: 'note',
        key: planciaKey('note', p.arg),
        title: p.arg ? base(p.arg) : '',
        props: p.arg ? { path: p.arg, mode: 'edit' } : {},
        focus: wantFocus,
      })
      return
    }
    const { title, props } = specForToken(p.type, p.arg)
    store.open({ type: p.type, key: planciaKey(p.type, p.arg), title, props, focus: wantFocus })
  }

  const winTokens = computed(() => store.windows.map(tokenForWindow))
  const focusToken = computed(() => (store.focused ? tokenForWindow(store.focused) : null))

  function persistLocal(w: string, f: string | null) {
    try {
      localStorage.setItem(LS_KEY, JSON.stringify({ w, f }))
    } catch {
      // ignore — storage may be unavailable (private mode / quota)
    }
  }

  let pushT: ReturnType<typeof setTimeout> | null = null
  function pushUrl() {
    const w = winTokens.value.join(',')
    const query: Record<string, string> = {}
    if (w) query.w = w
    if (focusToken.value != null) query.f = focusToken.value
    router.replace({ query }).catch(() => {})
    persistLocal(w, focusToken.value)
  }
  watch([winTokens, focusToken], () => {
    if (pushT) clearTimeout(pushT)
    pushT = setTimeout(pushUrl, 200)
  })

  function readLocal(): { w: string; f: string | null } | null {
    try {
      const raw = localStorage.getItem(LS_KEY)
      if (!raw) return null
      const o = JSON.parse(raw)
      if (typeof o?.w !== 'string') return null
      return { w: o.w, f: typeof o.f === 'string' ? o.f : null }
    } catch {
      return null
    }
  }

  function hydrate() {
    store.reset()
    // URL is primary; fall back to the last persisted plancia when empty.
    let w = String(route.query.w ?? '')
    let f = String(route.query.f ?? '')
    if (!w) {
      const local = readLocal()
      if (local) {
        w = local.w
        f = local.f ?? ''
      }
    }
    for (const tok of w.split(',')) openToken(tok, false)
    const fp = parseToken(f)
    if (fp) {
      const win = store.windows.find((x) => x.key === planciaKey(fp.type, fp.arg))
      if (win) store.focus(win.id)
    } else if (store.windows.length) {
      const last = store.windows[store.windows.length - 1]
      if (last) store.focus(last.id)
    }
  }

  // Back/forward: re-hydrate only if the URL token set actually changed.
  watch(
    () => [route.query.w, route.query.f],
    () => {
      const urlW = String(route.query.w ?? '')
      if (urlW !== winTokens.value.join(',')) {
        hydrate()
        return
      }
      const fp = parseToken(String(route.query.f ?? ''))
      if (fp) {
        const win = store.windows.find((x) => x.key === planciaKey(fp.type, fp.arg))
        if (win && win.id !== store.focusedId) store.focus(win.id)
      }
    },
  )

  return { hydrate, winTokens, focusToken }
}
