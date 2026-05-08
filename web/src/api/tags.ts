import client from './client'

export interface TagCount {
  tag: string
  count: number
}

export interface TagListResponse {
  items: TagCount[]
  total: number
}

export interface NoteSummary {
  path: string
  title: string
}

export interface TagNotesResponse {
  items: NoteSummary[]
  total: number
}

export async function listTags(project?: string): Promise<TagCount[]> {
  const { data } = await client.get<TagListResponse>('/tags', {
    params: project ? { project } : undefined,
  })
  return data.items
}

export async function notesByTag(tag: string, project?: string): Promise<NoteSummary[]> {
  const { data } = await client.get<TagNotesResponse>(`/tags/${encodeURIComponent(tag)}`, {
    params: project ? { project } : undefined,
  })
  return data.items
}
