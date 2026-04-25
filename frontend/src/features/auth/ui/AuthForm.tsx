import { useState } from "react"
import api from "@/shared/api/client"
import type { User } from "@/entities/user"

type Props = {
  onAuth: (user: User) => void
}

// AuthForm предоставляет UI для входа и регистрации по username.
export default function AuthForm({ onAuth }: Props) {
  const [username, setUsername] = useState("")
  const [error, setError] = useState("")

  // handleLogin выполняет вход и сохраняет пользователя в localStorage.
  async function handleLogin() {
    try {
      const user = await api.login(username)
      localStorage.setItem("user", JSON.stringify(user))
      onAuth(user)
    } catch {
      setError("user not found")
    }
  }

  // handleRegister создает пользователя и сразу авторизует его в UI.
  async function handleRegister() {
    try {
      const user = await api.register(username)
      localStorage.setItem("user", JSON.stringify(user))
      onAuth(user)
    } catch {
      setError("username taken")
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h1 className="auth-title">Messenger</h1>
        <p className="auth-subtitle">Вход или регистрация</p>

        <input
          className="auth-input"
          placeholder="username"
          value={username}
          onChange={e => setUsername(e.target.value)}
        />

        <div className="auth-actions">
          <button className="btn btn-primary" onClick={handleLogin}>
            Войти
          </button>

          <button className="btn btn-secondary" onClick={handleRegister}>
            Регистрация
          </button>
        </div>

        {error ? <div className="auth-error">{error}</div> : null}
      </div>
    </div>
  )
}
