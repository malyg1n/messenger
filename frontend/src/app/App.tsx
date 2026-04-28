import { useEffect, useState } from "react"
import { AuthForm } from "@/features/auth"
import { ChatPage } from "@/pages/chat-page"
import type { User } from "@/entities/user"
import { STORAGE_USER_KEY } from "@/shared/config/storage"

// App управляет верхнеуровневым состоянием авторизации и маршрутом на страницу чата.
export default function App() {
  const [user, setUser] = useState<User | null>(null)

  // handleLogout очищает локальную сессию и возвращает пользователя на экран входа.
  function handleLogout() {
    localStorage.removeItem(STORAGE_USER_KEY)
    setUser(null)
  }

  // При старте приложения восстанавливаем сохраненного пользователя из localStorage.
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_USER_KEY)
    if (saved) {
      const parsed = JSON.parse(saved)
      setUser(parsed)
    }
  }, [])

  if (!user) {
    return <AuthForm onAuth={setUser} />
  }

  return <ChatPage user={user} onLogout={handleLogout} />
}
