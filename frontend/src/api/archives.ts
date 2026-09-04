import { getToken, clearAuth } from '@/lib/auth'

export interface ArchiveNode {
  name: string
  path: string
  kind: 'file' | 'dir'
  section: string
  size: number
  mode?: string
  content?: string
  children?: ArchiveNode[]
}

export interface ArchiveAnalysis {
  filename: string
  type: string
  size: number
  debian_binary?: string
  package_info?: Record<string, string>
  files: ArchiveNode[]
  warnings?: string[]
}

export async function analyzeArchive(file: File): Promise<ArchiveAnalysis> {
  const formData = new FormData()
  formData.append('file', file)

  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch('/api/v1/archives/analyze', {
    method: 'POST',
    headers,
    body: formData,
  })

  if (response.status === 401) {
    clearAuth()
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
  }

  if (!response.ok) {
    const body = await response.json().catch(() => null)
    throw new Error(body?.msg || `HTTP ${response.status}`)
  }

  const body = await response.json()
  return body.data
}
