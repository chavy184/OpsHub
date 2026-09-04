import { create } from 'zustand'
import type { ReleaseRecord, ReleaseQueryParams, PageData } from '@/types/api'
import * as api from '@/api/releases'

interface ReleaseState {
  releases: ReleaseRecord[]
  total: number
  loading: boolean
  query: ReleaseQueryParams

  fetchReleases: () => Promise<void>
  setQuery: (q: Partial<ReleaseQueryParams>) => void
  createRelease: (payload: Parameters<typeof api.createRelease>[0]) => Promise<ReleaseRecord>
  executeRelease: (id: string) => Promise<ReleaseRecord>
  rollbackRelease: (id: string) => Promise<ReleaseRecord>
  deleteRelease: (id: string) => Promise<void>
  refreshRelease: (id: string) => Promise<void>
}

export const useReleaseStore = create<ReleaseState>((set, get) => ({
  releases: [],
  total: 0,
  loading: false,
  query: { page: 1, page_size: 20 },

  setQuery: (q) => {
    set((s) => ({ query: { ...s.query, ...q } }))
    get().fetchReleases()
  },

  fetchReleases: async () => {
    set({ loading: true })
    try {
      const data: PageData<ReleaseRecord> = await api.listReleases(get().query)
      set({ releases: data.list ?? [], total: data.total })
    } catch {
      set({ releases: [], total: 0 })
    } finally {
      set({ loading: false })
    }
  },

  createRelease: async (payload) => {
    const record = await api.createRelease(payload)
    get().fetchReleases()
    return record
  },

  executeRelease: async (id) => {
    const record = await api.executeRelease(id)
    get().fetchReleases()
    return record
  },

  rollbackRelease: async (id) => {
    const record = await api.rollbackRelease(id)
    get().fetchReleases()
    return record
  },

  deleteRelease: async (id) => {
    await api.deleteRelease(id)
    get().fetchReleases()
  },

  refreshRelease: async (id) => {
    const updated = await api.getRelease(id)
    set((s) => ({
      releases: s.releases.map((r) => (r.id === id ? updated : r)),
    }))
  },
}))
