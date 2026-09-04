import { http } from '@/lib/http'
import type {
  Container,
  UpdateContainerPayload,
  ContainerConfigFile,
  WriteConfigPayload,
  ContainerInspect,
} from '@/types/api'

export async function syncContainers(hostId: string) {
  const res = await http.post<Container[]>(`/api/v1/hosts/${hostId}/containers/sync`)
  return res.data
}

export async function listContainers(hostId: string) {
  const res = await http.get<Container[]>(`/api/v1/hosts/${hostId}/containers`)
  return res.data
}

export async function updateContainer(hostId: string, id: string, payload: UpdateContainerPayload) {
  const res = await http.put<Container>(`/api/v1/hosts/${hostId}/containers/${id}`, payload)
  return res.data
}

export async function startContainer(hostId: string, id: string) {
  await http.post(`/api/v1/hosts/${hostId}/containers/${id}/start`)
}

export async function stopContainer(hostId: string, id: string) {
  await http.post(`/api/v1/hosts/${hostId}/containers/${id}/stop`)
}

export async function restartContainer(hostId: string, id: string) {
  await http.post(`/api/v1/hosts/${hostId}/containers/${id}/restart`)
}

export async function inspectContainer(hostId: string, id: string) {
  const res = await http.get<ContainerInspect>(`/api/v1/hosts/${hostId}/containers/${id}/inspect`)
  return res.data
}

export async function readConfig(hostId: string, id: string, path: string) {
  const res = await http.get<ContainerConfigFile>(`/api/v1/hosts/${hostId}/containers/${id}/config`, { path })
  return res.data
}

export async function writeConfig(hostId: string, id: string, payload: WriteConfigPayload) {
  await http.put(`/api/v1/hosts/${hostId}/containers/${id}/config`, payload)
}
