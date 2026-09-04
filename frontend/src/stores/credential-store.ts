import { create } from 'zustand'
import type { Credential, CredentialQueryParams, PageData } from '@/types/api'
import * as api from '@/api/credentials'

interface CredentialState {
  credentials: Credential[]
  total: number
  loading: boolean
  query: CredentialQueryParams

  fetchCredentials: () => Promise<void>
  setQuery: (q: Partial<CredentialQueryParams>) => void
  createCredential: (payload: Parameters<typeof api.createCredential>[0]) => Promise<Credential>
  updateCredential: (id: string, payload: Parameters<typeof api.updateCredential>[1]) => Promise<void>
  deleteCredential: (id: string) => Promise<void>
}

export const useCredentialStore = create<CredentialState>((set, get) => ({
  credentials: [],
  total: 0,
  loading: false,
  query: { page: 1, page_size: 20 },

  setQuery: (q) => {
    set((s) => ({ query: { ...s.query, ...q } }))
    get().fetchCredentials()
  },

  fetchCredentials: async () => {
    set({ loading: true })
    try {
      const data: PageData<Credential> = await api.listCredentials(get().query)
      set({ credentials: data.list ?? [], total: data.total })
    } catch {
      set({ credentials: [], total: 0 })
    } finally {
      set({ loading: false })
    }
  },

  createCredential: async (payload) => {
    const cred = await api.createCredential(payload)
    get().fetchCredentials()
    return cred
  },

  updateCredential: async (id, payload) => {
    await api.updateCredential(id, payload)
    get().fetchCredentials()
  },

  deleteCredential: async (id) => {
    await api.deleteCredential(id)
    get().fetchCredentials()
  },
}))
