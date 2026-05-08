import client from './client'

export interface AttachResponse {
  path: string
  filename: string
  markdown: string
}

/**
 * POST /api/v1/attach — multipart upload for paste/drop into the
 * editor. Server returns the ready-to-splice markdown embed string.
 * Optional `project` scopes the attachment under a project's
 * attachments/ folder; falls back to vault root otherwise.
 */
export async function attachFile(file: File, project?: string): Promise<AttachResponse> {
  const form = new FormData()
  form.append('file', file)
  const params: Record<string, string> = {}
  if (project) params.project = project
  const { data } = await client.post<AttachResponse>('/attach', form, {
    params,
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}
