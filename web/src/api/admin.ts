import client from './client'

export interface MCPToken {
  id: string
  name: string
  project?: string
  scopes: string[]
  owner_user_id?: string
  created_at: string
  expires_at?: string
  expired?: boolean
}

export interface MCPTokenCreated {
  token: string
  record: MCPToken
  usage_hint: string
}

export interface CreateMCPTokenRequest {
  name: string
  project?: string
  scopes: string[]
  ttl_ms?: number
}

export interface SpaToken {
  id: string
  user_id: string
  user_agent?: string
  issued_at: string
  expires_at: string
  hard_expiry: string
  last_seen_at: string
}

export interface AdminUser {
  id: string
  username: string
  role: string
  created_at: string
  disabled_at?: string
}

export interface Invite {
  token: string
  created_by: string
  created_at: string
  expires_at: string
  consumed_by?: string
  consumed_at?: string
  pending: boolean
}

export interface AuditEntry {
  ts: string
  source: string
  token?: string
  actor?: string
  user_id?: string
  action: string
  path?: string
  to?: string
  size?: number
}

export interface AuditQuery {
  actor?: string
  user_id?: string
  action?: string
  source?: string
  path_prefix?: string
  since?: string
  until?: string
  limit?: number
}

interface ListResp<T> {
  items: T[]
  total: number
  limit?: number
}

// MCP tokens

export async function listMCPTokens(): Promise<MCPToken[]> {
  const { data } = await client.get<ListResp<MCPToken>>('/admin/tokens')
  return data.items
}

export async function createMCPToken(body: CreateMCPTokenRequest): Promise<MCPTokenCreated> {
  const { data } = await client.post<MCPTokenCreated>('/admin/tokens', body)
  return data
}

export async function revokeMCPToken(id: string): Promise<void> {
  await client.delete(`/admin/tokens/${encodeURIComponent(id)}`)
}

// SPA tokens

export async function listSpaTokens(): Promise<SpaToken[]> {
  const { data } = await client.get<ListResp<SpaToken>>('/admin/spa-tokens')
  return data.items
}

export async function revokeSpaToken(id: string): Promise<void> {
  await client.delete(`/admin/spa-tokens/${encodeURIComponent(id)}`)
}

// Users

export async function listUsers(): Promise<AdminUser[]> {
  const { data } = await client.get<ListResp<AdminUser>>('/admin/users')
  return data.items
}

export async function disableUser(id: string): Promise<void> {
  await client.delete(`/admin/users/${encodeURIComponent(id)}`)
}

// Invites

export async function listInvites(): Promise<Invite[]> {
  const { data } = await client.get<ListResp<Invite>>('/admin/invites')
  return data.items
}

export async function createInvite(ttlMs?: number): Promise<Invite> {
  const body = ttlMs ? { ttl_ms: ttlMs } : {}
  const { data } = await client.post<Invite>('/admin/invites', body)
  return data
}

export async function deleteInvite(token: string): Promise<void> {
  await client.delete(`/admin/invites/${encodeURIComponent(token)}`)
}

// Audit

export async function tailAudit(query: AuditQuery = {}): Promise<AuditEntry[]> {
  const params: Record<string, string | number> = {}
  if (query.actor) params.actor = query.actor
  if (query.user_id) params.user_id = query.user_id
  if (query.action) params.action = query.action
  if (query.source) params.source = query.source
  if (query.path_prefix) params.path_prefix = query.path_prefix
  if (query.since) params.since = query.since
  if (query.until) params.until = query.until
  if (query.limit) params.limit = query.limit
  const { data } = await client.get<ListResp<AuditEntry>>('/admin/audit', { params })
  return data.items
}
