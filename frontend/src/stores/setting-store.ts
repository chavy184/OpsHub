import { create } from 'zustand'
import type { SystemSetting, UpdateSettingPayload } from '@/types/api'
import * as api from '@/api/settings'

interface SettingState {
  settings: SystemSetting[]
  loading: boolean

  fetchSettings: (category?: string) => Promise<void>
  updateSetting: (payload: UpdateSettingPayload) => Promise<void>
  batchUpdate: (items: UpdateSettingPayload[]) => Promise<void>
}

export const useSettingStore = create<SettingState>((set) => ({
  settings: [],
  loading: false,

  fetchSettings: async (category) => {
    set({ loading: true })
    try {
      const data = await api.getSettings(category)
      set({ settings: Array.isArray(data) ? data : [] })
    } catch {
      set({ settings: [] })
    } finally {
      set({ loading: false })
    }
  },

  updateSetting: async (payload) => {
    await api.updateSetting(payload)
    const store = useSettingStore.getState()
    store.fetchSettings()
  },

  batchUpdate: async (items) => {
    await api.batchUpdateSettings({ items })
    const store = useSettingStore.getState()
    store.fetchSettings()
  },
}))
