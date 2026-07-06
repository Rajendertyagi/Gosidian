import client from './client'

export interface NoteSummary {
  path: string
  title: string
}

// MediaRef is the resolved payload of a media-style note — an image media
// note (ADR-013, `type: image`) or a CSV table note (ADR-016, `type: table`)
// — populated by the backend when the matching feature flag is on. `broken`
// is true when the `media:` pointer doesn't resolve.
export interface MediaRef {
  path: string
  url: string
  mime?: string
  size?: number
  broken?: boolean
}

export interface Note {
  path: string
  title: string
  content: string
  etag: string
  size: number
  mod_time: string
  kind?: 'image' | 'table'
  media?: MediaRef
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

export async function getNote(path: string, opts?: { inline?: boolean }): Promise<Note> {
  // inline=1 returns the content with image references embedded as data: URIs
  // (self-contained download); the stored note keeps the lightweight reference.
  const { data } = await client.get<Note>(`/notes/${encodeURIComponent(path)}`, {
    params: opts?.inline ? { inline: 1 } : undefined,
  })
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
