import { http } from '@/lib/http'

export interface SearchHit {
  type: 'service' | 'host'
  id: string
  title: string
  subtitle: string
  url: string
}

export interface SearchResult {
  query: string
  items: SearchHit[]
}

export const searchApi = {
  search: (q: string) => http.get<SearchResult>('/api/v1/search', { q }),
}
