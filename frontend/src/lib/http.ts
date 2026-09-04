import type { ApiResponse } from '@/types/api'
import { clearAuth, getToken } from '@/lib/auth'

// 开发环境通过 vite proxy 转发 /api 到后端，无需额外前缀
const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

class HttpClient {
  private baseURL: string

  constructor(baseURL: string) {
    this.baseURL = baseURL
  }

  private async request<T>(
    method: string,
    path: string,
    options?: {
      body?: unknown
      params?: Record<string, string | number | undefined>
      headers?: Record<string, string>
    }
  ): Promise<ApiResponse<T>> {
    const base = this.baseURL || window.location.origin
    const url = new URL(`${base}${path}`)

    if (options?.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        if (value !== undefined && value !== '') {
          url.searchParams.set(key, String(value))
        }
      })
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options?.headers,
    }

    // JWT token
    const token = getToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(url.toString(), {
      method,
      headers,
      body: options?.body ? JSON.stringify(options.body) : undefined,
    })

    if (response.status === 401 && path !== '/api/v1/auth/login') {
      clearAuth()
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }

    if (!response.ok) {
      const errorBody = await response.json().catch(() => null)
      throw new ApiError(
        response.status,
        errorBody?.msg || `HTTP ${response.status}`,
        errorBody?.code
      )
    }

    return response.json()
  }

  async get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<ApiResponse<T>> {
    return this.request<T>('GET', path, { params })
  }

  async post<T>(path: string, body?: unknown, headers?: Record<string, string>): Promise<ApiResponse<T>> {
    return this.request<T>('POST', path, { body, headers })
  }

  async put<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return this.request<T>('PUT', path, { body })
  }

  async del<T>(path: string): Promise<ApiResponse<T>> {
    return this.request<T>('DELETE', path)
  }

  async patch<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return this.request<T>('PATCH', path, { body })
  }
}

export class ApiError extends Error {
  status: number
  code?: number

  constructor(status: number, message: string, code?: number) {
    super(message)
    this.status = status
    this.code = code
  }
}

export const http = new HttpClient(BASE_URL)
