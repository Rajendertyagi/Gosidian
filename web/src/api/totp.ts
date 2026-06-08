import client from './client'

export interface AuthConfig {
  totp: boolean
  ldap: boolean
}

/** GET /api/v1/auth-config — public. Drives whether the LoginView renders the
 *  TOTP field (and, in Phase 3, the LDAP option). Booleans only. */
export async function getAuthConfig(): Promise<AuthConfig> {
  const { data } = await client.get<AuthConfig>('/auth-config')
  return data
}

export interface TotpEnrollData {
  secret: string
  otpauth_uri: string
}

/** Start enrolment: returns a fresh secret + otpauth URI (not yet active). */
export async function enrollTOTP(): Promise<TotpEnrollData> {
  const { data } = await client.post<TotpEnrollData>('/totp/enroll', {})
  return data
}

/** Confirm a code against the candidate secret to activate it. */
export async function confirmTOTP(secret: string, code: string): Promise<void> {
  await client.post('/totp/confirm', { secret, code })
}

/** Remove the current user's TOTP secret (403 if the policy requires it). */
export async function disenrollTOTP(): Promise<void> {
  await client.delete('/totp')
}
