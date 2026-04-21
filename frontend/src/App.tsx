import { useState } from "react"
import Auth from "./components/Auth"
import Chat from "./components/Chat"
import { User } from "./types"

export default function App() {
  const [user, setUser] = useState<User | null>(null)

  if (!user) {
    return <Auth onAuth={setUser} />
  }

  return <Chat user={user} />
}