import { http } from '@/lib/http'

export interface DocCategory {
  name: string
  path: string
}

export interface FileNode {
  name: string
  path: string
  is_dir: boolean
  size: number
  mod_time: string
  children?: FileNode[]
}

export interface FileContent {
  path: string
  content: string
  size: number
}

// --- 文档类型管理 ---

export async function listDocTypes(): Promise<DocCategory[]> {
  const res = await http.get<DocCategory[]>('/api/v1/documents/types')
  return res.data ?? []
}

export async function createDocType(name: string): Promise<DocCategory> {
  const res = await http.post<DocCategory>('/api/v1/documents/types', { name })
  return res.data!
}

export async function deleteDocType(name: string): Promise<void> {
  await http.del(`/api/v1/documents/types/${encodeURIComponent(name)}`)
}

// --- 文件操作 ---

export async function getDocumentTree(type: string): Promise<FileNode[]> {
  const res = await http.get<FileNode[]>(`/api/v1/documents/tree?type=${encodeURIComponent(type)}`)
  return res.data ?? []
}

export async function uploadDocument(
  type: string,
  path: string,
  file: File,
  overwrite = false
): Promise<FileNode> {
  const formData = new FormData()
  formData.append('type', type)
  formData.append('path', path)
  formData.append('file', file)
  formData.append('overwrite', overwrite ? 'true' : 'false')

  const res = await http.post<FileNode>('/api/v1/documents/upload', formData)
  return res.data!
}

export async function createFolder(type: string, path: string): Promise<FileNode> {
  const res = await http.post<FileNode>('/api/v1/documents/mkdir', { type, path })
  return res.data!
}

export async function deleteDocument(type: string, path: string): Promise<void> {
  await http.del(`/api/v1/documents?type=${encodeURIComponent(type)}&path=${encodeURIComponent(path)}`)
}

export async function getDocumentContent(type: string, path: string): Promise<FileContent> {
  const res = await http.get<FileContent>(
    `/api/v1/documents/content?type=${encodeURIComponent(type)}&path=${encodeURIComponent(path)}`
  )
  return res.data!
}

export function getDownloadUrl(type: string, path: string): string {
  return `/api/v1/documents/download?type=${encodeURIComponent(type)}&path=${encodeURIComponent(path)}`
}
