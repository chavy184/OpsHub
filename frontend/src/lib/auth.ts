const TOKEN_KEY = 'token'
const USERNAME_KEY = 'username'
const EXPIRES_AT_KEY = 'token_expires_at'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function getUsername() {
  return localStorage.getItem(USERNAME_KEY) || 'admin'
}

export function isAuthenticated() {
  const token = getToken()
  const expiresAt = localStorage.getItem(EXPIRES_AT_KEY)
  if (!token || !expiresAt) return false
  return Date.parse(expiresAt) > Date.now()
}

export function saveAuth(token: string, username: string, expiresAt: string) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USERNAME_KEY, username)
  localStorage.setItem(EXPIRES_AT_KEY, expiresAt)
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USERNAME_KEY)
  localStorage.removeItem(EXPIRES_AT_KEY)
}
