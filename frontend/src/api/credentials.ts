import { http } from '@/lib/http'
import type {
  Credential,
  CreateCredentialPayload,
  UpdateCredentialPayload,
  CredentialQueryParams,
  PageData,
} from '@/types/api'

const PREFIX = '/api/v1/credentials'

export async function listCredentials(params: CredentialQueryParams) {
  const res = await http.get<PageData<Credential>>(PREFIX, params as unknown as Record<string, string | number | undefined>)
  return res.data
}

export async function getCredential(id: string) {
  const res = await http.get<Credential>(`${PREFIX}/${id}`)
  return res.data
}

export async function createCredential(payload: CreateCredentialPayload) {
  const res = await http.post<Credential>(PREFIX, payload)
  return res.data
}

export async function updateCredential(id: string, payload: UpdateCredentialPayload) {
  const res = await http.put<Credential>(`${PREFIX}/${id}`, payload)
  return res.data
}

export async function deleteCredential(id: string) {
  await http.del(`${PREFIX}/${id}`)
}
