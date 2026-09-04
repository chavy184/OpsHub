import { http } from '@/lib/http'
import type { LoginPayload, LoginResult } from '@/types/api'

export async function login(payload: LoginPayload) {
  const res = await http.post<LoginResult>('/api/v1/auth/login', payload)
  return res.data
}

export async function getCurrentUser() {
  const res = await http.get<{ username: string }>('/api/v1/auth/me')
  return res.data
}
