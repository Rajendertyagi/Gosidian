import client from './client'

export interface PreviewResponse {
  html: string
}

/** POST /api/v1/preview — render markdown to sanitized HTML. */
export async function renderPreview(markdown: string): Promise<string> {
  const { data } = await client.post<PreviewResponse>('/preview', { markdown })
  return data.html
}
