import { http } from '@/lib/http'
import type {
  Host,
  CreateHostPayload,
  UpdateHostPayload,
  HostQueryParams,
  TestConnectionResult,
  PageData,
} from '@/types/api'

const PREFIX = '/api/v1/hosts'

export async function listHosts(params: HostQueryParams) {
  const res = await http.get<PageData<Host>>(PREFIX, params as unknown as Record<string, string | number | undefined>)
  return res.data
}

export async function getHost(id: string) {
  const res = await http.get<Host>(`${PREFIX}/${id}`)
  return res.data
}

export async function createHost(payload: CreateHostPayload) {
  const res = await http.post<Host>(PREFIX, payload)
  return res.data
}

export async function updateHost(id: string, payload: UpdateHostPayload) {
  const res = await http.put<Host>(`${PREFIX}/${id}`, payload)
  return res.data
}

export async function deleteHost(id: string) {
  await http.del(`${PREFIX}/${id}`)
}

export async function testConnection(id: string) {
  const res = await http.post<TestConnectionResult>(`${PREFIX}/${id}/test-connection`)
  return res.data
}
