import { http } from '@/lib/http'

export interface ImageSyncRecord {
  id: string
  source_host_id: string
  source_host_name: string
  target_host_id: string
  target_host_name: string
  image: string
  mode: string
  status: string
  source_image_id: string
  target_image_id: string
  image_size: number
  summary: string
  error: string
  started_at: string
  finished_at: string | null
  duration: number
  created_at: string
}

export interface HostImage {
  repository: string
  tag: string
  image_id: string
  created_at: string
  size: string
  name: string
}

export interface ExecuteImageSyncPayload {
  source_host_id: string
  target_host_id: string
  image: string
  mode: string
}

export async function executeImageSync(payload: ExecuteImageSyncPayload) {
  const res = await http.post<{ message: string; record_id: string }>('/api/v1/hosts/image-sync/execute', payload)
  return res.data
}

export async function listHostImages(hostId: string) {
  const res = await http.get<HostImage[]>(`/api/v1/hosts/${hostId}/images`)
  return res.data
}

export async function listImageSyncRecords(params?: {
  source_host_id?: string
  target_host_id?: string
  status?: string
  page?: number
  page_size?: number
}) {
  const res = await http.get<{ list: ImageSyncRecord[]; total: number }>('/api/v1/hosts/image-sync/records', params)
  return res.data
}

export async function getImageSyncRecord(id: string) {
  const res = await http.get<ImageSyncRecord>(`/api/v1/hosts/image-sync/records/${id}`)
  return res.data
}
