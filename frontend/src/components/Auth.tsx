import { useState } from "react"
import api from "../api/client"
import { User } from "../types"

type Props = {
  onAuth: (user: User) => void
}

export default function Auth({ onAuth }: Props) {
  const [username, setUsername] = useState("")
  const [error, setError] = useState("")

  async function register() {
    try {
      const user = await api.register(username)
      onAuth(user)
    } catch (e) {
      setError("username already taken")
    }
  }

  return (
    <div>
      <h2>Register</h2>

      <input
        placeholder="username"
        value={username}
        onChange={e => setUsername(e.target.value)}
      />

      <button onClick={register}>
        enter
      </button>

      <div>{error}</div>
    </div>
  )
}