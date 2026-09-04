import { http } from '@/lib/http'
import type {
  ReleaseRecord,
  ReleaseStepLog,
  CreateReleasePayload,
  ReleaseQueryParams,
  PageData,
} from '@/types/api'

const PREFIX = '/api/v1/releases'

/** 发布记录列表 */
export async function listReleases(params: ReleaseQueryParams) {
  const res = await http.get<PageData<ReleaseRecord>>(PREFIX, params as unknown as Record<string, string | number | undefined>)
  return res.data
}

/** 发布详情 */
export async function getRelease(id: string) {
  const res = await http.get<ReleaseRecord>(`${PREFIX}/${id}`)
  return res.data
}

/** 创建发布单 */
export async function createRelease(payload: CreateReleasePayload) {
  const res = await http.post<ReleaseRecord>(PREFIX, payload, {
    'Idempotency-Key': payload.idempotency_key || crypto.randomUUID(),
  })
  return res.data
}

/** 执行发布 */
export async function executeRelease(id: string) {
  const res = await http.post<ReleaseRecord>(`${PREFIX}/${id}/execute`)
  return res.data
}

/** 回滚发布 */
export async function rollbackRelease(id: string) {
  const res = await http.post<ReleaseRecord>(`${PREFIX}/${id}/rollback`)
  return res.data
}

/** 删除发布记录 */
export async function deleteRelease(id: string) {
  await http.del(`${PREFIX}/${id}`)
}

/** 发布步骤日志 */
export async function getReleaseSteps(id: string) {
  const res = await http.get<ReleaseStepLog[]>(`${PREFIX}/${id}/steps`)
  return res.data
}
