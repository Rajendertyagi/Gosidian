import client from './client'

export interface VersionInfo {
  version: string
  api: string
  build_time?: string
  commit?: string
  default_lang?: string
  enabled_langs?: string[]
}

/**
 * GET /api/v1/version — public, unauthenticated. Returns server
 * version + i18n defaults so the SPA can pick the initial UI
 * language from the operator's config before the user logs in.
 */
export async function getVersion(): Promise<VersionInfo> {
  const { data } = await client.get<VersionInfo>('/version')
  return data
}
