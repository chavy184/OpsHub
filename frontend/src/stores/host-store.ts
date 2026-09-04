import { create } from 'zustand'
import type { Host, HostQueryParams, PageData, TestConnectionResult } from '@/types/api'
import * as api from '@/api/hosts'

interface HostState {
  hosts: Host[]
  total: number
  loading: boolean
  query: HostQueryParams

  fetchHosts: () => Promise<void>
  setQuery: (q: Partial<HostQueryParams>) => void
  createHost: (payload: Parameters<typeof api.createHost>[0]) => Promise<Host>
  updateHost: (id: string, payload: Parameters<typeof api.updateHost>[1]) => Promise<void>
  deleteHost: (id: string) => Promise<void>
  testConnection: (id: string) => Promise<TestConnectionResult>
}

export const useHostStore = create<HostState>((set, get) => ({
  hosts: [],
  total: 0,
  loading: false,
  query: { page: 1, page_size: 20 },

  setQuery: (q) => {
    set((s) => ({ query: { ...s.query, ...q } }))
    get().fetchHosts()
  },

  fetchHosts: async () => {
    set({ loading: true })
    try {
      const data: PageData<Host> = await api.listHosts(get().query)
      set({ hosts: data.list ?? [], total: data.total })
    } catch {
      set({ hosts: [], total: 0 })
    } finally {
      set({ loading: false })
    }
  },

  createHost: async (payload) => {
    const host = await api.createHost(payload)
    get().fetchHosts()
    return host
  },

  updateHost: async (id, payload) => {
    await api.updateHost(id, payload)
    get().fetchHosts()
  },

  deleteHost: async (id) => {
    await api.deleteHost(id)
    get().fetchHosts()
  },

  testConnection: async (id) => {
    return api.testConnection(id)
  },
}))
