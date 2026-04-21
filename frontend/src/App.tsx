import { useEffect, useState } from "react"
import Auth from "./components/Auth"
import Chat from "./components/Chat"
import { User } from "./types"

export default function App() {

	const [user, setUser] = useState<User | null>(null)

	useEffect(() => {

		const saved = localStorage.getItem("user")

		if (saved) {

			setUser(JSON.parse(saved))

		}

	}, [])

	if (!user) {

		return <Auth onAuth={setUser} />

	}

	return <Chat user={user} />

}