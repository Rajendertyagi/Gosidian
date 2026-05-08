import client from './client'

export interface NoteSummary {
  path: string
  title: string
}

export interface Note {
  path: string
  title: string
  content: string
  etag: string
  size: number
  mod_time: string
}

export interface ListResponse {
  items: NoteSummary[]
  total: number
  limit: number
  offset: number
}

export async function listNotes(
  params: { project?: string; tag?: string; limit?: number } = {},
): Promise<ListResponse> {
  const { data } = await client.get<ListResponse>('/notes', { params })
  return data
}

export async function getNote(path: string): Promise<Note> {
  const { data } = await client.get<Note>(`/notes/${encodeURIComponent(path)}`)
  return data
}

export async function createNote(path: string, content: string): Promise<Note> {
  const { data } = await client.post<Note>('/notes', { path, content })
  return data
}

export async function updateNote(
  path: string,
  body: { content: string; ifMatch?: string },
): Promise<Note> {
  const headers: Record<string, string> = {}
  if (body.ifMatch) headers['If-Match'] = body.ifMatch
  const { data } = await client.put<Note>(
    `/notes/${encodeURIComponent(path)}`,
    { content: body.content },
    { headers },
  )
  return data
}

export async function deleteNote(path: string): Promise<void> {
  await client.delete(`/notes/${encodeURIComponent(path)}`)
}
