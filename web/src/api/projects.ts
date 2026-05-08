import client from './client'

export interface Project {
  name: string
  note_count: number
  hidden_from_mcp: boolean
  skip_git_sync: boolean
  /** RFC 3339 UTC string of the directory's mtime — proxy for
   *  "last activity" (fs birth time isn't preserved by rsync /
   *  git checkout / container layer copy). Empty when stat failed. */
  mod_time?: string
}

export interface ProjectListResponse {
  items: Project[]
  total: number
}

export interface UpdateProjectRequest {
  new_name?: string
  hidden_from_mcp?: boolean
  skip_git_sync?: boolean
}

export async function listProjects(): Promise<Project[]> {
  const { data } = await client.get<ProjectListResponse>('/projects')
  return data.items
}

export async function getProject(slug: string): Promise<Project> {
  const { data } = await client.get<Project>(`/projects/${encodeURIComponent(slug)}`)
  return data
}

export async function createProject(name: string): Promise<Project> {
  const { data } = await client.post<Project>('/projects', { name })
  return data
}

export async function updateProject(slug: string, body: UpdateProjectRequest): Promise<Project> {
  const { data } = await client.put<Project>(`/projects/${encodeURIComponent(slug)}`, body)
  return data
}

export async function deleteProject(slug: string): Promise<void> {
  await client.delete(`/projects/${encodeURIComponent(slug)}`)
}
