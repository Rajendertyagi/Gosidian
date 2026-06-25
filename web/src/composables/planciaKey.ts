/**
 * Host-side window keying + URL codec for gosidian's plancia.
 *
 * plancia's `usePlanciaSync(codec, options)` takes a pluggable `PlanciaCodec`;
 * the library no longer owns gosidian's per-type token scheme. This module:
 *
 *  - re-homes `planciaKey` (the stable, decoded de-dup key used across the app
 *    when opening windows) and `base` (note-path → display title) host-side,
 *    since plancia doesn't export them; and
 *  - builds the gosidian `codec`, mostly via the library's `createArgCodec`,
 *    with one bespoke wrap: the legacy `edit:<path>` deep-link decodes to a
 *    **note** window in edit mode, keyed/normalised as `note:<path>` so the URL
 *    collapses to `note:` and the window de-dups against an open read window.
 *
 * The token scheme (unchanged from the previous inlined composable): a token is
 * either the bare type (singleton) or `type:<encodeURIComponent(arg)>`; parsing
 * splits on the FIRST `:` so an encoded path (no raw `:`) round-trips cleanly.
 */
import { createArgCodec, type ArgTypeSpec, type PlanciaCodec } from 'plancia'

const str = (v: unknown): string | null => (typeof v === 'string' && v ? v : null)

/** Note path → display title: last segment without the `.md` suffix. */
export const base = (p: string): string => (p.split('/').pop() ?? p).replace(/\.md$/, '')

/** Stable de-dup key (decoded). Use this when opening windows from the app so a
 *  hydrated window de-dups against a later open of the same target. Mirrors the
 *  codec's own `key()` so host opens and URL hydration agree. */
export function planciaKey(type: string, arg?: string | null): string {
  return arg ? `${type}:${arg}` : type
}

/** Windows whose URL arg is the note path, with `base()` as the title. */
const PATH_TYPE = (): ArgTypeSpec => ({
  arg: (p) => str(p.path),
  props: (a) => (a ? { path: a } : {}),
  title: (a) => (a ? base(a) : ''),
})

/** gosidian's per-type URL scheme, expressed for `createArgCodec`. */
const baseCodec: PlanciaCodec = createArgCodec({
  bareToken: 'type',
  types: {
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
    // Singletons with a fixed display title.
    search: { arg: () => null, title: () => 'Search' },
    projects: { arg: () => null, title: () => 'Projects' },
    settings: { arg: () => null, title: () => 'Settings' },
    trash: { arg: () => null, title: () => 'Trash' },
  },
  // Unlisted types are bare singletons titled by their type.
  default: { arg: () => null },
})

/**
 * gosidian codec: `createArgCodec` for the standard types, plus a bespoke
 * `decode` that normalises the legacy `edit:<path>` deep-link to a `note`
 * window in edit mode. `encode`/`key` delegate to the base codec unchanged — a
 * window's `type` is always `note` once open, so it always serialises to
 * `note:<path>` (the `edit:` token only ever appears as an inbound deep-link).
 */
export const codec: PlanciaCodec = {
  key: baseCodec.key,
  encode: baseCodec.encode,
  decode(token) {
    const spec = baseCodec.decode(token)
    if (spec?.type === 'edit') {
      const path = str(spec.props?.path)
      return {
        type: 'note',
        key: planciaKey('note', path),
        title: path ? base(path) : '',
        props: path ? { path, mode: 'edit' } : {},
      }
    }
    return spec
  },
}
