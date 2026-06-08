/**
 * Typed wrapper for /api/v1/insights/pending (owner-only). Backs the
 * "new insights" badge — the count of un-triaged self-improvement insights.
 * Shape mirrors pendingInsightsResponse in internal/api/v1/insights.go.
 */
import client from './client'

export interface InsightRef {
  path: string
  title: string
}

export interface PendingInsights {
  enabled: boolean
  project: string
  count: number
  notes: InsightRef[]
}

export async function fetchPendingInsights(): Promise<PendingInsights> {
  const { data } = await client.get<PendingInsights>('/insights/pending')
  return data
}
