/**
 * Pinia auth store. Holds the SPA Bearer token + the redacted user
 * view returned by /api/v1/login. Persisted via
 * pinia-plugin-persistedstate so a refresh keeps the user logged in.
 *
 * The actions deliberately do NOT use the `client` axios instance —
 * client.ts imports this store for the request interceptor, and a
 * back-import would create a cycle. Login/refresh use a tiny fetch
 * helper inline; the rest of the codebase uses the configured axios
 * client.
 */
import { defineStore } from 'pinia'

export type Role = 'owner' | 'member' | 'guest'

export interface User {
  id: string
  username: string
  role: Role
  totp_enrolled?: boolean
}

interface AuthState {
  token: string
  expiresAt: string
  hardExpiry: string
  user: User | null
  /** Set when login reports the effective policy mandates TOTP but no secret
   *  is enrolled yet — the AppShell forces the enrolment interstitial. */
  enrollmentRequired: boolean
}

interface LoginResponse {
  token: string
  expires_at: string
  hard_expiry: string
  user: User
  totp_enrollment_required?: boolean
}

interface RefreshResponse {
  token: string
  expires_at: string
  hard_expiry: string
}

interface ErrorBody {
  error?: { code?: string; message?: string }
}

async function postJson<T>(path: string, body: object, token?: string): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(`/api/v1${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as ErrorBody
    throw new Error(data.error?.message ?? `HTTP ${res.status}`)
  }
  return (await res.json()) as T
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: '',
    expiresAt: '',
    hardExpiry: '',
    user: null,
    enrollmentRequired: false,
  }),

  getters: {
    isAuthenticated: (s) => Boolean(s.token && s.user),
    isOwner: (s) => s.user?.role === 'owner',
    isGuest: (s) => s.user?.role === 'guest',
    /** owner or member — may create/edit/delete. Guests are read-only. */
    canWrite: (s) => s.user?.role === 'owner' || s.user?.role === 'member',
    username: (s) => s.user?.username ?? '',
  },

  actions: {
    async login(username: string, password: string, totp?: string) {
      const body: { username: string; password: string; totp?: string } = { username, password }
      if (totp) body.totp = totp
      const data = await postJson<LoginResponse>('/login', body)
      this.token = data.token
      this.expiresAt = data.expires_at
      this.hardExpiry = data.hard_expiry
      this.user = data.user
      this.enrollmentRequired = Boolean(data.totp_enrollment_required)
    },

    clearEnrollment() {
      this.enrollmentRequired = false
    },

    setEnrolled(v: boolean) {
      if (this.user) this.user.totp_enrolled = v
    },

    async refresh() {
      if (!this.token) return
      const data = await postJson<RefreshResponse>('/refresh', {}, this.token)
      this.expiresAt = data.expires_at
      this.hardExpiry = data.hard_expiry
    },

    async logout() {
      if (this.token) {
        // Best-effort: a network failure on logout still clears
        // local state so the SPA never deadlocks an unrecoverable
        // session.
        await postJson<unknown>('/logout', {}, this.token).catch(() => {})
      }
      this.clear()
    },

    clear() {
      this.token = ''
      this.expiresAt = ''
      this.hardExpiry = ''
      this.user = null
      this.enrollmentRequired = false
    },
  },

  persist: {
    key: 'gosidian.auth',
    storage: localStorage,
    paths: ['token', 'expiresAt', 'hardExpiry', 'user', 'enrollmentRequired'],
  },
})
