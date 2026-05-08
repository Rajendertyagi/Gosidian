/**
 * Tree store — caches /api/v1/tree responses keyed by project (empty
 * string for vault-wide). The store invalidates entries on SSE
 * `tree` events, which the AppShell wires up so any tab sees writes
 * from MCP / the watcher / other tabs in real time.
 */
import { defineStore } from 'pinia'
import type { TreeNode } from '@/api/tree'
import { fetchTree } from '@/api/tree'

interface TreeState {
  byProject: Record<string, TreeNode | null>
  loading: Record<string, boolean>
  error: Record<string, string>
}

export const useTreeStore = defineStore('tree', {
  state: (): TreeState => ({
    byProject: {},
    loading: {},
    error: {},
  }),

  actions: {
    async load(project = '') {
      const key = project
      this.loading[key] = true
      this.error[key] = ''
      try {
        this.byProject[key] = await fetchTree(project || undefined)
      } catch (err) {
        this.error[key] = err instanceof Error ? err.message : String(err)
      } finally {
        this.loading[key] = false
      }
    },
    invalidate(project = '') {
      delete this.byProject[project]
    },
    invalidateAll() {
      this.byProject = {}
    },
  },
})
