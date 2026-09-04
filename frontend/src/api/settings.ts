import { http } from '@/lib/http'
import type {
  SystemSetting,
  UpdateSettingPayload,
  BatchUpdateSettingPayload,
} from '@/types/api'

const PREFIX = '/api/v1/settings'

export async function getSettings(category?: string) {
  const params = category ? { category } : undefined
  const res = await http.get<SystemSetting[]>(PREFIX, params)
  return res.data
}

export async function updateSetting(payload: UpdateSettingPayload) {
  const res = await http.put<SystemSetting>(PREFIX, payload)
  return res.data
}

export async function batchUpdateSettings(payload: BatchUpdateSettingPayload) {
  const res = await http.patch<SystemSetting[]>(`${PREFIX}/batch`, payload)
  return res.data
}
