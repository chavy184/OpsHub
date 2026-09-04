import { http } from '@/lib/http'
import type {
  Service,
  CreateServicePayload,
  UpdateServicePayload,
  ServiceQueryParams,
  ServiceEnv,
  UpdateEnvPayload,
  CreateEnvPayload,
  PageData,
} from '@/types/api'

const PREFIX = '/api/v1/services'

/** 服务列表 */
export async function listServices(params: ServiceQueryParams) {
  const res = await http.get<PageData<Service>>(PREFIX, params as unknown as Record<string, string | number | undefined>)
  return res.data
}

/** 服务详情 */
export async function getService(id: string) {
  const res = await http.get<Service>(`${PREFIX}/${id}`)
  return res.data
}

/** 创建服务 */
export async function createService(payload: CreateServicePayload) {
  const res = await http.post<Service>(PREFIX, payload)
  return res.data
}

/** 更新服务 */
export async function updateService(id: string, payload: UpdateServicePayload) {
  const res = await http.put<Service>(`${PREFIX}/${id}`, payload)
  return res.data
}

/** 删除服务 */
export async function deleteService(id: string) {
  await http.del(`${PREFIX}/${id}`)
}

/** 环境列表 */
export async function listEnvs(serviceId: string) {
  const res = await http.get<ServiceEnv[]>(`${PREFIX}/${serviceId}/envs`)
  return res.data
}

/** 创建环境 */
export async function createEnv(serviceId: string, payload: CreateEnvPayload) {
  const res = await http.post<ServiceEnv>(`${PREFIX}/${serviceId}/envs`, payload)
  return res.data
}

/** 更新环境 */
export async function updateEnv(serviceId: string, envId: string, payload: UpdateEnvPayload) {
  const res = await http.put<ServiceEnv>(`${PREFIX}/${serviceId}/envs/${envId}`, payload)
  return res.data
}

/** 删除环境 */
export async function deleteEnv(serviceId: string, envId: string) {
  await http.del(`${PREFIX}/${serviceId}/envs/${envId}`)
}
