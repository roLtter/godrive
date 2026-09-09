import axios from 'axios'
import { clearTokens, getAccessToken, getRefreshToken, setTokens } from './authStorage'

type MutableRetryConfig = { _retry?: boolean }

export type ApiError = {
  message: string
  statusCode?: number
}

function resolveApiBaseUrl(): string {
  const fromEnv = import.meta.env.VITE_API_BASE_URL
  if (fromEnv !== undefined && fromEnv !== '') {
    return fromEnv
  }
  if (import.meta.env.DEV) {
    return ''
  }
  return 'http://localhost:8080'
}

export const API_BASE_URL = resolveApiBaseUrl()

/** No auth interceptors — used for refresh to avoid recursion. */
export const plainApi = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
})

function isAuthPublicPath(url: string | undefined): boolean {
  if (!url) return false
  const path = url.split('?')[0] ?? ''
  return path === '/register' || path === '/login' || path === '/refresh' || path === '/logout'
}

function shouldSetJsonContentType(config: { method?: string; data?: unknown }): boolean {
  if (config.data instanceof FormData) return false
  const method = (config.method ?? 'get').toLowerCase()
  if (method === 'get' || method === 'head' || method === 'delete') return false
  return config.data !== undefined && config.data !== null
}

apiClient.interceptors.request.use((config) => {
  const headers = axios.AxiosHeaders.from(config.headers)

  if (shouldSetJsonContentType(config) && !headers.get('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  config.headers = headers

  if (isAuthPublicPath(config.url)) {
    return config
  }

  const access = getAccessToken()
  if (access) {
    headers.set('Authorization', `Bearer ${access}`)
  }

  return config
})

let refreshChain: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const refresh = getRefreshToken()
  if (!refresh) {
    clearTokens()
    return null
  }

  try {
    const { data } = await plainApi.post<{ access_token: string; refresh_token: string }>('/refresh', {
      refresh_token: refresh,
    })
    setTokens(data.access_token, data.refresh_token)
    return data.access_token
  } catch {
    clearTokens()
    return null
  }
}

function enqueueTokenRefresh(): Promise<string | null> {
  if (!refreshChain) {
    refreshChain = refreshAccessToken().finally(() => {
      refreshChain = null
    })
  }
  return refreshChain
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: unknown) => {
    if (!axios.isAxiosError(error)) {
      return Promise.reject(error)
    }

    const original = error.config
    const status = error.response?.status

    const originalWithRetry = original as typeof original & MutableRetryConfig

    if (status !== 401 || !original || originalWithRetry._retry) {
      return Promise.reject(error)
    }

    if (isAuthPublicPath(original.url)) {
      return Promise.reject(error)
    }

    originalWithRetry._retry = true
    const newAccess = await enqueueTokenRefresh()

    if (!newAccess) {
      return Promise.reject(error)
    }

    const retryHeaders = axios.AxiosHeaders.from(original.headers)
    retryHeaders.set('Authorization', `Bearer ${newAccess}`)
    original.headers = retryHeaders
    return apiClient.request(original)
  },
)

export function toApiError(error: unknown): ApiError {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { message?: string; error?: string } | undefined
    return {
      message: data?.message ?? data?.error ?? error.message,
      statusCode: error.response?.status,
    }
  }

  return {
    message: 'Unexpected error',
  }
}
