import client from './client'

export interface HistoryEntry {
  sha: string
  short_sha: string
  author: string
  date: string
  subject: string
}

interface ListResp {
  items: HistoryEntry[]
  total: number
  limit: number
}

export async function getHistory(path: string, limit = 50): Promise<HistoryEntry[]> {
  const { data } = await client.get<ListResp>(
    `/notes/${encodeURIComponent(path)}/history`,
    { params: { limit } },
  )
  return data.items
}
