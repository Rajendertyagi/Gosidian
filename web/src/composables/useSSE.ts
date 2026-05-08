/**
 * useSSE — singleton EventSource bound to /api/v1/events. Phase 2bis
 * scaffolding: the connection lifecycle (open on auth, close on
 * logout, reconnect on token refresh) is in place; the real
 * subscribers (tree store invalidation, editor "modified externally"
 * badge) wire up in Phase 3 alongside the components themselves.
 *
 * Usage from a Vue component:
 *
 *   import { useSSE } from '@/composables/useSSE'
 *   const sse = useSSE()
 *   sse.on('tree', (payload) => treeStore.invalidate())
 *   sse.on('note', (payload) => editor.handleExternalUpdate(payload))
 *
 * The composable is idempotent: calling useSSE() many times returns
 * the same shared connection, so a tree-store consumer and an
 * editor consumer don't open two parallel sockets.
 */
import { ref, onScopeDispose } from 'vue'
import { useAuthStore } from '@/stores/auth'

export type SSETopic = 'tree' | 'note' | 'sidebar' | 'audit'

export interface SSEPayload {
  action?: string
  path?: string
  etag?: string
  source?: string
  [key: string]: unknown
}

type Listener = (payload: SSEPayload) => void

let sharedSource: EventSource | null = null
let sharedToken = ''
const listeners = new Map<SSETopic, Set<Listener>>()
const status = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle')

function dispatch(topic: SSETopic, raw: string) {
  let payload: SSEPayload = {}
  try {
    payload = JSON.parse(raw) as SSEPayload
  } catch {
    payload = { raw } as SSEPayload
  }
  const subs = listeners.get(topic)
  if (!subs) return
  for (const fn of subs) {
    try {
      fn(payload)
    } catch (err) {
      // Surface to console once; never let one bad listener kill
      // the bus for the others.
      console.error('useSSE listener error for', topic, err)
    }
  }
}

function connect(token: string, topics: SSETopic[]) {
  if (sharedSource) {
    if (sharedToken === token) return // already on this token
    sharedSource.close()
    sharedSource = null
  }
  sharedToken = token
  status.value = 'connecting'
  const params = new URLSearchParams()
  params.set('token', token)
  if (topics.length) params.set('topics', topics.join(','))
  const url = `/api/v1/events?${params.toString()}`
  const es = new EventSource(url)
  sharedSource = es

  es.onopen = () => {
    status.value = 'open'
  }
  es.onerror = () => {
    status.value = 'error'
    // Browser EventSource auto-reconnects with exponential backoff;
    // we just let it. If the underlying token is revoked the next
    // reconnect lands on a 401 and the API client interceptor
    // routes the user back to /login.
  }
  // Wire each topic explicitly. The default `message` handler isn't
  // useful because we always send named events from the server.
  for (const topic of ['tree', 'note', 'sidebar', 'audit'] as SSETopic[]) {
    es.addEventListener(topic, (e) => {
      const data = (e as MessageEvent).data ?? ''
      dispatch(topic, String(data))
    })
  }
}

function disconnect() {
  if (sharedSource) {
    sharedSource.close()
    sharedSource = null
  }
  sharedToken = ''
  status.value = 'closed'
}

export function useSSE(topics: SSETopic[] = []) {
  const auth = useAuthStore()

  // Open the connection lazily on first use, but only when auth is
  // ready. Components that need SSE call useSSE() and we tie the
  // lifecycle to the auth token presence.
  if (auth.token && (!sharedSource || sharedToken !== auth.token)) {
    connect(auth.token, topics)
  }

  return {
    status,
    on(topic: SSETopic, fn: Listener): () => void {
      let set = listeners.get(topic)
      if (!set) {
        set = new Set()
        listeners.set(topic, set)
      }
      set.add(fn)
      // Auto-remove on the calling component's unmount so
      // /events cache stays small as routes change.
      onScopeDispose(() => set?.delete(fn))
      return () => set?.delete(fn)
    },
    disconnect,
  }
}

/** Test/dev helper: closes the singleton + clears listeners. */
export function _resetSSEForTests() {
  disconnect()
  listeners.clear()
  status.value = 'idle'
}
