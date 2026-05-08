/**
 * useSidebarResize — drag handle + persisted width composable.
 *
 * Bounds: 180–560 px (mirrors v1.x sidebar-resize.js). Double-click
 * the handle resets to 260 px default. Persisted in localStorage
 * under `gosidian.sidebar-width`.
 *
 * Returns:
 *   - width: ref<number> — bind to inline style
 *   - dragging: ref<boolean> — for cursor / overlay UX
 *   - startDrag: (e: PointerEvent) => void — wire to handle pointerdown
 *   - reset: () => void — wire to handle dblclick
 */
import { ref, onUnmounted } from 'vue'

const STORAGE_KEY = 'gosidian.sidebar-width'
const MIN = 180
const MAX = 560
const DEFAULT = 260

function loadWidth(): number {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT
    const n = parseInt(raw, 10)
    if (!Number.isFinite(n) || n < MIN || n > MAX) return DEFAULT
    return n
  } catch {
    return DEFAULT
  }
}

export function useSidebarResize() {
  const width = ref<number>(loadWidth())
  const dragging = ref<boolean>(false)

  function persist(value: number) {
    try {
      localStorage.setItem(STORAGE_KEY, String(value))
    } catch {
      // ignore — see useRecentlyViewed for rationale
    }
  }

  let raf = 0
  function onMove(e: PointerEvent) {
    if (!dragging.value) return
    const next = Math.max(MIN, Math.min(MAX, e.clientX))
    if (raf) cancelAnimationFrame(raf)
    raf = requestAnimationFrame(() => {
      width.value = next
    })
  }
  function onUp() {
    if (!dragging.value) return
    dragging.value = false
    persist(width.value)
    document.removeEventListener('pointermove', onMove)
    document.removeEventListener('pointerup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  function startDrag(e: PointerEvent) {
    e.preventDefault()
    dragging.value = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    document.addEventListener('pointermove', onMove)
    document.addEventListener('pointerup', onUp)
  }

  function reset() {
    width.value = DEFAULT
    persist(DEFAULT)
  }

  onUnmounted(() => {
    document.removeEventListener('pointermove', onMove)
    document.removeEventListener('pointerup', onUp)
    if (raf) cancelAnimationFrame(raf)
  })

  return { width, dragging, startDrag, reset }
}
