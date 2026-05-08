import client from './client'

export interface NoteTitleHit {
  title: string
  path: string
}

interface ListResp {
  items: NoteTitleHit[]
}

export async function suggestNoteTitles(q: string, limit = 10): Promise<NoteTitleHit[]> {
  const { data } = await client.get<ListResp>('/note-titles', { params: { q, limit } })
  return data.items
}
