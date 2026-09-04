import { create } from 'zustand'
import type { Service, ServiceQueryParams, PageData, ServiceEnv } from '@/types/api'
import * as api from '@/api/services'

interface ServiceState {
  // 列表
  services: Service[]
  total: number
  loading: boolean
  query: ServiceQueryParams

  // 详情
  currentService: Service | null
  envs: ServiceEnv[]
  detailLoading: boolean

  // 操作
  fetchServices: () => Promise<void>
  setQuery: (q: Partial<ServiceQueryParams>) => void
  fetchServiceDetail: (id: string) => Promise<void>
  fetchEnvs: (serviceId: string) => Promise<void>
  createService: (payload: Parameters<typeof api.createService>[0]) => Promise<Service>
  updateService: (id: string, payload: Parameters<typeof api.updateService>[1]) => Promise<void>
  deleteService: (id: string) => Promise<void>
  createEnv: (serviceId: string, payload: Parameters<typeof api.createEnv>[1]) => Promise<void>
  updateEnv: (serviceId: string, envId: string, payload: Parameters<typeof api.updateEnv>[2]) => Promise<void>
  deleteEnv: (serviceId: string, envId: string) => Promise<void>
}

export const useServiceStore = create<ServiceState>((set, get) => ({
  services: [],
  total: 0,
  loading: false,
  query: { page: 1, page_size: 20 },

  currentService: null,
  envs: [],
  detailLoading: false,

  setQuery: (q) => {
    set((s) => ({ query: { ...s.query, ...q } }))
    get().fetchServices()
  },

  fetchServices: async () => {
    set({ loading: true })
    try {
      const data: PageData<Service> = await api.listServices(get().query)
      set({ services: data.list ?? [], total: data.total })
    } catch {
      set({ services: [], total: 0 })
    } finally {
      set({ loading: false })
    }
  },

  fetchServiceDetail: async (id) => {
    set({ detailLoading: true })
    try {
      const svc = await api.getService(id)
      set({ currentService: svc })
    } finally {
      set({ detailLoading: false })
    }
  },

  fetchEnvs: async (serviceId) => {
    try {
      const envs = await api.listEnvs(serviceId)
      set({ envs: Array.isArray(envs) ? envs : [] })
    } catch {
      set({ envs: [] })
    }
  },

  createService: async (payload) => {
    const svc = await api.createService(payload)
    get().fetchServices()
    return svc
  },

  updateService: async (id, payload) => {
    await api.updateService(id, payload)
    get().fetchServices()
  },

  deleteService: async (id) => {
    await api.deleteService(id)
    get().fetchServices()
  },

  createEnv: async (serviceId, payload) => {
    await api.createEnv(serviceId, payload)
    get().fetchEnvs(serviceId)
  },

  updateEnv: async (serviceId, envId, payload) => {
    await api.updateEnv(serviceId, envId, payload)
    get().fetchEnvs(serviceId)
  },

  deleteEnv: async (serviceId, envId) => {
    await api.deleteEnv(serviceId, envId)
    get().fetchEnvs(serviceId)
  },
}))
