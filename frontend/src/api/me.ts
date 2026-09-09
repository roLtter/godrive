import { apiClient } from './client'

export type MeResponse = {
  user_id: string
  email: string
}

export async function fetchMe(): Promise<MeResponse> {
  const { data } = await apiClient.get<MeResponse>('/api/me')
  return data
}
