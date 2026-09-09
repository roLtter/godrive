import { apiClient, plainApi, toApiError } from './client'
import { clearTokens, getRefreshToken, setTokens } from './authStorage'

export type AuthTokens = {
  access_token: string
  refresh_token?: string
}

export type LoginPayload = {
  email: string
  password: string
}

export type RegisterPayload = {
  email: string
  password: string
}

export function persistAuthTokens(tokens: AuthTokens): void {
  if (!tokens.refresh_token) return
  setTokens(tokens.access_token, tokens.refresh_token)
}

export async function login(payload: LoginPayload): Promise<AuthTokens> {
  const { data } = await apiClient.post<AuthTokens>('/login', payload)
  return data
}

export async function register(payload: RegisterPayload): Promise<void> {
  await apiClient.post('/register', payload)
}

export async function logout(): Promise<void> {
  const refresh = getRefreshToken()
  if (refresh) {
    try {
      await plainApi.post('/logout', { refresh_token: refresh })
    } catch {
      // Session is cleared locally even if the server is unreachable.
    }
  }
  clearTokens()
}

export async function refreshSession(): Promise<AuthTokens | null> {
  const refresh = getRefreshToken()
  if (!refresh) return null
  try {
    const { data } = await plainApi.post<AuthTokens>('/refresh', { refresh_token: refresh })
    if (data.refresh_token) {
      setTokens(data.access_token, data.refresh_token)
    }
    return data
  } catch (error) {
    clearTokens()
    throw error
  }
}

export { toApiError }
