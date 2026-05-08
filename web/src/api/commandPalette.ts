import client from './client'

export interface CmdNote {
  path: string
  title: string
}
export interface CmdProject {
  name: string
  noteCount: number
}
export interface CmdTag {
  tag: string
  count: number
}
export interface CommandPaletteData {
  notes: CmdNote[]
  projects: CmdProject[]
  tags: CmdTag[]
}

/**
 * GET /api/v1/command-palette — full dataset for Cmd+K. The SPA
 * caches it after the first open and revalidates on focus.
 */
export async function fetchCommandPalette(): Promise<CommandPaletteData> {
  const { data } = await client.get<CommandPaletteData>('/command-palette')
  return data
}
