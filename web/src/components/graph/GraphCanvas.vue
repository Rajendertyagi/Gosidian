<script setup lang="ts">
/**
 * GraphCanvas — Cytoscape 3.30 + fcose layout. The bulky Cytoscape
 * runtime (~250KB raw / ~80KB gz) is dynamically imported here so
 * routes that don't open /graph never pay for it. fcose handles
 * layouts up to ~500 nodes smoothly; larger vaults will need WebGL
 * (Sigma.js) — deferred to v2.1 per plan.
 *
 * The component is presentational: it receives `nodes` + `edges` and
 * emits `select(path)` when the user clicks a node. Filter state
 * lives in the parent (GraphView) so the URL stays sharable.
 *
 * Theme: Cytoscape doesn't resolve CSS `var(--color-x)` references
 * inside style values — they fall back to its internal default
 * (black). We resolve the tokens up front via getComputedStyle and
 * pass concrete `rgb(R G B)` strings instead. We re-resolve on
 * theme switching by watching the `data-preset` attribute on
 * <html>; that lets the user flip Mocha → Latte without remounting
 * the canvas.
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Core, ElementDefinition } from 'cytoscape'
import type { GraphEdge, GraphNode } from '@/api/graph'

interface Props {
  nodes: GraphNode[]
  edges: GraphEdge[]
}
const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'select', path: string): void
}>()

const host = ref<HTMLDivElement | null>(null)
let cy: Core | null = null
let presetObserver: MutationObserver | null = null
let resizeObserver: ResizeObserver | null = null
let resizeRaf = 0

function elementsFor(nodes: GraphNode[], edges: GraphEdge[]): ElementDefinition[] {
  const els: ElementDefinition[] = []
  for (const n of nodes) {
    els.push({
      group: 'nodes',
      data: { id: n.id, label: n.label, project: n.project ?? '', degree: n.degree },
    })
  }
  for (const e of edges) {
    els.push({
      group: 'edges',
      data: {
        id: `${e.source}__${e.target}`,
        source: e.source,
        target: e.target,
        count: e.count,
        cross: e.cross_project ?? false,
      },
    })
  }
  return els
}

interface ThemeColors {
  accent: string
  warning: string
  text: string
  textMuted: string
  bg: string
}

// Tokens are stored as `R G B` triplets (Tailwind-alpha-friendly).
// Cytoscape's internal color parser is older than CSS Color Module
// 4 and refuses the space-separated `rgb(R G B)` form — it only
// accepts `rgb(R, G, B)` with commas. We split + recompose on
// commas so the painted labels actually use the theme colors.
// Hard-coded comma-form fallbacks default to Catppuccin Mocha so
// the canvas never falls back to black-on-black.
function tripletToCommaRGB(triplet: string): string | null {
  const parts = triplet.trim().split(/\s+/).filter(Boolean)
  if (parts.length !== 3) return null
  const nums = parts.map((p) => Number(p))
  if (nums.some((n) => !Number.isFinite(n))) return null
  return `rgb(${nums[0]}, ${nums[1]}, ${nums[2]})`
}

function resolveTheme(): ThemeColors {
  const root = document.documentElement
  const cs = getComputedStyle(root)
  const get = (name: string, fallback: string): string => {
    const raw = cs.getPropertyValue(name)
    const rgb = tripletToCommaRGB(raw)
    return rgb ?? fallback
  }
  return {
    accent: get('--color-accent', 'rgb(137, 180, 250)'),
    warning: get('--color-warning', 'rgb(250, 179, 135)'),
    text: get('--color-text', 'rgb(205, 214, 244)'),
    textMuted: get('--color-text-muted', 'rgb(166, 173, 200)'),
    bg: get('--color-bg', 'rgb(30, 30, 46)'),
  }
}

function styleSheet(t: ThemeColors) {
  return [
    {
      selector: 'node',
      style: {
        'background-color': t.accent,
        label: 'data(label)',
        color: t.text,
        'font-size': '10px',
        'text-valign': 'bottom',
        'text-margin-y': 4,
        'text-outline-color': t.bg,
        'text-outline-width': 2,
        // Trim long labels by default; the hovered class below
        // bumps the wrap width so the full title is readable.
        'text-wrap': 'ellipsis',
        'text-max-width': '120px',
        width: 'mapData(degree, 0, 8, 8, 28)',
        height: 'mapData(degree, 0, 8, 8, 28)',
      },
    },
    {
      selector: 'edge',
      style: {
        'curve-style': 'bezier',
        'line-color': t.textMuted,
        width: 'mapData(count, 1, 6, 1, 4)',
        opacity: 0.6,
      },
    },
    {
      selector: 'edge[?cross]',
      style: {
        'line-style': 'dashed',
        opacity: 0.4,
      },
    },
    {
      selector: 'node:selected',
      style: {
        'background-color': t.warning,
        'border-width': 2,
        'border-color': t.warning,
      },
    },
    // Hover state — applied via mouseover handler in mount() (Cytoscape
    // doesn't support :hover in selectors). The hovered node is
    // brought to the foreground (z-index 999) and gets a larger,
    // bolder, full-width label so the title is always readable even
    // when neighbours overlap.
    {
      selector: 'node.hovered',
      style: {
        'background-color': t.warning,
        'border-width': 2,
        'border-color': t.warning,
        'font-size': '14px',
        'font-weight': 'bold',
        'text-wrap': 'wrap',
        'text-max-width': '240px',
        'text-outline-width': 3,
        'z-compound-depth': 'top',
        'z-index': 999,
      },
    },
    // Connected edges + neighbour nodes light up too — the typical
    // graph-explore affordance: hover a node, see its neighbourhood.
    {
      selector: 'edge.connected',
      style: {
        'line-color': t.accent,
        opacity: 1,
        width: 'mapData(count, 1, 6, 2, 6)',
        'z-index': 50,
      },
    },
    {
      selector: 'node.neighbour',
      style: {
        'border-width': 1,
        'border-color': t.accent,
        'z-index': 50,
      },
    },
    // Everything else fades back so the hovered cluster pops.
    {
      selector: '.dimmed',
      style: {
        opacity: 0.25,
      },
    },
  ]
}

async function ensureCytoscape() {
  const [{ default: cytoscape }, { default: fcose }] = await Promise.all([
    import('cytoscape'),
    import('cytoscape-fcose'),
  ])
  if (!fcoseRegistered) {
    cytoscape.use(fcose)
    fcoseRegistered = true
  }
  return cytoscape
}
let fcoseRegistered = false

function reapplyTheme() {
  if (!cy) return
  // Cytoscape's StylesheetJson type is internal; cast to satisfy
  // its generic style() overload.
  ;(cy as unknown as { style: (s: unknown) => void }).style(styleSheet(resolveTheme()))
}

async function mount() {
  if (!host.value) return
  const cytoscape = await ensureCytoscape()
  cy = cytoscape({
    container: host.value,
    elements: elementsFor(props.nodes, props.edges),
    minZoom: 0.2,
    maxZoom: 4,
    wheelSensitivity: 0.2,
    // Cytoscape's StylesheetJson type is internal; cast through unknown.
    style: styleSheet(resolveTheme()) as never,
    layout: { name: 'fcose', animate: false, randomize: true } as unknown as { name: string },
  })
  cy.on('tap', 'node', (evt) => {
    emit('select', evt.target.id())
  })

  // Hover affordance: bring the hovered node forward, light up its
  // 1-hop neighbourhood, dim everything else. `.dimmed` is applied
  // only to the *complement* of the hovered cluster — adding it to
  // the cluster too would faded their opacity (the .dimmed style
  // block sits after .hovered in the stylesheet, so its opacity
  // declaration wins on overlap).
  type Collection = {
    addClass: (c: string) => Collection
    removeClass: (c: string) => Collection
    union: (c: Collection) => Collection
    difference: (c: Collection) => Collection
    neighborhood: () => Collection
  }
  cy.on('mouseover', 'node', (evt) => {
    if (!cy) return
    const node = evt.target as unknown as Collection
    const nbrs = node.neighborhood()
    const cluster = node.union(nbrs)
    const others = (cy as unknown as { elements: () => Collection })
      .elements()
      .difference(cluster)
    others.addClass('dimmed')
    node.addClass('hovered')
    nbrs.addClass('connected')
    nbrs.addClass('neighbour')
  })
  cy.on('mouseout', 'node', () => {
    if (!cy) return
    const everything = (cy as unknown as { elements: () => Collection }).elements()
    everything.removeClass('dimmed')
    everything.removeClass('hovered')
    everything.removeClass('connected')
    everything.removeClass('neighbour')
  })

  // React to preset switches (Mocha → Latte → Tokyo Night, etc.) so
  // the canvas reflects the new tokens without a remount.
  presetObserver = new MutationObserver(reapplyTheme)
  presetObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-preset'],
  })

  // Re-fit when the host resizes — e.g. a plancia window cycling its width
  // step, or the sidebar drag. Cytoscape needs an explicit resize() to pick
  // up the new container box; rAF-coalesced so a drag doesn't thrash layout.
  resizeObserver = new ResizeObserver(() => {
    if (resizeRaf) cancelAnimationFrame(resizeRaf)
    resizeRaf = requestAnimationFrame(() => {
      cy?.resize()
      cy?.fit(undefined, 30)
    })
  })
  resizeObserver.observe(host.value)
}

onMounted(mount)
onBeforeUnmount(() => {
  presetObserver?.disconnect()
  presetObserver = null
  resizeObserver?.disconnect()
  resizeObserver = null
  if (resizeRaf) cancelAnimationFrame(resizeRaf)
  cy?.destroy()
  cy = null
})

watch(
  () => [props.nodes, props.edges],
  () => {
    if (!cy) return
    cy.elements().remove()
    cy.add(elementsFor(props.nodes, props.edges))
    cy.layout({ name: 'fcose', animate: false, randomize: true } as unknown as { name: string }).run()
  },
  { deep: false },
)
</script>

<template>
  <div ref="host" class="w-full h-full bg-bg" />
</template>
