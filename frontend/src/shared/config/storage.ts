export const STORAGE_USER_KEY = "messenger:user"

type StoredUser = {
  token?: string
}

// getStoredUser безопасно читает пользователя из localStorage.
export function getStoredUser(): StoredUser | null {
  const raw = localStorage.getItem(STORAGE_USER_KEY)
  if (!raw) return null

  try {
    return JSON.parse(raw) as StoredUser
  } catch {
    return null
  }
}

// getAuthToken возвращает токен текущей сессии.
export function getAuthToken(): string {
  return getStoredUser()?.token ?? ""
}
