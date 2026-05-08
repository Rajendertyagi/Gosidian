import client from './client'

export interface TrashItem {
  id: string
  origin_path: string
  discarded_at: string
  is_dir: boolean
}

interface ListResp {
  items: TrashItem[]
  total: number
}

export async function listTrash(): Promise<TrashItem[]> {
  const { data } = await client.get<ListResp>('/trash')
  return data.items
}

export async function restoreTrash(id: string): Promise<{ restored: string }> {
  const { data } = await client.post<{ restored: string }>(
    `/trash/${encodeURIComponent(id)}/restore`,
    {},
  )
  return data
}

export async function purgeTrash(id: string): Promise<void> {
  await client.delete(`/trash/${encodeURIComponent(id)}`)
}
