import { useState } from "react"
import api from "../api/client"
import { User } from "../types"

type Props = {
	onAuth: (user: User) => void
}

export default function Auth({ onAuth }: Props) {

	const [username, setUsername] = useState("")

	const [error, setError] = useState("")

	async function handleLogin() {

		try {

			const user = await api.login(username)

			localStorage.setItem(
				"user",
				JSON.stringify(user)
			)

			onAuth(user)

		} catch {

			setError("user not found")

		}

	}

	async function handleRegister() {

		try {

			const user = await api.register(username)

			localStorage.setItem(
				"user",
				JSON.stringify(user)
			)

			onAuth(user)

		} catch {

			setError("username taken")

		}

	}

	return (

		<div>

			<h2>Login</h2>

			<input

				placeholder="username"

				value={username}

				onChange={e => setUsername(e.target.value)}

			/>

			<div>

				<button onClick={handleLogin}>
					login
				</button>

				<button onClick={handleRegister}>
					register
				</button>

			</div>

			<div>{error}</div>

		</div>

	)

}